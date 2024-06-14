package cpu

/* IBM 370 System instructions

   Copyright (c) 2024, Richard Cornwell

   Permission is hereby granted, free of charge, to any person obtaining a
   copy of this software and associated documentation files (the "Software"),
   to deal in the Software without restriction, including without limitation
   the rights to use, copy, modify, merge, publish, distribute, sublicense,
   and/or sell copies of the Software, and to permit persons to whom the
   Software is furnished to do so, subject to the following conditions:

   The above copyright notice and this permission notice shall be included in
   all copies or substantial portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL
   RICHARD CORNWELL BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
   IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

*/

import (
	"github.com/rcornwell/S370/internal/memory"
	"github.com/rcornwell/S370/internal/sys_channel"
)

// Set storage key
func (cpu *CPU) op_ssk(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		// Try to do quick SSK
		//if (QVMA && vma_stssk(R1, addr1))
		//    break;
		return IRC_PRIV
	} else if (step.address1 & 0x0f) != 0 {
		return IRC_SPEC
	} else if memory.CheckAddr(step.address1) {
		return IRC_ADDR
	}
	t := uint8(step.src1 & 0xf8)
	memory.PutKey(step.address1, t)
	return 0
}

// Insert storage Key into register
func (cpu *CPU) op_isk(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		//  if (QVMA && vma_stisk(src1, addr1))
		// break;
		return IRC_PRIV
	} else if (step.address1 & 0x0f) != 0 {
		return IRC_SPEC
	} else if memory.CheckAddr(step.address1) {
		return IRC_ADDR
	}
	t := memory.GetKey(step.address1)
	cpu.regs[step.R1] &= 0xffffff00
	if cpu.ec_mode {
		cpu.regs[step.R1] |= uint32(t) & 0xfe
	} else {
		cpu.regs[step.R1] |= uint32(t) & 0xf8
	}
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Supervisor call
func (cpu *CPU) op_svc(step *stepInfo) uint16 {
	//  if ((flags & PROBLEM) != 0 && \
	//  (cpu_unit[0].flags & (FEAT_370|FEAT_VMA)) == (FEAT_370|FEAT_VMA) && \
	//  (cregs[6] & 0x88000000) == MSIGN && vma_stsvc(reg))
	//  break
	irqaddr := cpu.storePSW(OSPSW, uint16(step.R1))
	mem_cycle++
	src1 := memory.GetMemory(irqaddr)
	mem_cycle++
	src2 := memory.GetMemory(irqaddr + 0x4)
	cpu.lpsw(src1, src2)
	return 0
}

// Set system mask
func (cpu *CPU) op_ssm(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		//      sim_debug(DEBUG_VMA, &cpu_dev, "SSM  CR6 %08x\n", cregs[6]);
		//      if (QVMA && vma_ssm(addr1))
		//          break;
		return IRC_PRIV
	} else if (cpu.cregs[0] & 0x40000000) != 0 {
		return IRC_SPOP
	} else {
		var t uint32
		var error uint16
		if t, error = cpu.readByte(step.address1); error != 0 {
			return error
		}

		cpu.ext_en = (t & 0x01) != 0
		if cpu.ec_mode {
			if (t & 0x02) != 0 {
				cpu.irq_en = true
				cpu.sysmask = uint16(cpu.cregs[2] >> 16)
			} else {
				cpu.irq_en = false
				cpu.sysmask = 0
			}
			cpu.page_en = (t & 0x04) != 0
			cpu.per_en = (t & 0x08) != 0
			if (t & 0xb8) != 0 {
				return IRC_SPEC
			}
		} else {
			cpu.sysmask = uint16(t&0xfc) << 8
			if (t & 0x2) != 0 {
				cpu.sysmask |= uint16((cpu.cregs[2] >> 16) & 0x3ff)
			}
			cpu.irq_en = cpu.sysmask != 0
			cpu.page_en = false
		}
	}
	sys_channel.Irq_pending = true
	return 0
}

// Load processor status word
func (cpu *CPU) op_lpsw(step *stepInfo) uint16 {
	var src1, src2 uint32
	var error uint16

	if (cpu.flags & PROBLEM) != 0 {
		//       if (QVMA && vma_lpsw(addr1))
		//           break;
		return IRC_PRIV
	} else if (step.address1 & 0x7) != 0 {
		return IRC_SPEC
	} else {
		if src1, error = cpu.readFull(step.address1); error != 0 {
			return error
		}
		if src2, error = cpu.readFull(step.address1 + 4); error != 0 {
			return error
		}
		cpu.lpsw(src1, src2)
	}
	return 0
}

// Compare and swap
func (cpu *CPU) op_cs(step *stepInfo) uint16 {
	var error uint16
	var orig uint32
	var src uint32

	if (step.address1 & 0x3) != 0 {
		return IRC_SPEC
	}
	if orig, error = cpu.readFull(step.address1); error != 0 {
		return error
	}
	src = cpu.regs[step.R2]
	if cpu.regs[step.R1] == orig {
		if error = cpu.writeFull(step.address1, src); error != 0 {
			return error
		}
		cpu.cc = 0
	} else {
		cpu.regs[step.R1] = orig
		cpu.per_mod |= 1 << uint32(step.R1)
		cpu.cc = 1
	}
	return 0
}

// Compare and swap double
func (cpu *CPU) op_cds(step *stepInfo) uint16 {
	var error uint16
	var origl, origh uint32
	var srcl, srch uint32

	if (step.address1&0x7) != 0 || (step.R1&1) != 0 || (step.R2&1) != 0 {
		return IRC_SPEC
	}
	if origl, error = cpu.readFull(step.address1); error != 0 {
		return error
	}
	if origh, error = cpu.readFull(step.address1 + 4); error != 0 {
		return error
	}
	srcl = cpu.regs[step.R2]
	srch = cpu.regs[step.R2|1]
	if origl == srcl && origh == srch {
		if error = cpu.writeFull(step.address1, srcl); error != 0 {
			return error
		}
		if error = cpu.writeFull(step.address1+4, srch); error != 0 {
			return error
		}
		cpu.cc = 0
	} else {
		cpu.regs[step.R1] = srcl
		cpu.regs[step.R1|1] = srch
		cpu.per_mod |= 3 << uint32(step.R1)
		cpu.cc = 1
	}
	return 0
}

// Translate virtual address to real address
func (cpu *CPU) op_lra(step *stepInfo) uint16 {
	// RX instruction in RS range
	if step.R2 != 0 {
		step.address1 += cpu.regs[step.R2]
		step.address1 &= AMASK
	}
	if (cpu.flags & PROBLEM) != 0 {
		//                     /* Try to do quick LRA */
		//                     if (QVMA && vma_lra(R1(reg), addr1, &cc))
		//                         break;
		return IRC_PRIV
		//                     storepsw(OPPSW, IRC_PRIV);
	}
	var seg, page, entry uint32
	var error bool

	// TLB not correct, try loading correct entry
	// Segment and page number to word address
	seg = (step.address1 >> cpu.seg_shift) & cpu.seg_mask
	page = (step.address1 >> cpu.page_shift) & cpu.page_index

	// Check address against length of segment table
	if seg > cpu.seg_len {
		// segment above length of table
		cpu.cc = 3
		cpu.regs[step.R1] = step.address1
		cpu.per_mod |= 1 << step.R1
		return 0
	}

	// Compute address of PTE table
	// Get pointer to page table
	addr := ((seg << 2) + cpu.seg_addr) & AMASK

	// If over size of memory, trap
	mem_cycle++
	if entry, error = memory.GetWord(addr); error {
		return IRC_ADDR
	}

	/* Check if entry valid and in correct length */
	if (entry & PTE_VALID) != 0 {
		cpu.cc = 1
		cpu.regs[step.R1] = addr
		cpu.per_mod |= 1 << step.R1
		return 0
	}

	// Extract length of Table pointer.
	addr = (entry >> 28) + 1

	// Check if entry over end of table
	if (page >> cpu.pte_len_shift) >= addr {
		cpu.cc = 3
		cpu.regs[step.R1] = addr
		cpu.per_mod |= 1 << step.R1
		return 0
	}

	// Now we need to fetch the actual entry
	addr = ((entry & PTE_ADR) + (page << 1)) & AMASK
	mem_cycle++
	if entry, error = memory.GetWord(addr); error {
		return IRC_ADDR
	}

	// extract actual PTE entry
	if (addr & 2) != 0 {
		entry = (addr >> 16)
	}
	entry = entry & 0xffff

	if (entry & (cpu.pte_avail | cpu.pte_mbz)) != 0 {
		cpu.cc = 3
		cpu.regs[step.R1] = addr
		cpu.per_mod |= 1 << step.R1
		return 0
	}

	// Compute correct entry
	entry = entry >> cpu.pte_shift // Move physical to correct spot
	addr = (step.address1 & cpu.page_mask) | (((entry & TLB_PHY) << cpu.page_shift) & AMASK)
	cpu.cc = 0
	cpu.regs[step.R1] = addr
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Execute instruction
func (cpu *CPU) op_ex(step *stepInfo) uint16 {
	var s stepInfo
	var opr uint32
	var error uint16

	// Fetch the next instruction
	if opr, error = cpu.readHalf(step.address1); error != 0 {
		return error
	}

	s.opcode = uint8((opr >> 8) & 0xff)

	// Can't execute execute instruction
	if s.opcode == OP_EX {
		return IRC_EXEC
	}
	s.reg = uint8(opr & 0xff)
	s.R1 = (step.reg >> 4) & 0xf
	s.R2 = step.reg & 0xf
	step.address1 += 2

	// Check type of instruction
	if (s.opcode & 0xc0) != 0 {
		// Check if we need new word
		if s.address1, error = cpu.readHalf(step.address1); error != 0 {
			return error
		}
		s.address1 &= 0xffff
		step.address1 += 2
		// SI instruction
		if (s.opcode & 0xc0) == 0xc0 {
			if s.address2, error = cpu.readHalf(step.address1); error != 0 {
				return error
			}
			s.address2 &= 0xfff
		}
	}

	// Execute instruction
	return cpu.execute(&s)
}

// Signal second processor
func (cpu *CPU) op_sigp(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		return IRC_PRIV
	} else {
		return IRC_OPR // Not supported
	}
}

// Machine check
func (cpu *CPU) op_mc(step *stepInfo) uint16 {
	if (step.reg & 0xf0) != 0 {
		return IRC_SPEC
	}
	if (cpu.cregs[8] & (1 << step.reg)) != 0 {
		mem_cycle++
		memory.SetMemoryMask(0x94, uint32(step.reg)<<16, memory.UMASK)
		return IRC_MCE
	}
	return 0
}

func (cpu *CPU) op_stxsm(step *stepInfo) uint16 {
	var t uint32
	var r uint32

	if (cpu.flags & PROBLEM) != 0 {
		// Try to do quick STNSM
		// if QVMA & vma_stnsm(reg, addr1))
		// 	break
		return IRC_PRIV
	}

	t = 0
	if cpu.ec_mode {
		if cpu.page_en {
			t |= 0x04
		}
		if cpu.irq_en {
			t |= 0x02
		}
		if cpu.per_en {
			t |= 0x40
		}
		if cpu.ext_en {
			t |= 0x01
		}
	} else {
		t = (uint32(cpu.sysmask) >> 8 & 0xfe)
		if cpu.ext_en {
			t |= 0x01
		}
	}

	// Merge mask
	if step.opcode == OP_STNSM {
		r = uint32(step.reg) & t
	} else {
		r = uint32(step.reg) | t
	}

	// Store original value

	if error := cpu.writeByte(step.address1, t); error != 0 {
		return error
	}

	// Set new PSW
	if cpu.ec_mode {
		if (r & 0xb8) != 0 {
			return IRC_SPEC
		}
		cpu.page_en = (r & 0x04) != 0
		cpu.irq_en = (r & 0x02) != 0
		cpu.per_en = (r & 0x40) != 0
		if cpu.irq_en {
			cpu.sysmask = uint16(cpu.cregs[2] >> 16)
		} else {
			cpu.sysmask = 0
		}
		if (r & 0xb8) != 0 {
			return IRC_SPEC
		}
	} else {
		cpu.sysmask = uint16((r << 8) & 0xfc00)
		if (r & 0x2) != 0 {
			cpu.sysmask |= uint16((cpu.cregs[2] >> 16) & 0x3ff)
		}
		cpu.irq_en = cpu.sysmask != 0
	}
	sys_channel.Irq_pending = true
	cpu.ext_en = (r & 0x01) != 0
	return 0
}

// Load control registers
func (cpu *CPU) op_lctl(step *stepInfo) uint16 {
	var error uint16
	var t uint32
	var purge bool

	t = 0
	purge = false
	if (cpu.flags & PROBLEM) != 0 {
		return IRC_PRIV
	}

	for {
		if t, error = cpu.readFull(step.address1); error != 0 {
			return error
		}
		cpu.cregs[step.R1] = t
		switch step.R1 {
		case 0: // General control register
			/* CR0 values
				|    |     |     |   |   |   |   |
			0 0 0 00000 00 1 11 111 1111222222222231|
			0 1 2 34567 89 0 12 345 6789012345678901|
			b s t xxxxx ps 0 ss xxx iiiiiixxiiixxxxx|
			m s d                   mmmmct  iIE     |
			*/
			if (t & 0x80000000) != 0 {
				sys_channel.SetBMUXenable(true)
			} else {
				sys_channel.SetBMUXenable(false)
			}
			cpu.page_shift = 0
			cpu.seg_shift = 0
			switch (t >> 22) & 3 {
			default: // Generate translation exception
			case 1: // 2K page
				cpu.page_shift = 11
				cpu.page_mask = 0x7ff
				cpu.pte_avail = 4
				cpu.pte_mbz = 2
				cpu.pte_shift = 3
				cpu.pte_len_shift = 1
			case 2: // 4K page
				cpu.page_shift = 12
				cpu.page_mask = 0xfff
				cpu.pte_avail = 8
				cpu.pte_mbz = 6
				cpu.pte_shift = 4
				cpu.pte_len_shift = 0
			}

			switch (t >> 19) & 0x7 {
			default: // Generate translation exception
			case 0: // 64K segments
				cpu.seg_shift = 16
				cpu.seg_mask = AMASK >> 16
			case 2: // 1M segments
				cpu.seg_shift = 20
				cpu.seg_mask = AMASK >> 20
				cpu.pte_len_shift += 4
			}
			// Generate PTE index mask
			cpu.page_index = ((^(cpu.seg_mask << cpu.seg_shift) &
				^cpu.page_mask) & AMASK) >> cpu.page_shift
			cpu.interval_en = (t & 0x400) != 0
			cpu.tod_en = (t & 0x800) != 0
		case 1: // Segment table address and length
			purge = true
			cpu.seg_addr = t & AMASK
			cpu.seg_len = (((t >> 24) & 0xff) + 1) << 4
		case 2: // Masks
			if cpu.ec_mode {
				if cpu.irq_en {
					cpu.sysmask = uint16(t >> 16)
				} else {
					cpu.sysmask = 0
				}
				sys_channel.Irq_pending = true
			}
		case 6: // Assist function control
		case 8: // Monitor masks
		case 9: // PER general register masks
		case 10: // PER staring address
		case 11: // PER ending address
		case 14: // Machine Check handleing
		case 15: // Machine check address
		default:
		}
		if step.R1 == step.R2 {
			break
		}
		step.R1++
		step.R1 &= 0xf
		step.address1 += 4
	}

	// Purge TLB if segment pointer is updated
	if purge {
		for i := 0; i < 256; i++ {
			cpu.tlb[i] = 0
		}
	}
	return 0
}

// Store control
func (cpu *CPU) op_stctl(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		// Try to do a quick STCTL
		// if (QVMA && vm_stctl(step))
		// return 0
		return IRC_PRIV
	}
	for {
		if error := cpu.writeFull(step.address1, cpu.cregs[step.R1]); error != 0 {
			return error
		}
		if step.R1 == step.R2 {
			break
		}
		step.R1++
		step.R1 &= 0xf
		step.address1 += 4
	}
	return 0
}

func (cpu *CPU) op_diag(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		return IRC_PRIV
	}
	cpu.storePSW(OMPSW, uint16(step.reg))
	return 0
}

func (cpu *CPU) op_370(step *stepInfo) uint16 {
	var error uint16
	var t1, t2 uint32

	if step.reg > 0x13 {
		return IRC_OPR
	}
	if step.reg != 5 && (cpu.flags&PROBLEM) != 0 {
		//                        /* Try to do quick IPK */
		//                        if (QVMA && vma_370(reg, addr1))
		//                            break;
		return IRC_PRIV
	}
	switch step.reg {
	case 0x00: // CONCS
		// Connect channel set
		fallthrough
	case 0x01: // Disconnect channel set
		if step.address1 == 0 {
			cpu.cc = 0
		} else {
			cpu.cc = 3
		}
	case 0x02: // STIDP
		// Store CPUID in double word
		t1 = uint32(100)
		if error = cpu.writeFull(step.address1, t1); error != 0 {
			return error
		}
		t2 = uint32(0x145) << 16
		return cpu.writeFull(step.address1+4, t2)
	case 0x03: // STIDC
		// Store channel id
		c := uint16(step.address1 & memory.HMASK)
		r := uint32(0)
		switch sys_channel.GetType(c) {
		case sys_channel.TYPE_UNA:
			cpu.cc = 3
			return 0
		case sys_channel.TYPE_MUX:
			r = uint32(0x10000000)
		case sys_channel.TYPE_BMUX:
			r = uint32(0x20000000)
		default:
			// Nop
		}
		memory.SetMemory(0xA8, r)
		cpu.cc = 0
		return 0
	case 0x04: // SCK
		// Load check with double word
		if t1, error = cpu.readFull(step.address1); error != 0 {
			return error
		}
		if t2, error = cpu.readFull(step.address1 + 4); error != 0 {
			return error
		}
		cpu.tod_clock[0] = t1
		cpu.tod_clock[1] = t2
		cpu.tod_set = true
		// cpu.check_tod_irq()
		cpu.cc = 0
	case 0x05: // STCK
		// Store TOD clock in location
		t1 = cpu.tod_clock[0]
		t2 = cpu.tod_clock[1]
		// Update clock based on time before next irq
		t2 &= 0xffff000
		if error = cpu.writeFull(step.address1, t1); error != 0 {
			return error
		}
		if error = cpu.writeFull(step.address1+4, t2); error != 0 {
			return error
		}
		if cpu.tod_set {
			cpu.cc = 0
		} else {
			cpu.cc = 1
		}
	case 0x06: // SCKC
		// Load Clock compare with double word
		if t1, error = cpu.readFull(step.address1); error != 0 {
			return error
		}
		if t2, error = cpu.readFull(step.address1 + 4); error != 0 {
			return error
		}
		cpu.clk_cmp[0] = t1
		cpu.clk_cmp[1] = t2
		// cpu.check_tod_irq()
	case 0x07: // STCKC
		// Store TOD clock in location
		t1 = cpu.clk_cmp[0]
		t2 = cpu.clk_cmp[1]
		if error = cpu.writeFull(step.address1, t1); error != 0 {
			return error
		}
		if error = cpu.writeFull(step.address1+4, t2); error != 0 {
			return error
		}
	case 0x08: // SPT
		// Set the CPU timer with double word
		if t1, error = cpu.readFull(step.address1); error != 0 {
			return error
		}
		if t2, error = cpu.readFull(step.address1 + 4); error != 0 {
			return error
		}
		cpu.cpu_timer[0] = t1
		cpu.cpu_timer[1] = t2
		cpu.tod_set = true
		//                               if (sim_is_active(&cpu_unit[0])) {
		//                                   double nus = sim_activate_time_usecs(&cpu_unit[0]);
		//                                   timer_tics = (int)(nus);
		//                               }
		//                               clk_irq = (cpu_timer[0] & MSIGN) != 0;
	case 0x09: // STPT
		// Store the CPU timer in double word
		t1 = cpu.cpu_timer[0]
		t2 = cpu.cpu_timer[1]
		// Update clock based on time before next irq
		t2 &= 0xffff000
		if error = cpu.writeFull(step.address1, t1); error != 0 {
			return error
		}
		if error = cpu.writeFull(step.address1+4, t2); error != 0 {
			return error
		}
	case 0x0a: // SPKA
		cpu.st_key = uint8(0xf0 & step.address1)
	case 0x0b: // IPK
		cpu.regs[2] = (cpu.regs[2] & 0xffffff00) | (uint32(cpu.st_key) & 0xf0)
		cpu.per_mod |= 1 << 2
	case 0x0d: // PTLB
		for i := 0; i < 256; i++ {
			cpu.tlb[i] = 0
		}
	case 0x10: // SPX
		return IRC_OPR
	case 0x11: // SPTX
		return IRC_OPR
	case 0x12: // STAP
		return IRC_OPR
	case 0x13: // RRB
		// Set storage block reference bit to zero
		k := memory.GetKey(step.address1)
		memory.PutKey(step.address1, k&0xfb)
		cpu.cc = (k >> 1) & 0x3
	default:
		return IRC_OPR
	}
	return 0
}

func (cpu *CPU) op_sio(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		return IRC_PRIV
	}
	cpu.cc = sys_channel.StartIO(uint16(step.address1 & 0xfff))
	return 0
}

func (cpu *CPU) op_tio(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		return IRC_PRIV
	}
	cpu.cc = sys_channel.TestIO(uint16(step.address1 & 0xfff))
	return 0
}

func (cpu *CPU) op_hio(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		return IRC_PRIV
	}
	cpu.cc = sys_channel.HaltioIO(uint16(step.address1 & 0xfff))
	return 0
}

func (cpu *CPU) op_tch(step *stepInfo) uint16 {
	if (cpu.flags & PROBLEM) != 0 {
		return IRC_PRIV
	}
	cpu.cc = sys_channel.TestChan(uint16(step.address1 & 0xfff))
	return 0
}
