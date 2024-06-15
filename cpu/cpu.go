package cpu

/* CPU definitions for IBM 370 simulator definitions

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
   ROBERT M SUPNIK BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
   IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

*/

import (
	"time"

	"github.com/rcornwell/S370/memory"
	"github.com/rcornwell/S370/sys_channel"
)

var cpu CPU

var mem_cycle int

// Return pointer to CPU variable
func New() *CPU {
	return &cpu
}

// Initialize CPU to basic state
func InitializeCPU() {
	cpu.createTable()
	cpu.PC = 0
	cpu.sysMask = 0
	cpu.stKey = 0
	cpu.cc = 0
	cpu.ilc = 0
	cpu.pmask = 0
	cpu.flags = 0
	cpu.per_mod = 0
	cpu.per_addr = 0
	cpu.per_code = 0
	cpu.clk_cmp[0] = FMASK
	cpu.clk_cmp[1] = FMASK
	cpu.clk_state = false
	cpu.timer_tics = 0
	cpu.cpu_timer[0] = 0
	cpu.cpu_timer[1] = 0
	cpu.per_en = false
	cpu.ecMode = false
	cpu.pageEnb = false
	cpu.irq_en = false
	cpu.ext_en = false
	cpu.ext_irq = false
	cpu.interval_irq = false
	cpu.interval_en = false
	cpu.tod_en = false
	cpu.tod_irq = false
	cpu.vmEnb = false

	// Clear registers
	for i := range 16 {
		cpu.regs[i] = 0
		cpu.cregs[i] = 0
	}

	// Initialize Control regisers to default
	cpu.cregs[0] = 0x000000e0
	cpu.cregs[2] = 0xffffffff
	cpu.cregs[14] = 0xc2000000
	cpu.cregs[15] = 512

	// Clear floating point registers
	for i := range 8 {
		cpu.fpregs[i] = 0
	}

	// Clear TBL tables
	for i := range 256 {
		cpu.tlb[i] = 0
	}

	// Set clock to current time
	if !cpu.tod_set {
		// Set TOD to current time
		now := time.Now()
		sec := now.Unix()

		// IBM measures time from 1900, Unix starts at 1970
		// Add in number of years from 1900 to 1970 + 17 leap days
		sec += ((70 * 365) + 17) * 86400
		sec *= 1000000
		sec <<= 12
		usec := uint64(sec)
		cpu.tod_clock[0] = uint32((usec >> 32) & LMASKL)
		cpu.tod_clock[1] = uint32(usec & LMASKL)
		cpu.tod_set = true
	}

	cpu.page_mask = 0
}

// Execute one instruction or take an interrupt
func Cycle() int {
	var err uint16
	mem_cycle = 0
	cpu.per_mod = 0
	cpu.per_code = 0
	cpu.per_addr = cpu.PC
	cpu.iPC = cpu.PC
	cpu.ilc = 0

	// Check if we should see if an IRQ is pending
	irq := sys_channel.Chan_scan(cpu.sysMask, cpu.irq_en)
	if irq != sys_channel.NO_DEV {
		cpu.ilc = 0
		if sys_channel.Loading != sys_channel.NO_DEV {
			cpu.suppress(OIOPSW, irq)
		}
		return mem_cycle
	}

	// Check for external interrupts
	if cpu.ext_en {
		if cpu.ext_irq {
			if !cpu.ecMode || (cpu.cregs[0]&0x20) != 0 ||
				(cpu.cregs[6]&0x40) != 0 {
				cpu.ext_irq = false
				cpu.suppress(OEPSW, 0x40)
				return mem_cycle
			}
		}

		if cpu.interval_irq && (cpu.cregs[0]&0x80) != 0 {
			cpu.interval_irq = false
			cpu.suppress(OEPSW, 0x80)
			return mem_cycle
		}
		if cpu.clk_irq && cpu.interval_en {
			cpu.clk_irq = false
			cpu.suppress(OEPSW, 0x1005)
			return mem_cycle
		}
		if cpu.tod_irq && cpu.tod_en {
			cpu.tod_irq = false
			cpu.suppress(OEPSW, 0x1004)
			return mem_cycle
		}
	}

	/* If we have wait flag or loading, nothing more to do */
	if sys_channel.Loading != sys_channel.NO_DEV || (cpu.flags&WAIT) != 0 {
		/* CPU IDLE */
		if !cpu.irq_en && !cpu.ext_en {
			return mem_cycle
		}
	}

	if (cpu.PC & 1) != 0 {
		cpu.suppress(OPPSW, IRC_SPEC)
		return mem_cycle
	}

	// Check if triggered PER event.
	if cpu.per_en && (cpu.cregs[9]&0x40000000) != 0 {
		if cpu.cregs[10] <= cpu.cregs[11] {
			if cpu.PC >= cpu.cregs[10] && cpu.PC <= cpu.cregs[11] {
				cpu.per_code |= 0x4000
			}
		} else {
			if cpu.PC >= cpu.cregs[11] || cpu.PC <= cpu.cregs[10] {
				cpu.per_code |= 0x4000
			}
		}
	}

	var opr, t uint32
	var step stepInfo

	// Fetch the next instruction
	t, err = cpu.readFull(cpu.PC & ^uint32(0x2))
	if err != 0 {
		cpu.suppress(OPPSW, err)
		return mem_cycle
	}

	// Save instruction
	if (cpu.PC & 2) == 0 {
		opr = (t >> 16) & 0xffff
	} else {
		opr = t & 0xffff
	}
	cpu.ilc++
	step.opcode = uint8((opr >> 8) & 0xff)
	step.reg = uint8(opr & 0xff)
	step.R1 = (step.reg >> 4) & 0xf
	step.R2 = step.reg & 0xf
	cpu.PC += 2

	// Check type of instruction
	if (step.opcode & 0xc0) != 0 {
		// Check if we need new word
		cpu.ilc++
		if (cpu.PC & 2) == 0 {
			t, err = cpu.readFull(cpu.PC & ^uint32(0x2))
			if err != 0 {
				cpu.suppress(OPPSW, err)
				return mem_cycle
			}
			step.address1 = (t >> 16)
		} else {
			step.address1 = t
		}
		step.address1 &= 0xffff
		cpu.PC += 2
		// SI instruction
		if (step.opcode & 0xc0) == 0xc0 {
			cpu.ilc++
			if (cpu.PC & 2) != 0 {
				t, err = cpu.readFull(cpu.PC & ^uint32(0x2))
				if err != 0 {
					cpu.suppress(OPPSW, err)
					return mem_cycle
				}
				step.address2 = (t >> 16)
			} else {
				step.address2 = t
			}
			step.address2 &= 0xffff
			cpu.PC += 2
		}
	}

	err = cpu.execute(&step)
	if err != 0 {
		cpu.suppress(OPPSW, err)
	}

	// See if PER event happened
	if cpu.per_en && cpu.per_code != 0 {
		cpu.suppress(OPPSW, 0)
	}
	return mem_cycle
}

// Generate addresses for operands and if
// approperate fetch the values. Then execute the
// instruction and return any error condition
func (cpu *CPU) execute(step *stepInfo) uint16 {
	// Compute addresses of operands
	if (step.opcode & 0xc0) != 0 { // RS, RX, SS
		temp := (step.address1 >> 12) & 0xf
		step.address1 = step.address1 & 0xfff
		if temp != 0 {
			step.address1 += cpu.regs[temp]
		}
		step.address1 &= AMASK
		step.src1 = step.address1

		//* Handle RX type operands
		if (step.opcode & 0x80) == 0 {
			if step.R2 != 0 {
				step.address1 += cpu.regs[step.R2]
			}
		} else if (step.opcode & 0xc0) != 0xc0 { // SS
			temp = (step.address2 >> 12) & 0xf
			step.address2 = step.address2 & 0xfff
			if temp != 0 {
				step.address2 += cpu.regs[temp]
			}
			step.address2 &= AMASK
		}
	}

	var err uint16

	// Read operands
	// Check if floating point
	if (step.opcode & 0xA0) == 0x20 {
		if (step.R1 & 0x9) != 0 {
			return IRC_SPEC
		}

		// Load operands
		step.fsrc1 = cpu.fpregs[step.R1]
		// Check for short
		if (step.opcode & 0x10) != 0 {
			step.fsrc1 &= HMASKL
		}

		// RX instruction
		if (step.opcode & 0x40) != 0 {
			var src1, src2 uint32
			src1, err = cpu.readFull(step.address1)
			if err != 0 {
				return err
			}

			// Check for long
			if (step.opcode & 0x10) == 0 {
				src2, err = cpu.readFull(step.address2)
				if err != 0 {
					return err
				}
			} else {
				src2 = 0
			}
			step.fsrc2 = (uint64(src1) << 32) | uint64(src2)
		} else {
			if (step.R2 & 0x9) != 0 {
				return IRC_SPEC
			}
			step.fsrc2 = cpu.fpregs[step.R2]
			if (step.opcode & 0x10) != 0 {
				step.fsrc2 &= HMASKL
			}
		}
		// All RR opcodes
	} else if (step.opcode & 0xe0) == 0 {
		step.src1 = cpu.regs[step.R1]
		step.src2 = cpu.regs[step.R2]
		step.address1 = (step.src2) & AMASK
		// All RX integer ops
	} else if (step.opcode & 0xe0) == 0x40 {
		step.src1 = cpu.regs[step.R1]
		// Read half word if 010010xx or 01001100
		if (step.opcode&0xfc) == 0x48 || step.opcode == OP_MH {
			step.src2, err = cpu.readHalf(step.address1)
			if err != 0 {
				return err
			}
			// Read full word if 0101xxx and not xxxx00xx (ST)
		} else if (step.opcode&0x10) != 0 && (step.opcode&0x0c) != 0 {
			step.src2, err = cpu.readFull(step.address1)
			if err != 0 {
				return err
			}
		} else {
			step.address2 = step.src2
		}
	}

	// Execute the instruction.
	err = cpu.table[step.opcode](step)
	if cpu.per_en && (cpu.cregs[9]&0x10000000) != 0 && (cpu.cregs[9]&0xffff&cpu.per_mod) != 0 {
		cpu.per_code |= 0x1000
	}

	return err
}

// Create function table
func (c *CPU) createTable() {
	c.table = [256]func(*stepInfo) uint16{
		//  0         1         2         3          4         5         6          7
		c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opSPM, c.opBAL, c.opBCT, c.opBC, // 0x
		//  8         9         A         B          C         D         E          F
		c.opSSK, c.opISK, c.opSVC, c.opUnk, c.opUnk, c.opBAS, c.opMVCL, c.opCLCL,

		c.opLPR, c.opLNR, c.opLTR, c.opLCR, c.opAnd, c.opCmpL, c.opOr, c.opXor, // 1x
		c.opL, c.opCmp, c.opAdd, c.opSub, c.opMul, c.opDiv, c.opAddL, c.opSubL,

		c.opLcs, c.opLcs, c.opLcs, c.opLcs, c.opFPHalf, c.opLRDR, c.opMXR, c.opMXD, // 2x
		c.opFPLoad, c.opCD, c.opFPAddD, c.opFPAddD, c.opFPMul, c.opFPDiv, c.opFPAddD, c.opFPAddD,

		c.opLcs, c.opLcs, c.opLcs, c.opLcs, c.opFPHalf, c.opLRER, c.opAXR, c.opAXR, // 3x
		c.opFPLoad, c.opCE, c.opFPAdd, c.opFPAdd, c.opFPMul, c.opFPDiv, c.opFPAdd, c.opFPAdd,

		c.opSTH, c.opL, c.opSTC, c.opIC, c.opEX, c.opBAL, c.opBCT, c.opBC, // 4x
		c.opL, c.opCmp, c.opAdd, c.opSub, c.opMulH, c.opBAS, c.opCVD, c.opCVB,

		c.opST, c.opUnk, c.opUnk, c.opUnk, c.opAnd, c.opCmpL, c.opOr, c.opXor, // 5x
		c.opL, c.opCmp, c.opAdd, c.opSub, c.opMul, c.opDiv, c.opAddL, c.opSubL,

		c.opSTD, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opMXD, // 6x
		c.opFPLoad, c.opCD, c.opFPAddD, c.opFPAddD, c.opFPMul, c.opFPDiv, c.opFPAddD, c.opFPAddD,

		c.opSTE, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, // 7x
		c.opFPLoad, c.opCE, c.opFPAdd, c.opFPAdd, c.opFPMul, c.opFPDiv, c.opFPAdd, c.opFPAdd,

		c.opSSM, c.opUnk, c.opLPSW, c.opDIAG, c.opUnk, c.opUnk, c.opBXH, c.op_BXLE, // 8x
		c.opSRL, c.opSLL, c.opSRA, c.opSLA, c.opSRDL, c.opSLDL, c.opSRDA, c.opSLDA,

		c.opSTM, c.opTM, c.opMVI, c.opTS, c.opNI, c.opCLI, c.opOI, c.opXI, // 9x
		c.opLM, c.opUnk, c.opUnk, c.opUnk, c.opSIO, c.opTIO, c.opHIO, c.opTCH,

		c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, // Ax
		c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opSTxSM, c.opSTxSM, c.opSIGP, c.opMC,

		c.opUnk, c.opLRA, c.opB2, c.opUnk, c.opUnk, c.opUnk, c.opSTCTL, c.opLCTL, // Bx
		c.opUnk, c.opUnk, c.opCS, c.opCDS, c.opUnk, c.opCLM, c.opSTCM, c.opICM,

		c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, // Cx
		c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk,

		c.opUnk, c.opMem, c.opMem, c.opMem, c.opMem, c.opCLC, c.opMem, c.opMem, // Dx
		c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opTR, c.opTR, c.opED, c.opED,

		c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, // Ex
		c.opMVCIN, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk, c.opUnk,

		c.opSRP, c.opMVO, c.opPACK, c.opUNPK, c.opUnk, c.opUnk, c.opUnk, c.opUnk, // Fx
		c.opDecAdd, c.opDecAdd, c.opDecAdd, c.opDecAdd, c.opMP, c.opDP, c.opUnk, c.opUnk,
	}
}

/*
 *     PS = 2K     page_shift = 11   pte_avail = 0x4  pte_mbz = 0x2 pte_shift = 3
 *     PS = 4K     page_shift = 12   pte_avail = 0x8  pte_mbz = 0x6 pte_shift = 4
 *
 *       SS = 64K  page_mask = 0x1F     PS=4K page_mask = 0xF
 *                 seg_shift = 16
 *                 seg_mask = 0xff
 *       SS = 1M   page_mask = 0xff     PS=4k page_mask = 07F
 *                 seg_shift = 20
 *                 seg_mask = 0xF
 * For 360/67
 *                 page_shift = 12
 *                 page_mask = 0xff
 *                 seg_shift = 20
 *                 seg_mask = 0xfff
 */

// Translate an address from virtual to physical.
func (cpu *CPU) transAddr(va uint32) (pa uint32, error uint16) {
	var entry uint32
	var err bool

	// Check address in range
	addr := va & AMASK

	// If paging not enabled, return address.
	if !cpu.pageEnb {
		return addr, 0
	}

	// Extract page address is on
	page := addr >> cpu.page_shift

	// Extract segment and move it into place.
	seg := (page & 0x1f00) << 4

	// Only 256 pages.
	page &= 0xff

	//* Quick check if TLB correct
	entry = cpu.tlb[page]
	if (entry&TLB_VALID) != 0 && ((entry^seg)&TLB_SEG) == 0 {
		addr = (va & cpu.page_mask) | ((entry & TLB_PHY) << cpu.page_shift)
		return addr, 0
	}

	// TLB entry does not match, replace it.
	// Clear whatever was in entry
	cpu.tlb[page] = 0
	// TLB not correct, try loading correct entry
	// Segment and page number to word address
	seg = (addr >> cpu.seg_shift) & cpu.seg_mask
	page = (addr >> cpu.page_shift) & cpu.page_index

	// Check address against length of segment table
	if seg > cpu.seg_len {
		// segment above length of table,
		// write failed address and 90, then trigger trap.
		_ = memory.PutWord(0x90, va)
		mem_cycle++
		cpu.PC = cpu.iPC
		return 0, IRC_SEG
	}

	// Compute address of PTE table
	// Get pointer to page table
	addr = ((seg << 2) + cpu.seg_addr) & AMASK

	// Get entry on error throw trap.
	mem_cycle++
	entry, err = memory.GetWord(addr)
	if err {
		return 0, IRC_ADDR
	}

	// Extract length of Table pointer.
	addr = (entry >> 28) + 1

	/* Check if entry valid and in correct length */
	if (entry&PTE_VALID) != 0 || (page>>cpu.pte_len_shift) >= addr {
		mem_cycle++
		memory.SetMemory(0x90, va)
		cpu.PC = cpu.iPC
		if (entry & PTE_VALID) != 0 {
			return 0, IRC_SEG
		}
		return 0, IRC_PAGE
	}

	// Now we need to fetch the actual entry
	addr = ((entry & PTE_ADR) + (page << 1)) & AMASK
	mem_cycle++
	entry, err = memory.GetWord(addr)
	if err {
		return 0, IRC_ADDR
	}

	// extract actual PTE entry
	if (addr & 2) != 0 {
		entry = (addr >> 16)
	}
	entry = entry & 0xffff

	if (entry & cpu.pte_mbz) != 0 {
		mem_cycle++
		memory.SetMemory(0x90, va)
		cpu.PC = cpu.iPC
		return 0, IRC_SPEC
	}

	// Check if entry valid and in correct length
	if (entry & cpu.pte_avail) != 0 {
		mem_cycle++
		memory.SetMemory(0x90, va)
		cpu.PC = cpu.iPC
		return 0, IRC_PAGE
	}

	// Compute correct entry
	entry = entry >> cpu.pte_shift // Move physical to correct spot
	page = va >> cpu.page_shift
	entry = entry | ((page & 0x1f00) << 4) | TLB_VALID
	// Update TLB with new entry
	cpu.tlb[page&0xff] = entry
	// Compute physical address
	addr = (va & cpu.page_mask) | (((entry & TLB_PHY) << cpu.page_shift) & AMASK)
	return addr, 0
}

/*
 * Store the PSW at given address with irq value.
 */
func (cpu *CPU) storePSW(vector uint32, irqcode uint16) (irqaddr uint32) {
	var word1, word2 uint32
	irqaddr = vector + 0x40

	if vector == OPPSW && cpu.per_en && cpu.per_code != 0 {
		irqcode |= IRC_PER
	}
	if cpu.ecMode {
		// Generate first word
		word1 = uint32(0x80000) |
			(uint32(cpu.stKey) << 16) |
			(uint32(cpu.flags) << 16) |
			(uint32(cpu.cc) << 12) |
			(uint32(cpu.pmask) << 8)
		if cpu.pageEnb {
			word1 |= 1 << 26
		}
		if cpu.per_en {
			word1 |= 1 << 30
		}
		if cpu.irq_en {
			word1 |= 1 << 25
		}
		if cpu.ext_en {
			word1 |= 1 << 24
		}

		// Save code where 370 expects it to be
		switch vector {
		case OEPSW:
			mem_cycle++
			memory.SetMemoryMask(0x84, uint32(irqcode), memory.HMASK)
		case OSPSW:
			mem_cycle++
			memory.SetMemory(0x88, ((uint32(cpu.ilc) << 17) | uint32(irqcode)))
		case OPPSW:
			mem_cycle++
			memory.SetMemory(0x8c, ((uint32(cpu.ilc) << 17) | uint32(irqcode)))
		case OIOPSW:
			mem_cycle++
			memory.SetMemory(0xb8, uint32(irqcode))
		}
		if (irqcode & IRC_PER) != 0 {
			mem_cycle++
			memory.SetMemory(150, (cpu.per_code<<16)|(cpu.per_addr>>16))
			mem_cycle++
			temp := memory.GetMemory(154)
			memory.SetMemory(154, ((cpu.per_addr&0xffff)<<16)|(temp&0xffff))
		}
		// Generate second word.
		word2 = cpu.PC
	} else {
		// Generate first word.
		word1 = (uint32(cpu.sysMask&0xfe00) << 16) |
			(uint32(cpu.stKey) << 16) |
			(uint32(cpu.flags) << 16) |
			uint32(irqcode)
		if cpu.ext_en {
			word1 |= 1 << 24
		}
		// Generate second word. */
		word2 = (uint32(cpu.ilc) << 30) |
			(uint32(cpu.cc) << 28) |
			(uint32(cpu.pmask) << 24) |
			(cpu.PC & AMASK)
	}
	mem_cycle++
	memory.SetMemory(vector, word1)
	mem_cycle++
	memory.SetMemory(vector+4, word2)
	//	sim_debug(DEBUG_INST, &cpu_dev, "store %02x %d %x %03x PSW=%08x %08x\n", addr, ilc,
	//		cc, ircode, word, word2)
	return irqaddr
}

/*
 * Check for protection violation.
 */
func (cpu *CPU) checkProtect(addr uint32, write bool) bool {
	/* Check storage key */
	if cpu.stKey == 0 {
		return false
	}
	k := memory.GetKey(addr)
	if write {
		if (k & 0xf0) != cpu.stKey {
			return true
		}
	} else {
		if (k&0x8) != 0 && (k&0xf0) != cpu.stKey {
			return true
		}
	}
	return false
}

/*
 * Check if we can access a range of memory.
 */
func (cpu *CPU) testAccess(va uint32, size uint32, write bool) uint16 {

	// Translate address
	if pa, err := cpu.transAddr(va); err != 0 {
		return err
	} else {
		if cpu.checkProtect(pa, write) {
			return IRC_PROT
		}
	}

	if size != 0 && (va&SPMASK) != ((va+size)&SPMASK) {
		// Translate end address
		if pa, err := cpu.transAddr(va + size); err != 0 {
			return err
		} else {
			if cpu.checkProtect(pa, write) {
				return IRC_PROT
			}
		}
	}
	return 0
}

/*
 * Read a full word from memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *CPU) readFull(addr uint32) (uint32, uint16) {
	var pa uint32
	var v uint32
	var err bool
	var e uint16

	offset := addr & 3

	// Validate address
	pa, e = cpu.transAddr(addr)
	if e != 0 {
		return 0, e
	}

	if cpu.checkProtect(pa, false) {
		return 0, IRC_PROT
	}

	// Read actual data
	mem_cycle++
	v, err = memory.GetWord(addr)
	if err {
		return 0, IRC_ADDR
	}

	// Handle unaligned access
	if offset != 0 {
		addr2 := addr + 4
		pa2 := pa + 4

		if (addr & SPMASK) != (addr2 & SPMASK) {
			// Check if possible next page
			pa2, e = cpu.transAddr(addr2)
			if e != 0 {
				return 0, e
			}
			// Check access protection
			if cpu.checkProtect(pa2, false) {
				return 0, IRC_PROT
			}
		}

		mem_cycle++
		if t, err := memory.GetWord(pa2); err {
			return 0, IRC_ADDR
		} else {
			v <<= (8 * offset)
			v |= (t >> (8 * (4 - offset)))
		}
	}

	//	sim_debug(DEBUG_DATA, &cpu_dev, "RD A=%08x %08x\n", addr, *v)
	return v, 0
}

/*
 * Read a half word from memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *CPU) readHalf(addr uint32) (uint32, uint16) {
	var pa uint32
	var v uint32
	var err bool
	var e uint16

	offset := addr & 3

	/* Validate address */
	pa, e = cpu.transAddr(addr)
	if e != 0 {
		return 0, e
	}

	// Check storage key
	if cpu.checkProtect(pa, false) {
		return 0, IRC_PROT
	}

	// Get data
	mem_cycle++
	v, err = memory.GetWord(pa)
	if err {
		return 0, IRC_ADDR
	}

	switch offset {
	case 0:
		v >>= 16
	case 1:
		v >>= 8
	case 2:
	case 3:
		pa2 := pa + 1
		// Check if past a word
		if (addr & SPMASK) != ((addr + 1) & SPMASK) {
			/* Check if possible next page */
			pa2, e = cpu.transAddr(addr + 1)
			if e != 0 {
				return 0, e
			}

			// Check storage key
			if cpu.checkProtect(pa2, false) {
				return 0, IRC_PROT
			}
		}

		mem_cycle++
		if v2, err := memory.GetWord(pa2); err {
			return 0, IRC_ADDR
		} else {
			v = (v & 0xff) << 8
			v |= v2 & 0xff
		}
	}

	// Sign extend the result
	v &= HMASK
	if (v & 0x8000) != 0 {
		v |= 0xffff0000
	}
	return v, 0
}

/*
 * Read a byte from memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *CPU) readByte(addr uint32) (uint32, uint16) {
	var pa uint32
	var v uint32
	var err bool
	var e uint16

	offset := addr & 3

	// Validate address
	pa, e = cpu.transAddr(addr)
	if e != 0 {
		return 0, e
	}

	if cpu.checkProtect(pa, false) {
		return 0, IRC_PROT
	}

	// Read actual data
	mem_cycle++
	v, err = memory.GetWord(addr)
	if err {
		return 0, IRC_ADDR
	}

	v = (v >> (8 * (3 - offset))) & 0xff
	//sim_debug(DEBUG_DATA, &cpu_dev, "RD B=%08x %08x\n", addr, *v)
	return v, 0
}

// Check if address is in the range of PER modify range
func (cpu *CPU) perCheck(addr uint32) {
	if cpu.per_en && (cpu.cregs[9]&0x20000000) != 0 {
		if cpu.cregs[10] <= cpu.cregs[11] {
			if addr >= cpu.cregs[10] && addr <= cpu.cregs[11] {
				cpu.per_code |= 0x2000
			}
		} else {
			if addr >= cpu.cregs[11] || addr <= cpu.cregs[10] {
				cpu.per_code |= 0x2000
			}
		}
	}
}

/*
 * Update a full word in memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *CPU) writeFull(addr, data uint32) uint16 {
	var e uint16
	var pa uint32
	var err1, err2 bool

	offset := addr & 3

	// Validate address
	pa, e = cpu.transAddr(addr)
	if e != 0 {
		return e
	}

	// Check storage key
	if cpu.checkProtect(pa, true) {
		return IRC_PROT
	}

	// Check if in storage area
	cpu.perCheck(addr)

	pa2 := pa + 4
	addr2 := (addr & 0x00fffffc) + 4
	if offset != 0 {

		// Check if we handle unaligned access
		if (addr & SPMASK) != (addr2 & SPMASK) {
			// Validate address
			pa2, e = cpu.transAddr(addr2)
			if e != 0 {
				return e
			}

			// Check against storage key
			if cpu.checkProtect(pa2, true) {
				return IRC_PROT
			}
		}

		// Check if in storage area
		cpu.perCheck(addr2)
	}

	switch offset {
	case 0:
		mem_cycle++
		err1 = memory.PutWord(pa, data)
		err2 = false
	case 1:
		mem_cycle++
		err1 = memory.PutWordMask(pa, data>>8, 0x00ffffff)
		mem_cycle++
		err2 = memory.PutWordMask(pa2, data<<24, 0xff000000)
	case 2:
		mem_cycle++
		err1 = memory.PutWordMask(pa, data>>16, 0x0000ffff)
		mem_cycle++
		err2 = memory.PutWordMask(pa2, data<<16, 0xffff0000)
	case 3:
		mem_cycle++
		err1 = memory.PutWordMask(pa, data>>24, 0xff000000)
		mem_cycle++
		err2 = memory.PutWordMask(pa2, data<<8, 0x00ffffff)
	}

	if err1 || err2 {
		e = IRC_ADDR
	} else {
		e = 0
	}
	//	sim_debug(DEBUG_DATA, &cpu_dev, "WR A=%08x %08x\n", addr, data)
	return e
}

/*
 * Update a half word in memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *CPU) writeHalf(addr, data uint32) uint16 {
	var e uint16
	var pa uint32
	var err bool

	offset := addr & 3

	// Validate address			cy = dec_divstep(l int, s1 int, s2 int, v1 *[32]uint8, v2 *[32]uint8) uint8
	pa, e = cpu.transAddr(addr)
	if e != 0 {
		return e
	}

	if cpu.checkProtect(pa, true) {
		return IRC_PROT
	}

	cpu.perCheck(addr)

	switch offset {
	case 0:
		mem_cycle++
		err = memory.PutWordMask(pa, data<<16, 0xffff0000)
	case 1:
		mem_cycle++
		err = memory.PutWordMask(pa, data<<8, 0x00ffff00)
	case 2:
		mem_cycle++
		err = memory.PutWordMask(pa, data, HMASK)
	case 3:
		addr2 := addr + 1
		pa2 := pa + 1

		cpu.perCheck(addr)

		if (addr & SPMASK) != (addr2 & SPMASK) {
			// Validate address
			pa2, e = cpu.transAddr(addr2)
			if e != 0 {
				return e
			}

			// Check against storage key
			if cpu.checkProtect(pa2, true) {
				return IRC_PROT
			}
		}

		mem_cycle++
		mem_cycle++
		err = memory.PutWordMask(pa, data>>8, 0x000000ff)
		err2 := memory.PutWordMask(pa2, data<<24, 0xff000000)
		if err || err2 {
			return IRC_ADDR
		}
	}
	if err {
		return IRC_ADDR
	}
	return 0
}

/*
 * Update a byte in memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *CPU) writeByte(addr, data uint32) uint16 {
	var e uint16
	var pa uint32
	var err bool

	// Validate address
	pa, e = cpu.transAddr(addr)
	if e != 0 {
		return e
	}

	if cpu.checkProtect(pa, true) {
		return IRC_PROT
	}

	cpu.perCheck(addr)

	var mask uint32 = 0x000000ff

	var offset = 8 * (3 - (addr & 0x3))
	mem_cycle++
	if err = memory.PutWordMask(pa, data<<offset, mask<<offset); err {
		return IRC_ADDR
	}
	//	sim_debug(DEBUG_DATA, &cpu_dev, "WR A=%08x %02x\n", addr, data)
	return 0
}

// Suppress execution of instruction
func (cpu *CPU) suppress(code uint32, irc uint16) {
	irqaddr := cpu.storePSW(code, irc)

	// For IPL, save device after saving load complete
	if irqaddr == 0 {
		mem_cycle++
		_ = memory.PutWordMask(0, code, HMASK)
		mem_cycle++
		_ = memory.PutWordMask(0xba, code, HMASK)
	}
	mem_cycle++
	src1, _ := memory.GetWord(irqaddr)
	mem_cycle++
	src2, _ := memory.GetWord(irqaddr + 0x4)
	cpu.lpsw(src1, src2)
}

// Load new processor status double word
func (cpu *CPU) lpsw(src1, src2 uint32) {
	cpu.ecMode = (src1 & 0x00080000) != 0
	cpu.ext_en = (src1 & 0x01000000) != 0

	if cpu.ecMode {
		cpu.irq_en = (src1 & 0x02000000) != 0
		cpu.pageEnb = (src1 & 0x04000000) != 0
		cpu.cc = uint8((src1 >> 12) & 0x3)
		cpu.pmask = uint8((src1 >> 8) & 0xf)
		cpu.per_en = (src1 & 0x40000000) != 0
		if cpu.irq_en {
			cpu.sysMask = uint16(cpu.cregs[2] >> 16)
		} else {
			cpu.sysMask = 0
		}
	} else {
		cpu.sysMask = uint16((src1 >> 16) & 0xfc00)
		if (src1 & 0x2000000) != 0 {
			cpu.sysMask |= uint16((cpu.cregs[2] >> 16) & 0x3ff)
		}
		cpu.irq_en = cpu.sysMask != 0
		cpu.per_en = false
		cpu.cc = uint8((src2 >> 28) & 0x3)
		cpu.pmask = uint8((src2 >> 24) & 0xf)
		cpu.pageEnb = false
	}
	sys_channel.Irq_pending = true
	cpu.stKey = uint8((src1 >> 16) & 0xf0)
	cpu.flags = uint8((src1 >> 16) & 0x7)
	cpu.PC = src2 & AMASK
	//	sim_debug(DEBUG_INST, &cpu_dev, "PSW=%08x %08x  ", src1, src2)
	if cpu.ecMode && ((src1&0xb800c0ff) != 0 || (src2&0xff000000) != 0) {
		cpu.suppress(OPPSW, IRC_SPEC)
	}
}

// Load register pair into 64 bit integer
func (cpu *CPU) loadDouble(r uint8) uint64 {
	t := (uint64(cpu.regs[r]) << 32) | uint64(cpu.regs[r|1])
	return t
}

// Store a 64 bit integer in register pair
func (cpu *CPU) storeDouble(r uint8, v uint64) {
	cpu.regs[r|1] = uint32(v & LMASKL)
	cpu.regs[r] = uint32((v >> 32) & LMASKL)
	cpu.per_mod |= 3 << r
}

//
// /* Reset */

// t_stat
// cpu_reset (DEVICE *dptr)
// {
//     int     i;

//     /* Make sure devices are mapped correctly */
//     chan_set_devs();
//     sim_vm_fprint_stopped = &cpu_fprint_stopped;
//     /* Create memory array if it does not exist. */
//     if (M == NULL) {                        /* first time init? */
//         sim_brk_types = sim_brk_dflt = SWMASK ('E');
//         M = (uint32 *) calloc (((uint32) MEMSIZE) >> 2, sizeof (uint32));
//         if (M == NULL)
//             return SCPE_MEM;
//     }
//     /* Set up channels */
//     chan_set_devs();

//     sysmsk = irqcode = irqaddr = loading = 0;
//     st_key = cc = pmsk = ec_mode = interval_irq = flags = 0;
//     page_en = irq_en = ext_en = per_en = 0;
//     clk_state = CLOCK_UNSET;
//     for (i = 0; i < 256; i++)
//        tlb[i] = 0;
//     for (i = 0; i < 4096; i++)
//        key[i] = 0;
//     for (i = 0; i < 16; i++)
//        cregs[i] = 0;
//     clk_cmp[0] = clk_cmp[1] = 0xffffffff;
//     if (Q370) {
//         if (clk_state == CLOCK_UNSET) {
//             /* Set TOD to current time */
//             time_t seconds = sim_get_time(NULL);
//             t_uint64  lsec = (t_uint64)seconds;
//             /* IBM measures time from 1900, Unix starts at 1970 */
//             /* Add in number of years from 1900 to 1970 + 17 leap days */
//             lsec += ((70 * 365) + 17) * 86400ULL;
//             lsec *= 1000000ULL;
//             lsec <<= 12;
//             tod_clock[0] = (uint32)(lsec >> 32);
//             tod_clock[1] = (uint32)(lsec & FMASK);
//             clk_state = CLOCK_SET;
//         }
//         cregs[0]  = 0x000000e0;
//         cregs[2]  = 0xffffffff;
//         cregs[14] = 0xc2000000;
//         cregs[15] = 512;
//     }

//     if (cpu_unit[0].flags & (FEAT_370|FEAT_TIMER)) {
//        sim_rtcn_init_unit (&cpu_unit[0], 1000, TMR_RTC);
//        sim_activate(&cpu_unit[0], 100);
//     }
//     idle_stop_tm0 = 0;
//     return SCPE_OK;
// }

// /* Interval timer routines */
// t_stat
// rtc_srv(UNIT * uptr)
// {
//     (void)sim_rtcn_calb (rtc_tps, TMR_RTC);
//     sim_activate_after(uptr, 1000000/rtc_tps);
//     M[0x50>>2] -= 0x100;
//     if ((M[0x50>>2] & 0xfffff00) == 0)  {
//         sim_debug(DEBUG_INST, &cpu_dev, "TIMER IRQ %08x\n", M[0x50>>2]);
//         interval_irq = 1;
//     }
//     key[0] |= 0x6;
//     sim_debug(DEBUG_INST, &cpu_dev, "TIMER = %08x\n", M[0x50>>2]);
//     /* Time of day clock and timer on IBM 370 */
//     if (Q370) {
//         uint32 t;
//         if (clk_state && (cregs[0] & 0x20000000) == 0) {
//            t = tod_clock[1] + (13333333);
//            if (t < tod_clock[1])
//                 tod_clock[0]++;
//            tod_clock[1] = t;
//            sim_debug(DEBUG_INST, &cpu_dev, "TOD = %08x %08x\n", tod_clock[0], tod_clock[1]);
//            check_tod_irq();
//         }
//         t = cpu_timer[1] - (timer_tics << 12);
//         if (t > cpu_timer[1])
//             cpu_timer[0]--;
//         cpu_timer[1] = t;
//         sim_debug(DEBUG_INST, &cpu_dev, "INTER = %08x %08x\n", cpu_timer[0], cpu_timer[1]);
//         timer_tics = 3333;
//         if (cpu_timer[0] & MSIGN) {
//             sim_debug(DEBUG_INST, &cpu_dev, "CPU TIMER IRQ %08x%08x\n", cpu_timer[0],
//               cpu_timer[1]);
//             clk_irq = 1;
//         }
//     }
//     return SCPE_OK;
// }

// void
// check_tod_irq()
// {
//     tod_irq = 0;
//     if ((clk_cmp[0] < tod_clock[0]) ||
//        ((clk_cmp[0] == tod_clock[0]) && (clk_cmp[1] < tod_clock[1]))) {
//         sim_debug(DEBUG_INST, &cpu_dev, "CPU TIMER CCK IRQ %08x %08x\n", clk_cmp[0],
//                   clk_cmp[1]);
//         tod_irq = 1;
//     }
// }

// /* RSV: Set CPU IDLESTOP=<val>
//  *      <val>=number of seconds.
//  *
//  *      Sets max time in secounds CPU is IDLE but waiting for interrupt
//  *      from device. if <val> not zero, simulated CPU will wait for this wallclock
//  *      number of seconds, then stop. This allows to script a BOOT command and the
//  *      continue automatically when IPL has finished. Set to zero to disable.
//  */

// t_stat cpu_set_idle_stop (UNIT *uptr, int32 val, CONST char *cptr, void *desc)
// {
//     int32               n;
//     t_stat              r;

//     if (cptr == NULL) {
//         return SCPE_ARG;
//     }
//     n = (int32) get_uint(cptr, 10, 60, &r);
//     if (r != SCPE_OK) return SCPE_ARG;
//     idle_stop_msec = n * 1000;
//     idle_stop_tm0 = 0;
//     return SCPE_OK;
// }

// t_bool
// cpu_fprint_stopped (FILE *st, t_stat v)
// {
//     if (ec_mode) {
//         if (Q370)
//             fprintf(st, " PSW=%08x %08x\n",
//                (((uint32)page_en) << 26) | ((per_en) ? 1<<30:0) | ((irq_en) ? 1<<25:0) |
//                ((ext_en) ? 1<<24:0) | 0x80000 | (((uint32)st_key) << 16) |
//                (((uint32)flags) << 16) | (((uint32)cc) << 12) | (((uint32)pmsk) << 8), PC);
//         else
//             fprintf(st, " PSW=%08x %08x\n",
//                (((uint32)page_en) << 26) | ((irq_en) ? 1<<25:0) | ((ext_en) ? 1<<24:0) |
//                (((uint32)st_key) << 16) | (((uint32)flags) << 16) |
//                (((uint32)ilc) << 14) | (((uint32)cc) << 12) | (((uint32)pmsk) << 8), PC);
//     } else {
//         fprintf(st, " PSW=%08x %08x\n",
//             ((uint32)(ext_en) << 24) | (((uint32)sysmsk & 0xfe00) << 16) |
//             (((uint32)st_key) << 16) | (((uint32)flags) << 16) | ((uint32)irqcode),
//             (((uint32)ilc) << 30) | (((uint32)cc) << 28) | (((uint32)pmsk) << 24) | PC);
//     }
//     return FALSE;
// } */
