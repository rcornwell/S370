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

package cpu

import (
	"time"

	Dv "github.com/rcornwell/S370/emu/device"
	mem "github.com/rcornwell/S370/emu/memory"
	ch "github.com/rcornwell/S370/emu/sys_channel"
)

/*
   Introduced by IBM on Jun 30th, 1970. The IBM370 was an upgrade to the
   IBM 360, it added many new instruction and Dynamic Address Translation.

   The IBM 370 supported 32 bit memory and 16 32 bit registers. Optionally
   it could have 4 64 bit floating point registers. Optionally the machine
   could also process packed decimal numbers directly. There was also a
   64 bit processor status word. Up to 16MB of memory could be supported.

   Instructions ranged from 2 bytes to up to 6 bytes. In the following formats:
   Address are a 12 bit offset and one or two index registers. Index register
   0 results in a zero value.

    RR format:  (Register to Register).

      +----+----+----+----+
      |   op    | R1 | R2 |
      +----+----+----+----+
       * R1 could be register or 4 bit mask.

    RX format:  (Memory to Register).
      +----+----+----+----+----+----+----+----+
      |   op    | R1 | B2 | D2 |   Offset2    |
      +----+----+----+----+----+----+----+----+

    RS format:  (Memory to Register).
      +----+----+----+----+----+----+----+----+
      |   op    | R1 | R3 | D2 |   Offset2    |
      +----+----+----+----+----+----+----+----+
       * R3 could be register or 4 bit mask.

    SI format:  (Immediate to Memory).
      +----+----+----+----+----+----+----+----+
      |   op    |  Immed  | D1 |   Offset1    |
      +----+----+----+----+----+----+----+----+

    SS format:  (Memory to Memory).
      +----+----+----+----+----+----+----+----+----+----+----+----+
      |   op    |  Length | D1 |   Offset1    | D2 |   Offset2    |
      +----+----+----+----+----+----+----+----+----+----+----+----+
        * Length could be either 1 byte or to 4 bit lengths.

*/

var cpuState cpu

var memCycle int

// Initialize CPU to basic state.
func InitializeCPU() {
	cpuState.createTable()
	cpuState.PC = 0
	cpuState.sysMask = 0
	cpuState.stKey = 0
	cpuState.cc = 0
	cpuState.ilc = 0
	cpuState.progMask = 0
	cpuState.flags = 0
	cpuState.perRegMod = 0
	cpuState.perAddr = 0
	cpuState.perCode = 0
	cpuState.clkCmp[0] = FMASK
	cpuState.clkCmp[1] = FMASK
	cpuState.timerTics = 0
	cpuState.cpuTimer[0] = 0
	cpuState.cpuTimer[1] = 0
	cpuState.perEnb = false
	cpuState.ecMode = false
	cpuState.pageEnb = false
	cpuState.irqEnb = false
	cpuState.extEnb = false
	cpuState.extIrq = false
	cpuState.intIrq = false
	cpuState.intEnb = false
	cpuState.todEnb = false
	cpuState.todIrq = false
	cpuState.vmEnb = false

	// Clear registers
	for i := range 16 {
		cpuState.regs[i] = 0
		cpuState.cregs[i] = 0
	}

	// Initialize Control regisers to default
	cpuState.cregs[0] = 0x000000e0
	cpuState.cregs[2] = 0xffffffff
	cpuState.cregs[14] = 0xc2000000
	cpuState.cregs[15] = 512

	// Clear floating point registers
	for i := range 8 {
		cpuState.fpregs[i] = 0
	}

	// Clear TBL tables
	for i := range 256 {
		cpuState.tlb[i] = 0
	}

	// Set clock to current time
	if !cpuState.todSet {
		// Set TOD to current time
		now := time.Now()
		sec := now.Unix()

		// IBM measures time from 1900, Unix starts at 1970
		// Add in number of years from 1900 to 1970 + 17 leap days
		sec += ((70 * 365) + 17) * 86400
		sec *= 1000000
		sec <<= 12
		usec := uint64(sec)
		cpuState.todClock[0] = uint32((usec >> 32) & LMASKL)
		cpuState.todClock[1] = uint32(usec & LMASKL)
		cpuState.todSet = true
	}

	cpuState.pageMask = 0
}

// Execute one instruction or take an interrupt.
func CycleCPU() int {
	var err uint16
	memCycle = 0

	// Check if we should see if an IRQ is pending
	irq := ch.ChanScan(cpuState.sysMask, cpuState.irqEnb)
	if irq != Dv.NoDev {
		cpuState.ilc = 0
		if ch.Loading == Dv.NoDev {
			cpuState.suppress(oIOPSW, irq)
		}
		return memCycle
	}

	// Check for external interrupts
	if cpuState.extEnb {
		if cpuState.extIrq {
			if !cpuState.ecMode || (cpuState.cregs[0]&0x20) != 0 ||
				(cpuState.cregs[6]&0x40) != 0 {
				cpuState.extIrq = false
				cpuState.suppress(oEPSW, 0x40)
				return memCycle
			}
		}

		if cpuState.intIrq && (cpuState.cregs[0]&0x80) != 0 {
			cpuState.intIrq = false
			cpuState.suppress(oEPSW, 0x80)
			return memCycle
		}
		if cpuState.clkIrq && cpuState.intEnb {
			cpuState.clkIrq = false
			cpuState.suppress(oEPSW, 0x1005)
			return memCycle
		}
		if cpuState.todIrq && cpuState.todEnb {
			cpuState.todIrq = false
			cpuState.suppress(oEPSW, 0x1004)
			return memCycle
		}
	}

	/* If we have wait flag or loading, nothing more to do */
	if ch.Loading != Dv.NoDev || (cpuState.flags&wait) != 0 {
		/* CPU IDLE */
		if !cpuState.irqEnb && !cpuState.extEnb {
			return memCycle
		}
		return memCycle
	}

	if (cpuState.PC & 1) != 0 {
		cpuState.suppress(oPPSW, ircSpec)
		return memCycle
	}

	// Check if triggered PER event.
	if cpuState.perEnb && cpuState.perFetch {
		cpuState.perAddrCheck(cpuState.PC, 0x4000)
	}

	var opr, t uint32
	var step stepInfo

	// Fetch the next instruction
	t, err = cpuState.readFull(cpuState.PC & ^uint32(0x2))
	if err != 0 {
		cpuState.suppress(oPPSW, err)
		return memCycle
	}

	// Save instruction
	if (cpuState.PC & 2) == 0 {
		opr = (t >> 16) & 0xffff
	} else {
		opr = t & 0xffff
	}
	cpuState.perRegMod = 0
	cpuState.perCode = 0
	cpuState.perAddr = cpuState.PC
	cpuState.iPC = cpuState.PC
	cpuState.ilc = 1
	step.opcode = uint8((opr >> 8) & 0xff)
	step.reg = uint8(opr & 0xff)
	step.R1 = (step.reg >> 4) & 0xf
	step.R2 = step.reg & 0xf
	cpuState.PC += 2

	// Check type of instruction
	if (step.opcode & 0xc0) != 0 {
		// RX, RS, SI, SS
		cpuState.ilc++
		// Check if we need new word?
		if (cpuState.PC & 2) == 0 {
			t, err = cpuState.readFull(cpuState.PC & ^uint32(0x2))
			if err != 0 {
				cpuState.suppress(oPPSW, err)
				return memCycle
			}
			step.address1 = (t >> 16)
		} else {
			step.address1 = t
		}
		step.address1 &= 0xffff
		cpuState.PC += 2
	}

	// If SS
	if (step.opcode & 0xc0) == 0xc0 {
		cpuState.ilc++
		// Do we need another word?
		if (cpuState.PC & 2) == 0 {
			t, err = cpuState.readFull(cpuState.PC & ^uint32(0x2))
			if err != 0 {
				cpuState.suppress(oPPSW, err)
				return memCycle
			}
			step.address2 = (t >> 16)
		} else {
			step.address2 = t
		}
		step.address2 &= 0xffff
		cpuState.PC += 2
	}

	err = cpuState.execute(&step)
	if err != 0 {
		cpuState.suppress(oPPSW, err)
	}

	// See if PER event happened
	if cpuState.perEnb && cpuState.perCode != 0 {
		cpuState.suppress(oPPSW, 0)
	}
	return memCycle
}

// Generate addresses for operands and if
// approperate fetch the values. Then execute the
// instruction and return any error condition.
func (cpu *cpu) execute(step *stepInfo) uint16 {
	// Compute addresses of operands
	if (step.opcode & 0xc0) != 0 { // RS, RX, SS
		temp := (step.address1 >> 12) & 0xf
		step.address1 &= 0xfff
		if temp != 0 {
			step.address1 += cpu.regs[temp]
		}
		step.address1 &= AMASK
		step.src1 = step.address1

		// Handle RX type operands
		if (step.opcode & 0x80) == 0 {
			if step.R2 != 0 {
				step.address1 += cpu.regs[step.R2]
			}
		} else if (step.opcode & 0xc0) == 0xc0 { // SS
			temp = (step.address2 >> 12) & 0xf
			step.address2 &= 0xfff
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
			return ircSpec
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
				return ircSpec
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
		if (step.opcode&0xfc) == 0x48 || step.opcode == OpMH {
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
			step.src2 = step.address1
		}
	}

	// Execute the instruction.
	err = cpu.table[step.opcode](step)
	if cpu.perEnb && cpu.perReg && (cpu.cregs[9]&0xffff&cpu.perRegMod) != 0 {
		cpu.perCode |= 0x1000
	}

	return err
}

// Create function table.
func (cpu *cpu) createTable() {
	cpu.table = [256]func(*stepInfo) uint16{
		//  0         1         2         3          4         5         6          7
		cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opSPM, cpu.opBAL, cpu.opBCT, cpu.opBC, // 0x
		//  8         9         A         B          C         D         E          F
		cpu.opSSK, cpu.opISK, cpu.opSVC, cpu.opUnk, cpu.opUnk, cpu.opBAS, cpu.opMVCL, cpu.opCLCL,

		cpu.opLPR, cpu.opLNR, cpu.opLTR, cpu.opLCR, cpu.opAnd, cpu.opCmpL, cpu.opOr, cpu.opXor, // 1x
		cpu.opL, cpu.opCmp, cpu.opAdd, cpu.opSub, cpu.opMul, cpu.opDiv, cpu.opAddL, cpu.opSubL,

		cpu.opLcs, cpu.opLcs, cpu.opLcs, cpu.opLcs, cpu.opFPHalf, cpu.opLRDR, cpu.opMXR, cpu.opMXD, // 2x
		cpu.opFPLoad, cpu.opCD, cpu.opFPAddD, cpu.opFPAddD, cpu.opFPMul, cpu.opFPDiv, cpu.opFPAddD, cpu.opFPAddD,

		cpu.opLcs, cpu.opLcs, cpu.opLcs, cpu.opLcs, cpu.opFPHalf, cpu.opLRER, cpu.opAXR, cpu.opAXR, // 3x
		cpu.opFPLoad, cpu.opCE, cpu.opFPAdd, cpu.opFPAdd, cpu.opFPMul, cpu.opFPDiv, cpu.opFPAdd, cpu.opFPAdd,

		cpu.opSTH, cpu.opL, cpu.opSTC, cpu.opIC, cpu.opEX, cpu.opBAL, cpu.opBCT, cpu.opBC, // 4x
		cpu.opL, cpu.opCmp, cpu.opAdd, cpu.opSub, cpu.opMulH, cpu.opBAS, cpu.opCVD, cpu.opCVB,

		cpu.opST, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opAnd, cpu.opCmpL, cpu.opOr, cpu.opXor, // 5x
		cpu.opL, cpu.opCmp, cpu.opAdd, cpu.opSub, cpu.opMul, cpu.opDiv, cpu.opAddL, cpu.opSubL,

		cpu.opSTD, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opMXD, // 6x
		cpu.opFPLoad, cpu.opCD, cpu.opFPAddD, cpu.opFPAddD, cpu.opFPMul, cpu.opFPDiv, cpu.opFPAddD, cpu.opFPAddD,

		cpu.opSTE, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, // 7x
		cpu.opFPLoad, cpu.opCE, cpu.opFPAdd, cpu.opFPAdd, cpu.opFPMul, cpu.opFPDiv, cpu.opFPAdd, cpu.opFPAdd,

		cpu.opSSM, cpu.opUnk, cpu.opLPSW, cpu.opDIAG, cpu.opUnk, cpu.opUnk, cpu.opBXH, cpu.opBXLE, // 8x
		cpu.opSRL, cpu.opSLL, cpu.opSRA, cpu.opSLA, cpu.opSRDL, cpu.opSLDL, cpu.opSRDA, cpu.opSLDA,

		cpu.opSTM, cpu.opTM, cpu.opMVI, cpu.opTS, cpu.opNI, cpu.opCLI, cpu.opOI, cpu.opXI, // 9x
		cpu.opLM, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opSIO, cpu.opTIO, cpu.opHIO, cpu.opTCH,

		cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, // Ax
		cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opSTxSM, cpu.opSTxSM, cpu.opSIGP, cpu.opMC,

		cpu.opUnk, cpu.opLRA, cpu.opB2, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opSTCTL, cpu.opLCTL, // Bx
		cpu.opUnk, cpu.opUnk, cpu.opCS, cpu.opCDS, cpu.opUnk, cpu.opCLM, cpu.opSTCM, cpu.opICM,

		cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, // Cx
		cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk,

		cpu.opUnk, cpu.opMem, cpu.opMem, cpu.opMem, cpu.opMem, cpu.opCLC, cpu.opMem, cpu.opMem, // Dx
		cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opTR, cpu.opTR, cpu.opED, cpu.opED,

		cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, // Ex
		cpu.opMVCIN, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk,

		cpu.opSRP, cpu.opMVO, cpu.opPACK, cpu.opUNPK, cpu.opUnk, cpu.opUnk, cpu.opUnk, cpu.opUnk, // Fx
		cpu.opDecAdd, cpu.opDecAdd, cpu.opDecAdd, cpu.opDecAdd, cpu.opMP, cpu.opDP, cpu.opUnk, cpu.opUnk,
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
func (cpu *cpu) transAddr(va uint32) (uint32, uint16) {
	var entry uint32
	var err bool

	// Check address in range
	addr := va & AMASK

	// If paging not enabled, return address.
	if !cpu.pageEnb {
		return addr, 0
	}

	// Extract page address is on
	page := addr >> cpu.pageShift

	// Extract segment and move it into place.
	seg := (page & 0x1f00) << 4

	// Only 256 pages.
	page &= 0xff

	// Quick check if TLB correct
	entry = cpu.tlb[page]
	if (entry&tlbValid) != 0 && ((entry^seg)&tlbSeg) == 0 {
		addr = (va & cpu.pageMask) | ((entry & tlbPhy) << cpu.pageShift)
		return addr, 0
	}

	// TLB entry does not match, replace it.
	// Clear whatever was in entry
	cpu.tlb[page] = 0
	// TLB not correct, try loading correct entry
	// Segment and page number to word address
	seg = (addr >> cpu.segShift) & cpu.segMask
	page = (addr >> cpu.pageShift) & cpu.pageIndex

	// Check address against length of segment table
	if seg > cpu.segLen {
		// segment above length of table,
		// write failed address and 90, then trigger trap.
		_ = mem.PutWord(0x90, va)
		memCycle++
		cpu.PC = cpu.iPC
		return 0, ircSeg
	}

	// Compute address of PTE table
	// Get pointer to page table
	addr = ((seg << 2) + cpu.segAddr) & AMASK

	// Get entry on error throw trap.
	memCycle++
	entry, err = mem.GetWord(addr)
	if err {
		return 0, ircAddr
	}

	// Extract length of Table pointer.
	addr = (entry >> 28) + 1

	/* Check if entry valid and in correct length */
	if (entry&pteValid) != 0 || (page>>cpu.pteLenShift) >= addr {
		memCycle++
		mem.SetMemory(0x90, va)
		cpu.PC = cpu.iPC
		if (entry & pteValid) != 0 {
			return 0, ircSeg
		}
		return 0, ircPage
	}

	// Now we need to fetch the actual entry
	addr = ((entry & pteAddr) + (page << 1)) & AMASK
	memCycle++
	entry, err = mem.GetWord(addr)
	if err {
		return 0, ircAddr
	}

	// extract actual PTE entry
	if (addr & 2) != 0 {
		entry = (addr >> 16)
	}
	entry &= 0xffff

	if (entry & cpu.pteMBZ) != 0 {
		memCycle++
		mem.SetMemory(0x90, va)
		cpu.PC = cpu.iPC
		return 0, ircSpec
	}

	// Check if entry valid and in correct length
	if (entry & cpu.pteAvail) != 0 {
		memCycle++
		mem.SetMemory(0x90, va)
		cpu.PC = cpu.iPC
		return 0, ircPage
	}

	// Compute correct entry
	entry >>= cpu.pteShift // Move physical to correct spot
	page = va >> cpu.pageShift
	entry = entry | ((page & 0x1f00) << 4) | tlbValid
	// Update TLB with new entry
	cpu.tlb[page&0xff] = entry
	// Compute physical address
	addr = (va & cpu.pageMask) | (((entry & tlbPhy) << cpu.pageShift) & AMASK)
	return addr, 0
}

// Store the PSW at given address with irq value.
func (cpu *cpu) storePSW(vector uint32, irqcode uint16) (irqaddr uint32) {
	var word1, word2 uint32
	irqaddr = vector + 0x40

	if vector == oPPSW && cpu.perEnb && cpu.perCode != 0 {
		irqcode |= ircPer
	}
	if cpu.ecMode {
		// Generate first word
		word1 = uint32(0x80000) |
			(uint32(cpu.stKey) << 16) |
			(uint32(cpu.flags) << 16) |
			(uint32(cpu.cc) << 12) |
			(uint32(cpu.progMask) << 8)
		if cpu.pageEnb {
			word1 |= 1 << 26
		}
		if cpu.perEnb {
			word1 |= 1 << 30
		}
		if cpu.irqEnb {
			word1 |= 1 << 25
		}
		if cpu.extEnb {
			word1 |= 1 << 24
		}

		// Save code where 370 expects it to be
		switch vector {
		case oEPSW:
			memCycle++
			mem.SetMemoryMask(0x84, uint32(irqcode), LMASK)
		case oSPSW:
			memCycle++
			mem.SetMemory(0x88, ((uint32(cpu.ilc) << 17) | uint32(irqcode)))
		case oPPSW:
			memCycle++
			mem.SetMemory(0x8c, ((uint32(cpu.ilc) << 17) | uint32(irqcode)))
		case oIOPSW:
			memCycle++
			mem.SetMemory(0xb8, uint32(irqcode))
		}
		if (irqcode & ircPer) != 0 {
			memCycle++
			mem.SetMemory(150, (uint32(cpu.perCode)<<16)|(cpu.perAddr>>16))
			memCycle++
			mem.SetMemoryMask(154, (cpu.perAddr&0xffff)<<16, LMASK)
		}
		// Generate second word.
		word2 = cpu.PC
	} else {
		// Generate first word.
		word1 = (uint32(cpu.sysMask&0xfe00) << 16) |
			(uint32(cpu.stKey) << 16) |
			(uint32(cpu.flags) << 16) |
			uint32(irqcode)
		if cpu.extEnb {
			word1 |= 1 << 24
		}
		// Generate second word. */
		word2 = (uint32(cpu.ilc) << 30) |
			(uint32(cpu.cc) << 28) |
			(uint32(cpu.progMask) << 24) |
			(cpu.PC & AMASK)
	}
	memCycle++
	mem.SetMemory(vector, word1)
	memCycle++
	mem.SetMemory(vector+4, word2)
	//	sim_debug(DEBUG_INST, &cpu_dev, "store %02x %d %x %03x PSW=%08x %08x\n", addr, ilc,
	//		cc, ircode, word, word2)
	return irqaddr
}

// Check for protection violation.
func (cpu *cpu) checkProtect(addr uint32, write bool) bool {
	/* Check storage key */
	if cpu.stKey == 0 {
		return false
	}
	k := mem.GetKey(addr)
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

// * Check if we can access a range of mem.
func (cpu *cpu) testAccess(va uint32, size uint32, write bool) uint16 {
	// Translate address
	pa, err := cpu.transAddr(va)
	if err != 0 {
		return err
	}
	if cpu.checkProtect(pa, write) {
		return ircProt
	}

	if size != 0 && (va&SPMASK) != ((va+size)&SPMASK) {
		// Translate end address
		pa, err := cpu.transAddr(va + size)
		if err != 0 {
			return err
		}
		if cpu.checkProtect(pa, write) {
			return ircProt
		}
	}
	return 0
}

/*
 * Read a full word from memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *cpu) readFull(addr uint32) (uint32, uint16) {
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
		return 0, ircProt
	}

	// Read actual data
	memCycle++
	v, err = mem.GetWord(addr)
	if err {
		return 0, ircAddr
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
				return 0, ircProt
			}
		}

		memCycle++
		t, err := mem.GetWord(pa2)
		if err {
			return 0, ircAddr
		}
		v <<= (8 * offset)
		v |= (t >> (8 * (4 - offset)))
	}

	//	sim_debug(DEBUG_DATA, &cpu_dev, "RD A=%08x %08x\n", addr, *v)
	return v, 0
}

/*
 * Read a half word from memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *cpu) readHalf(addr uint32) (uint32, uint16) {
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
		return 0, ircProt
	}

	// Get data
	memCycle++
	v, err = mem.GetWord(pa)
	if err {
		return 0, ircAddr
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
				return 0, ircProt
			}
		}

		memCycle++
		if v2, err := mem.GetWord(pa2); err {
			return 0, ircAddr
		} else {
			v = (v & 0xff) << 8
			v |= (v2 >> 24) & 0xff
		}
	}

	// Sign extend the result
	v &= LMASK
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
func (cpu *cpu) readByte(addr uint32) (uint32, uint16) {
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
		return 0, ircProt
	}

	// Read actual data
	memCycle++
	v, err = mem.GetWord(addr)
	if err {
		return 0, ircAddr
	}

	v = (v >> (8 * (3 - offset))) & 0xff
	// sim_debug(DEBUG_DATA, &cpu_dev, "RD B=%08x %08x\n", addr, *v)
	return v, 0
}

func (cpu *cpu) perAddrCheck(addr uint32, code uint16) {
	if cpu.cregs[10] <= cpu.cregs[11] {
		if addr >= cpu.cregs[10] && addr <= cpu.cregs[11] {
			cpu.perCode |= code
		}
	} else {
		if addr >= cpu.cregs[11] || addr <= cpu.cregs[10] {
			cpu.perCode |= code
		}
	}
}

// Check if address is in the range of PER modify range.
func (cpu *cpu) perCheck(addr uint32) {
	if cpu.perEnb && cpu.perStore {
		cpu.perAddrCheck(addr, 0x2000)
	}
}

/*
 * Update a full word in memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *cpu) writeFull(addr, data uint32) uint16 {
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
		return ircProt
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
				return ircProt
			}
		}

		// Check if in storage area
		cpu.perCheck(addr2)
	}

	switch offset {
	case 0:
		memCycle++
		err1 = mem.PutWord(pa, data)
		err2 = false
	case 1:
		memCycle++
		err1 = mem.PutWordMask(pa, data>>8, 0x00ffffff)
		memCycle++
		err2 = mem.PutWordMask(pa2, data<<24, 0xff000000)
	case 2:
		memCycle++
		err1 = mem.PutWordMask(pa, data>>16, 0x0000ffff)
		memCycle++
		err2 = mem.PutWordMask(pa2, data<<16, 0xffff0000)
	case 3:
		memCycle++
		err1 = mem.PutWordMask(pa, data>>24, 0x000000ff)
		memCycle++
		err2 = mem.PutWordMask(pa2, data<<8, 0xffffff00)
	}

	if err1 || err2 {
		e = ircAddr
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
func (cpu *cpu) writeHalf(addr, data uint32) uint16 {
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
		return ircProt
	}

	cpu.perCheck(addr)

	switch offset {
	case 0:
		memCycle++
		err = mem.PutWordMask(pa, data<<16, 0xffff0000)
	case 1:
		memCycle++
		err = mem.PutWordMask(pa, data<<8, 0x00ffff00)
	case 2:
		memCycle++
		err = mem.PutWordMask(pa, data, LMASK)
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
				return ircProt
			}
		}

		memCycle++
		memCycle++
		err = mem.PutWordMask(pa, data>>8, 0x000000ff)
		err2 := mem.PutWordMask(pa2, data<<24, 0xff000000)
		if err || err2 {
			return ircAddr
		}
	}
	if err {
		return ircAddr
	}
	return 0
}

/*
 * Update a byte in memory, checking protection
 * and alignment restrictions. Return 1 if failure, 0 if
 * success.
 */
func (cpu *cpu) writeByte(addr, data uint32) uint16 {
	var e uint16
	var pa uint32
	var err bool

	// Validate address
	pa, e = cpu.transAddr(addr)
	if e != 0 {
		return e
	}

	if cpu.checkProtect(pa, true) {
		return ircProt
	}

	cpu.perCheck(addr)

	var mask uint32 = 0x000000ff

	offset := 8 * (3 - (addr & 0x3))
	memCycle++
	if err = mem.PutWordMask(pa, data<<offset, mask<<offset); err {
		return ircAddr
	}
	//	sim_debug(DEBUG_DATA, &cpu_dev, "WR A=%08x %02x\n", addr, data)
	return 0
}

// Suppress execution of instruction.
func (cpu *cpu) suppress(code uint32, irc uint16) {
	irqaddr := cpu.storePSW(code, irc)

	// For IPL, save device after saving load complete
	if irqaddr == 0 {
		memCycle++
		_ = mem.PutWordMask(0, code, LMASK)
		memCycle++
		_ = mem.PutWordMask(0xba, code, LMASK)
	}
	memCycle++
	src1, _ := mem.GetWord(irqaddr)
	memCycle++
	src2, _ := mem.GetWord(irqaddr + 0x4)
	cpu.lpsw(src1, src2)
}

// Load new processor status double word.
func (cpu *cpu) lpsw(src1, src2 uint32) {
	cpu.ecMode = (src1 & 0x00080000) != 0
	cpu.extEnb = (src1 & 0x01000000) != 0

	if cpu.ecMode {
		cpu.irqEnb = (src1 & 0x02000000) != 0
		cpu.pageEnb = (src1 & 0x04000000) != 0
		cpu.cc = uint8((src1 >> 12) & 0x3)
		cpu.progMask = uint8((src1 >> 8) & 0xf)
		cpu.perEnb = (src1 & 0x40000000) != 0
		cpu.sysMask = 0
		if cpu.irqEnb {
			cpu.sysMask = uint16(cpu.cregs[2] >> 16)
		}
	} else {
		cpu.sysMask = uint16((src1 >> 16) & 0xfc00)
		if (src1 & 0x2000000) != 0 {
			cpu.sysMask |= uint16((cpu.cregs[2] >> 16) & 0x3ff)
		}
		cpu.irqEnb = cpu.sysMask != 0
		cpu.perEnb = false
		cpu.cc = uint8((src2 >> 28) & 0x3)
		cpu.progMask = uint8((src2 >> 24) & 0xf)
		cpu.pageEnb = false
	}
	ch.IrqPending = true
	cpu.stKey = uint8((src1 >> 16) & 0xf0)
	cpu.flags = uint8((src1 >> 16) & 0x7)
	cpu.PC = src2 & AMASK
	//	sim_debug(DEBUG_INST, &cpu_dev, "PSW=%08x %08x  ", src1, src2)
	if cpu.ecMode && ((src1&0xb800c0ff) != 0 || (src2&0xff000000) != 0) {
		cpu.suppress(oPPSW, ircSpec)
	}
}

// Load register pair into 64 bit integer.
func (cpu *cpu) loadDouble(r uint8) uint64 {
	t := (uint64(cpu.regs[r]) << 32) | uint64(cpu.regs[r|1])
	return t
}

// Store a 64 bit integer in register pair.
func (cpu *cpu) storeDouble(r uint8, v uint64) {
	cpu.regs[r|1] = uint32(v & LMASKL)
	cpu.regs[r] = uint32((v >> 32) & LMASKL)
	cpu.perRegMod |= 3 << r
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
