/*
   IBM 370 System instructions

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

package cpu

import (
	dev "github.com/rcornwell/S370/emu/device"
	"github.com/rcornwell/S370/emu/memory"
	ch "github.com/rcornwell/S370/emu/sys_channel"
)

// Set storage key.
func (cpu *cpuState) opSSK(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		// Try to do quick SSK
		// if (QVMA && vma_stssk(R1, addr1))
		//    break;
		return ircPriv
	}
	if (step.address1 & 0x0f) != 0 {
		return ircSpec
	}
	if !memory.CheckAddr(step.address1) {
		return ircAddr
	}
	memory.PutKey(step.address1, uint8(step.src1&0xf8))
	return 0
}

// Insert storage Key into register.
func (cpu *cpuState) opISK(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		//  if (QVMA && vma_stisk(src1, addr1))
		// break;
		return ircPriv
	}
	if (step.address1 & 0x0f) != 0 {
		return ircSpec
	}
	if !memory.CheckAddr(step.address1) {
		return ircAddr
	}
	key := memory.GetKey(step.address1)
	cpu.regs[step.R1] &= 0xffffff00
	if cpu.ecMode {
		cpu.regs[step.R1] |= uint32(key) & 0xfe
	} else {
		cpu.regs[step.R1] |= uint32(key) & 0xf8
	}
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Supervisor call.
func (cpu *cpuState) opSVC(step *stepInfo) uint16 {
	//  if ((flags & PROBLEM) != 0 && \
	//  (cpu_unit[0].flags & (FEAT_370|FEAT_VMA)) == (FEAT_370|FEAT_VMA) && \
	//  (cregs[6] & 0x88000000) == MSIGN && vma_stsvc(reg))
	//  break
	irqaddr := cpu.storePSW(oSPSW, uint16(step.reg))
	memCycle++
	src1 := memory.GetMemory(irqaddr)
	memCycle++
	src2 := memory.GetMemory(irqaddr + 0x4)
	cpu.lpsw(src1, src2)
	return 0
}

// Set system mask.
func (cpu *cpuState) opSSM(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		//      sim_debug(DEBUG_VMA, &cpu_dev, "SSM  CR6 %08x\n", cregs[6]);
		//      if (QVMA && vma_ssm(addr1))
		//          break;
		return ircPriv
	} else if (cpu.cregs[0] & 0x40000000) != 0 {
		return ircSpecOp
	}

	// Fetch new system mask
	newSSM, err := cpu.readByte(step.address1)
	if err != 0 {
		return err
	}

	// If in EC Mode, update various flags.
	cpu.extEnb = (newSSM & uint32(extEnable)) != 0
	if cpu.ecMode {
		if (newSSM & uint32(irqEnable)) != 0 {
			cpu.irqEnb = true
			cpu.sysMask = uint16(cpu.cregs[2] >> 16)
		} else {
			cpu.irqEnb = false
			cpu.sysMask = 0
		}
		cpu.pageEnb = (newSSM & uint32(datEnable)) != 0
		cpu.perEnb = (newSSM & uint32(perEnable)) != 0
		if (newSSM & 0xb8) != 0 {
			return ircSpec
		}
	} else {
		cpu.sysMask = uint16(newSSM&0xfc) << 8
		if (newSSM & 0x2) != 0 {
			cpu.sysMask |= uint16((cpu.cregs[2] >> 16) & 0x3ff)
		}
		cpu.irqEnb = cpu.sysMask != 0
		cpu.pageEnb = false
	}
	ch.IrqPending = true
	return 0
}

// Load processor status word.
func (cpu *cpuState) opLPSW(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		//       if (QVMA && vma_lpsw(addr1))
		//           break;
		return ircPriv
	} else if (step.address1 & 0x7) != 0 {
		return ircSpec
	}
	var src1, src2 uint32
	var err uint16

	src1, err = cpu.readFull(step.address1)
	if err != 0 {
		return err
	}
	src2, err = cpu.readFull(step.address1 + 4)
	if err != 0 {
		return err
	}
	//fmt.Printf("LPSW: %08x %08x\n", src1, src2)
	cpu.lpsw(src1, src2)
	return 0
}

// Compare and swap.
func (cpu *cpuState) opCS(step *stepInfo) uint16 {
	var err uint16
	var orig uint32
	var src uint32

	if (step.address1 & 0x3) != 0 {
		return ircSpec
	}
	orig, err = cpu.readFull(step.address1)
	if err != 0 {
		return err
	}
	src = cpu.regs[step.R2]
	if cpu.regs[step.R1] == orig {
		err = cpu.writeFull(step.address1, src)
		if err != 0 {
			return err
		}
		cpu.cc = 0
	} else {
		cpu.regs[step.R1] = orig
		cpu.perRegMod |= 1 << uint32(step.R1)
		cpu.cc = 1
	}
	return 0
}

// Compare and swap double.
func (cpu *cpuState) opCDS(step *stepInfo) uint16 {
	var err uint16
	var origl, origh uint32
	var srcl, srch uint32

	if (step.address1&0x7) != 0 || (step.R1&1) != 0 || (step.R2&1) != 0 {
		return ircSpec
	}
	origl, err = cpu.readFull(step.address1)
	if err != 0 {
		return err
	}
	origh, err = cpu.readFull(step.address1 + 4)
	if err != 0 {
		return err
	}
	srcl = cpu.regs[step.R2]
	srch = cpu.regs[step.R2|1]
	if origl == srcl && origh == srch {
		err = cpu.writeFull(step.address1, srcl)
		if err != 0 {
			return err
		}
		err = cpu.writeFull(step.address1+4, srch)
		if err != 0 {
			return err
		}
		cpu.cc = 0
	} else {
		cpu.regs[step.R1] = srcl
		cpu.regs[step.R1|1] = srch
		cpu.perRegMod |= 3 << uint32(step.R1)
		cpu.cc = 1
	}
	return 0
}

// Translate virtual address to real address.
func (cpu *cpuState) opLRA(step *stepInfo) uint16 {
	// RX instruction in RS range
	if step.R2 != 0 {
		step.address1 += cpu.regs[step.R2]
		step.address1 &= AMASK
	}
	if (cpu.flags & problem) != 0 {
		//                     /* Try to do quick LRA */
		//                     if (QVMA && vma_lra(R1(reg), addr1, &cc))
		//                         break;
		return ircPriv
		//                     storepsw(OPPSW, IRC_PRIV);
	}
	var seg, page, entry uint32
	var err bool

	// TLB not correct, try loading correct entry
	// Segment and page number to word address
	seg = (step.address1 >> cpu.segShift) & cpu.segMask
	page = (step.address1 >> cpu.pageShift) & cpu.pageIndex

	// Check address against length of segment table
	if seg > cpu.segLen {
		// segment above length of table
		cpu.cc = 3
		cpu.regs[step.R1] = step.address1
		cpu.perRegMod |= 1 << step.R1
		return 0
	}

	// Compute address of PTE table
	// Get pointer to page table
	addr := ((seg << 2) + cpu.segAddr) & AMASK

	// If over size of memory, trap
	memCycle++
	entry, err = memory.GetWord(addr)
	if err {
		return ircAddr
	}

	/* Check if entry valid and in correct length */
	if (entry & pteValid) != 0 {
		cpu.cc = 1
		cpu.regs[step.R1] = addr
		cpu.perRegMod |= 1 << step.R1
		return 0
	}

	// Extract length of Table pointer.
	addr = (entry >> 28) + 1

	// Check if entry over end of table
	if (page >> cpu.pteLenShift) >= addr {
		cpu.cc = 3
		cpu.regs[step.R1] = addr
		cpu.perRegMod |= 1 << step.R1
		return 0
	}

	// Now we need to fetch the actual entry
	addr = ((entry & pteAddr) + (page << 1)) & AMASK
	memCycle++
	entry, err = memory.GetWord(addr)
	if err {
		return ircAddr
	}

	// extract actual PTE entry
	if (addr & 2) != 0 {
		entry = (addr >> 16)
	}
	entry &= 0xffff

	if (entry & (cpu.pteAvail | cpu.pteMBZ)) != 0 {
		cpu.cc = 3
		cpu.regs[step.R1] = addr
		cpu.perRegMod |= 1 << step.R1
		return 0
	}

	// Compute correct entry
	entry >>= cpu.pteShift // Move physical to correct spot
	addr = (step.address1 & cpu.pageMask) | (((entry & tlbPhy) << cpu.pageShift) & AMASK)
	cpu.cc = 0
	cpu.regs[step.R1] = addr
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Execute instruction.
func (cpu *cpuState) opEX(step *stepInfo) uint16 {
	var s stepInfo // New stepInfo

	// Fetch the next instruction
	opr, err := cpu.readHalf(step.address1)
	if err != 0 {
		return err
	}
	s.opcode = uint8((opr >> 8) & 0xff)

	// Check if triggered PER event.
	if cpu.perEnb && cpu.perFetch {
		cpu.perAddrCheck(step.address1, 0x4000)
	}

	// Can't execute an execute instruction
	if s.opcode == OpEX {
		return ircExec
	}
	s.reg = uint8(step.src1 & 0xff)
	s.R1 = (s.reg >> 4) & 0xf
	s.R2 = s.reg & 0xf
	step.address1 += 2

	// Check type of instruction
	if (s.opcode & 0xc0) != 0 {
		// Check if we need new word
		a1, err := cpu.readHalf(step.address1)
		if err != 0 {
			return err
		}
		s.address1 = a1 & 0xffff
		step.address1 += 2
		// SI instruction
		if (s.opcode & 0xc0) == 0xc0 {
			a2, err := cpu.readHalf(step.address1)
			if err != 0 {
				return err
			}
			s.address2 = a2 & 0xfff
		}
	}

	// Execute instruction
	return cpu.execute(&s)
}

// Signal second processor.
func (cpu *cpuState) opSIGP(_ *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		return ircPriv
	}
	return ircOper // Not supported
}

// Machine check.
func (cpu *cpuState) opMC(step *stepInfo) uint16 {
	if (step.reg & 0xf0) != 0 {
		return ircSpec
	}
	if (cpu.cregs[8] & (1 << step.reg)) != 0 {
		memCycle++
		memory.SetMemoryMask(0x94, uint32(step.reg)<<16, HMASK)
		return ircMCE
	}
	return 0
}

// And or Or byte with system mask.
func (cpu *cpuState) opSTxSM(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		// Try to do quick STNSM
		// if QVMA & vma_stnsm(reg, addr1))
		// 	break
		return ircPriv
	}

	var oldSSM uint8
	var newSSM uint8

	if cpu.ecMode {
		if cpu.pageEnb {
			oldSSM |= datEnable
		}
		if cpu.irqEnb {
			oldSSM |= irqEnable
		}
		if cpu.perEnb {
			oldSSM |= perEnable
		}
		if cpu.extEnb {
			oldSSM |= extEnable
		}
	} else {
		oldSSM = uint8((cpu.sysMask) >> 8 & 0xfe)
		if cpu.extEnb {
			oldSSM |= extEnable
		}
	}

	// Merge mask
	if step.opcode == OpSTNSM {
		newSSM = step.reg & oldSSM
	} else {
		newSSM = step.reg | oldSSM
	}

	// Store original value

	if err := cpu.writeByte(step.address1, uint32(newSSM)); err != 0 {
		return err
	}

	// Set new PSW
	if cpu.ecMode {
		if (newSSM & 0xb8) != 0 {
			return ircSpec
		}
		cpu.pageEnb = (newSSM & datEnable) != 0
		cpu.irqEnb = (newSSM & irqEnable) != 0
		cpu.perEnb = (newSSM & perEnable) != 0
		if cpu.irqEnb {
			cpu.sysMask = uint16(cpu.cregs[2] >> 16)
		} else {
			cpu.sysMask = 0
		}
	} else {
		cpu.sysMask = (uint16(newSSM) << 8) & uint16(0xfc00)
		if (newSSM & irqEnable) != 0 {
			cpu.sysMask |= uint16(cpu.cregs[2]>>16) & uint16(0x3ff)
		}
		cpu.irqEnb = cpu.sysMask != 0
	}
	ch.IrqPending = true
	cpu.extEnb = (newSSM & extEnable) != 0
	return 0
}

// Load control registers.
func (cpu *cpuState) opLCTL(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		return ircPriv
	}

	for {
		var temp uint32
		var err uint16
		if temp, err = cpu.readFull(step.address1); err != 0 {
			return err
		}
		cpu.cregs[step.R1] = temp
		switch step.R1 {
		case 0: // General control register
			/* CR0 values
				|    |     |     |   |   |   |   |
			0 0 0 00000 00 1 11 111 1111222222222231|
			0 1 2 34567 89 0 12 345 6789012345678901|
			b s t xxxxx ps 0 ss xxx iiiiiixxiiixxxxx|
			m s d                   mmmmct  iIE     |
			*/
			if (temp & 0x80000000) != 0 {
				ch.SetBMUXenable(true)
			} else {
				ch.SetBMUXenable(false)
			}
			cpu.pageShift = 0
			cpu.segShift = 0
			switch (temp >> 22) & 3 {
			default: // Generate translation exception
			case 1: // 2K page
				cpu.pageShift = 11
				cpu.pageMask = 0x7ff
				cpu.pteAvail = 4
				cpu.pteMBZ = 2
				cpu.pteShift = 3
				cpu.pteLenShift = 1
			case 2: // 4K page
				cpu.pageShift = 12
				cpu.pageMask = 0xfff
				cpu.pteAvail = 8
				cpu.pteMBZ = 6
				cpu.pteShift = 4
				cpu.pteLenShift = 0
			}

			switch (temp >> 19) & 0x7 {
			default: // Generate translation exception
			case 0: // 64K segments
				cpu.segShift = 16
				cpu.segMask = AMASK >> 16
			case 2: // 1M segments
				cpu.segShift = 20
				cpu.segMask = AMASK >> 20
				cpu.pteLenShift += 4
			}
			// Generate PTE index mask
			cpu.pageIndex = ((^(cpu.segMask << cpu.segShift) &
				^cpu.pageMask) & AMASK) >> cpu.pageShift
			cpu.intEnb = (temp & 0x400) != 0
			cpu.todEnb = (temp & 0x800) != 0

		case 1: // Segment table address and length
			for i := range 256 {
				cpu.tlb[i] = 0
			}
			cpu.segAddr = temp & AMASK
			cpu.segLen = (((temp >> 24) & 0xff) + 1) << 4

		case 2: // Masks
			if cpu.ecMode {
				if cpu.irqEnb {
					cpu.sysMask = uint16(temp >> 16)
				} else {
					cpu.sysMask = 0
				}
				ch.IrqPending = true
			}

		case 6: // Assist function control
			if cpu.vmAssist && (temp&0xc0000000) == 0x80000000 {
				cpu.vmaEnb = true
			} else {
				cpu.vmaEnb = false
			}

		case 8: // Monitor masks

		case 9: // PER general register masks
			cpu.perBranch = (temp & 0x80000000) != 0
			cpu.perFetch = (temp & 0x40000000) != 0
			cpu.perStore = (temp & 0x20000000) != 0
			cpu.perReg = (temp & 0x10000000) != 0
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

	return 0
}

// Store control.
func (cpu *cpuState) opSTCTL(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		// Try to do a quick STCTL
		// if (QVMA && vm_stctl(step))
		// return 0
		return ircPriv
	}
	for {
		if err := cpu.writeFull(step.address1, cpu.cregs[step.R1]); err != 0 {
			return err
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

// CPU Diagnostic instruction.
func (cpu *cpuState) opDIAG(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		return ircPriv
	}
	cpu.storePSW(oMPSW, uint16(step.reg))
	return 0
}

// Handle special 370 opcodes.
func (cpu *cpuState) opB2(step *stepInfo) uint16 {
	if step.reg > 0x13 {
		return ircOper
	}
	if step.reg != 5 && (cpu.flags&problem) != 0 {
		//                        /* Try to do quick IPK */
		//                        if (QVMA && vma_370(reg, addr1))
		//                            break;
		return ircPriv
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
		t1 := uint32(100)
		err := cpu.writeFull(step.address1, t1)
		if err != 0 {
			return err
		}
		t2 := uint32(0x145) << 16
		return cpu.writeFull(step.address1+4, t2)
	case 0x03: // STIDC
		// Store channel id
		testChan := uint16(step.address1 & HMASK)
		var result uint32
		switch ch.GetType(testChan) {
		case dev.TypeUNA:
			cpu.cc = 3
			return 0
		case dev.TypeMux:
			result = uint32(0x10000000)
		case dev.TypeBMux:
			result = uint32(0x20000000)
		default:
			// Nop
		}
		memory.SetMemory(0xA8, result)
		cpu.cc = 0
		return 0
	case 0x04: // SCK
		var low, high uint32
		var err uint16

		// Load check with double word
		low, err = cpu.readFull(step.address1)
		if err != 0 {
			return err
		}
		high, err = cpu.readFull(step.address1 + 4)
		if err != 0 {
			return err
		}
		cpu.todClock[0] = low
		cpu.todClock[1] = high
		cpu.todSet = true
		// cpu.check_tod_irq()
		cpu.cc = 0
	case 0x05: // STCK
		var low, high uint32

		// Store TOD clock in location
		low = cpu.todClock[0]
		high = cpu.todClock[1]
		// Update clock based on time before next irq
		high &= 0xffff000
		err := cpu.writeFull(step.address1, low)
		if err != 0 {
			return err
		}
		err = cpu.writeFull(step.address1+4, high)
		if err != 0 {
			return err
		}
		if cpu.todSet {
			cpu.cc = 0
		} else {
			cpu.cc = 1
		}
	case 0x06: // SCKC
		var low, high uint32
		var err uint16

		// Load Clock compare with double word
		low, err = cpu.readFull(step.address1)
		if err != 0 {
			return err
		}
		high, err = cpu.readFull(step.address1 + 4)
		if err != 0 {
			return err
		}
		cpu.clkCmp[0] = low
		cpu.clkCmp[1] = high
		// cpu.check_tod_irq()
	case 0x07: // STCKC
		var low, high uint32

		// Store TOD clock in location
		low = cpu.clkCmp[0]
		high = cpu.clkCmp[1]
		err := cpu.writeFull(step.address1, low)
		if err != 0 {
			return err
		}
		err = cpu.writeFull(step.address1+4, high)
		if err != 0 {
			return err
		}
	case 0x08: // SPT
		var low, high uint32
		var err uint16

		// Set the CPU timer with double word
		low, err = cpu.readFull(step.address1)
		if err != 0 {
			return err
		}
		high, err = cpu.readFull(step.address1 + 4)
		if err != 0 {
			return err
		}
		cpu.cpuTimer[0] = low
		cpu.cpuTimer[1] = high
		cpu.todSet = true
		//                               if (sim_is_active(&cpu_unit[0])) {
		//                                   double nus = sim_activate_time_usecs(&cpu_unit[0]);
		//                                   timer_tics = (int)(nus);
		//                               }
		//                               clk_irq = (cpu_timer[0] & MSIGN) != 0;
	case 0x09: // STPT
		var low, high uint32

		// Store the CPU timer in double word
		low = cpu.cpuTimer[0]
		high = cpu.cpuTimer[1]
		// Update clock based on time before next irq
		high &= 0xffff000
		err := cpu.writeFull(step.address1, low)
		if err != 0 {
			return err
		}
		err = cpu.writeFull(step.address1+4, high)
		if err != 0 {
			return err
		}
	case 0x0a: // SPKA
		cpu.stKey = uint8(0xf0 & step.address1)
	case 0x0b: // IPK
		cpu.regs[2] = (cpu.regs[2] & 0xffffff00) | (uint32(cpu.stKey) & 0xf0)
		cpu.perRegMod |= 1 << 2
	case 0x0d: // PTLB
		for i := range 256 {
			cpu.tlb[i] = 0
		}
	case 0x10: // SPX
		return ircOper
	case 0x11: // SPTX
		return ircOper
	case 0x12: // STAP
		return ircOper
	case 0x13: // RRB
		// Set storage block reference bit to zero
		key := memory.GetKey(step.address1)
		memory.PutKey(step.address1, key&0xfb)
		cpu.cc = (key >> 1) & 0x3
	default:
		return ircOper
	}
	return 0
}

// Start I/O Operation.
func (cpu *cpuState) opSIO(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		return ircPriv
	}
	cpu.cc = ch.StartIO(uint16(step.address1 & 0xfff))
	return 0
}

// Test state of device.
func (cpu *cpuState) opTIO(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		return ircPriv
	}
	cpu.cc = ch.TestIO(uint16(step.address1 & 0xfff))
	return 0
}

// Halt I/O device.
func (cpu *cpuState) opHIO(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		return ircPriv
	}
	cpu.cc = ch.HaltIO(uint16(step.address1 & 0xfff))
	return 0
}

// Check state of channel.
func (cpu *cpuState) opTCH(step *stepInfo) uint16 {
	if (cpu.flags & problem) != 0 {
		return ircPriv
	}
	cpu.cc = ch.TestChan(uint16(step.address1 & 0xfff))
	return 0
}
