/* IBM 370 Standard instruction execution

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

// Handle an unknown instruction.
func (cpu *cpu) opUnk(_ *stepInfo) uint16 {
	return ircOper
}

// Set program mask.
func (cpu *cpu) opSPM(step *stepInfo) uint16 {
	cpu.progMask = uint8(step.src1>>24) & 0xf
	cpu.cc = uint8(step.src1>>28) & 0x3
	return 0
}

// Branch and save.
func (cpu *cpu) opBAS(step *stepInfo) uint16 {
	dest := cpu.PC
	if step.opcode != OpBASR || step.R2 != 0 {
		// Check if triggered PER event.
		if cpu.perEnb && cpu.perBranch {
			cpu.perCode |= 0x8000 // Set PER branch
		}
		cpu.PC = step.address1 & AMASK
	}
	cpu.regs[step.R1] = dest
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Branch and link.
func (cpu *cpu) opBAL(step *stepInfo) uint16 {
	// Set up return register, flags are in upper byte
	dest := (uint32(cpu.ilc) << 30) |
		(uint32(cpu.cc) << 28) |
		(uint32(cpu.progMask) << 24) |
		cpu.PC
	if step.opcode != OpBALR || step.R2 != 0 {
		// Check if triggered PER event.
		if cpu.perEnb && cpu.perBranch {
			cpu.perCode |= 0x8000 // Set PER branch
		}
		cpu.PC = step.address1 & AMASK
	}
	cpu.regs[step.R1] = dest
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Branch and count register.
func (cpu *cpu) opBCT(step *stepInfo) uint16 {
	dest := step.src1 - 1

	if dest != 0 && (step.opcode != OpBCTR || step.R2 != 0) {
		// Check if triggered PER event.
		if cpu.perEnb && cpu.perBranch {
			cpu.perCode |= 0x8000
		}
		cpu.PC = step.address1 & AMASK
	}
	cpu.regs[step.R1] = dest
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Branch conditional.
func (cpu *cpu) opBC(step *stepInfo) uint16 {
	if ((0x8>>cpu.cc)&step.R1) != 0 && (step.opcode != OpBCR || step.R2 != 0) {
		// Check if triggered PER event.
		if cpu.perEnb && cpu.perBranch {
			cpu.perCode |= 0x8000
		}
		cpu.PC = step.address1 & AMASK
	}
	return 0
}

// Branch on index high.
func (cpu *cpu) opBXH(step *stepInfo) uint16 {
	src1 := cpu.regs[step.R2|1]
	dest := cpu.regs[step.R1] + cpu.regs[step.R2]
	cpu.regs[step.R1] = dest
	cpu.perRegMod |= 1 << step.R1
	if int32(dest) > int32(src1) {
		// Check if triggered PER event.
		if cpu.perEnb && cpu.perBranch {
			cpu.perCode |= 0x8000
		}
		cpu.PC = step.address1 & AMASK
	}
	return 0
}

// Branch of index low or equal.
func (cpu *cpu) opBXLE(step *stepInfo) uint16 {
	src1 := cpu.regs[step.R2|1]
	dest := cpu.regs[step.R1] + cpu.regs[step.R2]
	cpu.regs[step.R1] = dest
	cpu.perRegMod |= 1 << step.R1
	if int32(dest) <= int32(src1) {
		// Check if triggered PER event.
		if cpu.perEnb && cpu.perBranch {
			cpu.perCode |= 0x8000
		}
		cpu.PC = step.address1 & AMASK
	}
	return 0
}

// Set the condition code based on value provided.
func (cpu *cpu) setCC(value uint32) {
	if (value & MSIGN) != 0 {
		cpu.cc = 1
	} else {
		if value == 0 {
			cpu.cc = 0
		} else {
			cpu.cc = 2
		}
	}
}

// Load register and make positive.
func (cpu *cpu) opLPR(step *stepInfo) uint16 {
	if step.src2 == MSIGN { // Overflow
		cpu.cc = 3
	} else if (step.src2 & MSIGN) != 0 {
		step.src2 = (FMASK ^ step.src2) + 1
		cpu.setCC(step.src2)
	} else {
		cpu.setCC(step.src2)
	}
	cpu.regs[step.R1] = step.src2
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Load complement of register.
func (cpu *cpu) opLCR(step *stepInfo) uint16 {
	if step.src2 != MSIGN {
		step.src2 = (FMASK ^ step.src2) + 1
		cpu.setCC(step.src2)
	} else {
		cpu.cc = 3
	}
	cpu.regs[step.R1] = step.src2
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Load and test register.
func (cpu *cpu) opLTR(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src2
	cpu.perRegMod |= 1 << step.R1
	cpu.setCC(step.src2)
	return 0
}

// Load number and make it negative.
func (cpu *cpu) opLNR(step *stepInfo) uint16 {
	if (step.src2 & MSIGN) == 0 {
		step.src2 = (FMASK ^ step.src2) + 1
	}
	cpu.regs[step.R1] = step.src2
	cpu.perRegMod |= 1 << step.R1
	cpu.setCC(step.src2)
	return 0
}

// Load value into register, handle RR and RX.
// Also handle LH and LA.
func (cpu *cpu) opL(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src2
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Load character into register.
func (cpu *cpu) opIC(step *stepInfo) uint16 {
	by, err := cpu.readByte(step.address1)
	if err != 0 {
		return err
	}
	cpu.regs[step.R1] = (step.src1 & 0xffffff00) | (by & 0xff)
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Insert character into register under mask.
func (cpu *cpu) opICM(step *stepInfo) uint16 {
	// Fetch register
	reg := cpu.regs[step.R1]
	bits := step.R2
	cpu.cc = 0
	// If no bits are set, read first byte to check for trap
	if bits == 0 {
		_, err := cpu.readByte(step.address1)
		return err
	}

	// Set flag to check first bit
	high := uint32(0x80)
	mask := uint32(0xff000000)
	shift := 24 // Shift count

	// Scan from bit 12 to bit 15
	for i := uint8(0x8); i != 0; i >>= 1 {
		// If the bit is one, read in the byte
		if (bits & i) != 0 {
			by, err := cpu.readByte(step.address1)
			if err != 0 {
				return err
			}

			// Put byte in place
			reg = (reg & ^mask) | ((by << shift) & mask)

			// If byte is not zero, adjust CC
			if by != 0 {
				if (by & high) != 0 {
					cpu.cc = 1
				}
				if cpu.cc == 0 {
					cpu.cc = 2
				}
			}
			high = 0 // no need to check after first byte
			step.address1++
		}
		shift -= 8
		mask >>= 8
	}
	cpu.regs[step.R1] = reg
	cpu.perRegMod |= 1 << step.reg
	return 0
}

// Add value to register, handle RR, RX and RH.
func (cpu *cpu) opAdd(step *stepInfo) uint16 {
	sum := step.src1 + step.src2
	carry := (step.src1 & step.src2) | ((step.src1 ^ step.src2) & ^sum)
	cpu.regs[step.R1] = sum
	cpu.perRegMod |= 1 << step.R1
	if (((carry << 1) ^ carry) & MSIGN) != 0 {
		cpu.cc = 3
		if (cpu.progMask & FIXOVER) != 0 {
			return ircFixOver
		}
		return 0
	}
	cpu.setCC(sum)
	return 0
}

// Subtract value from register, handle RR, RX and RH.
func (cpu *cpu) opSub(step *stepInfo) uint16 {
	src2 := step.src2 ^ FMASK
	diff := step.src1 + src2 + 1
	carry := (step.src1 & src2) | ((step.src1 ^ src2) & ^diff)
	cpu.regs[step.R1] = diff
	cpu.perRegMod |= 1 << step.R1
	if (((carry << 1) ^ carry) & MSIGN) != 0 {
		cpu.cc = 3
		if (cpu.progMask & FIXOVER) != 0 {
			return ircFixOver
		}
		return 0
	}
	cpu.setCC(diff)
	return 0
}

// Compare register with value, handle RR, RX and RH.
func (cpu *cpu) opCmp(step *stepInfo) uint16 {
	if int32(step.src1) > int32(step.src2) {
		cpu.cc = 2
		return 0
	}
	if step.src1 != step.src2 {
		cpu.cc = 1
		return 0
	}
	cpu.cc = 0
	return 0
}

// Logical add value to register, handle RR, RX.
func (cpu *cpu) opAddL(step *stepInfo) uint16 {
	sum := step.src1 + step.src2
	cpu.regs[step.R1] = sum
	isum := FMASK ^ sum
	carry := (step.src1 & step.src2) | ((step.src1 ^ step.src2) & isum)
	cpu.perRegMod |= 1 << step.R1
	if sum != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	if (carry & MSIGN) != 0 {
		cpu.cc |= 2
	}
	return 0
}

// Subtract logical value to register, handle RR, RX.
func (cpu *cpu) opSubL(step *stepInfo) uint16 {
	src2 := step.src2 ^ FMASK
	sum := step.src1 + src2 + 1
	carry := (step.src1 & src2) | ((step.src1 ^ src2) & ^sum)
	if sum != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	if (carry & MSIGN) != 0 {
		cpu.cc |= 2
	}
	cpu.regs[step.R1] = sum
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Logical compare with register.
func (cpu *cpu) opCmpL(step *stepInfo) uint16 {
	if step.src1 > step.src2 {
		cpu.cc = 2
		return 0
	}
	if step.src1 != step.src2 {
		cpu.cc = 1
		return 0
	}
	cpu.cc = 0
	return 0
}

// Compare character under mask.
func (cpu *cpu) opCLM(step *stepInfo) uint16 {
	// Fetch register
	reg := cpu.regs[step.R1]
	bits := step.R2
	cpu.cc = 0
	// If no bits are set, read first byte to check for trap
	if bits == 0 {
		_, err := cpu.readByte(step.address1)
		return err
	}

	shift := 24 // Shift count

	// Scan from bit 12 to bit 15
	for i := uint8(0x8); i != 0; i >>= 1 {
		// If the bit is one, read in the byte
		if (bits & i) != 0 {
			by, err := cpu.readByte(step.address1)
			if err != 0 {
				return err
			}
			// Put byte in place
			cmpBy := (reg >> shift) & 0xff

			// Compare values
			if cmpBy != by {
				if cmpBy < by {
					cpu.cc = 1
				} else {
					cpu.cc = 2
				}
				return 0
			}
			step.address1++
		}
		shift -= 8
	}
	return 0
}

// Multiply register by value, handle RR and RX.
func (cpu *cpu) opMul(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return ircSpec
	}
	prod := int64(int32(cpu.regs[step.R1|1]))
	prod *= int64(int32(step.src2))
	cpu.storeDouble(step.R1, uint64(prod))
	return 0
}

// Multiply half word.
func (cpu *cpu) opMulH(step *stepInfo) uint16 {
	prod := int64(int32(cpu.regs[step.R1]))
	prod *= int64(int32(step.src2))
	cpu.regs[step.R1] = uint32(uint64(prod) & LMASKL)
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Divide register by value, handle RR and RX.
func (cpu *cpu) opDiv(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return ircSpec
	}
	if step.src2 == 0 {
		return ircFixDiv
	}
	var sign1 bool
	var sign2 bool
	divdendLow := cpu.regs[step.R1]
	divdenHigh := cpu.regs[step.R1|1]

	// Make dividend positive
	if (divdendLow & MSIGN) != 0 {
		sign1 = true
		divdenHigh ^= FMASK
		divdendLow ^= FMASK
		if divdenHigh == FMASK {
			divdendLow++
		}
		divdenHigh++
	}

	// Make divisor positive
	if (step.src2 & MSIGN) != 0 {
		sign2 = true
		step.src2 = (step.src2 ^ FMASK) + 1
	}

	var result uint32
	result = 0
	// Do actual divide
	for range 32 {
		// Shift left by one
		divdendLow <<= 1
		if (divdenHigh & MSIGN) != 0 {
			divdendLow |= 1
		}
		divdenHigh <<= 1
		temp := divdendLow - step.src2
		result <<= 1
		if (temp & MSIGN) == 0 {
			divdendLow = temp
			result |= 1
		}
	}

	// Check for overflow
	if (result&MSIGN) != 0 && result != MSIGN {
		return ircFixDiv
	}

	// If signs are different result is negative
	if sign1 != sign2 {
		result = (result ^ FMASK) + 1
	}

	// Remember sign same as divendend
	if sign1 {
		divdendLow = (divdendLow ^ FMASK) + 1
	}
	cpu.regs[step.R1] = divdendLow
	cpu.regs[step.R1|1] = result
	cpu.perRegMod |= 3 << step.R1
	return 0
}

// Logical And register with value, handle RR and RX.
func (cpu *cpu) opAnd(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src1 & step.src2
	cpu.perRegMod |= 1 << step.R1
	if cpu.regs[step.R1] != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical Or register with value, handle RR and RX.
func (cpu *cpu) opOr(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src1 | step.src2
	cpu.perRegMod |= 1 << step.R1
	if cpu.regs[step.R1] != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical Exclusive Or register with value, handle RR and RX.
func (cpu *cpu) opXor(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src1 ^ step.src2
	cpu.perRegMod |= 1 << step.R1
	if cpu.regs[step.R1] != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical And immeditate to memory.
func (cpu *cpu) opNI(step *stepInfo) uint16 {
	by, err := cpu.readByte(step.address1)
	if err != 0 {
		return err
	}
	by &= uint32(step.reg)
	if err = cpu.writeByte(step.address1, by); err != 0 {
		return err
	}
	if by != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical Or immediate to memory.
func (cpu *cpu) opOI(step *stepInfo) uint16 {
	by, err := cpu.readByte(step.address1)
	if err != 0 {
		return err
	}
	by |= uint32(step.reg)
	if err = cpu.writeByte(step.address1, by); err != 0 {
		return err
	}
	if by != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical Exclusive Or immediate to memory.
func (cpu *cpu) opXI(step *stepInfo) uint16 {
	by, err := cpu.readByte(step.address1)
	if err != 0 {
		return err
	}
	by ^= uint32(step.reg)
	err = cpu.writeByte(step.address1, by)
	if err != 0 {
		return err
	}
	if by != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Compare immediate with memory.
func (cpu *cpu) opCLI(step *stepInfo) uint16 {
	by, err := cpu.readByte(step.address1)
	if err != 0 {
		return err
	}
	if by != uint32(step.reg) {
		if by < uint32(step.reg) {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	} else {
		cpu.cc = 0
	}
	return 0
}

// Move immediate value to memory.
func (cpu *cpu) opMVI(step *stepInfo) uint16 {
	return cpu.writeByte(step.address1, uint32(step.reg))
}

// Store full word into memory.
func (cpu *cpu) opST(step *stepInfo) uint16 {
	return cpu.writeFull(step.address1, step.src1)
}

// Store half word into memory.
func (cpu *cpu) opSTH(step *stepInfo) uint16 {
	return cpu.writeHalf(step.address1, step.src1)
}

// Store character into memory.
func (cpu *cpu) opSTC(step *stepInfo) uint16 {
	return cpu.writeByte(step.address1, step.src1)
}

// Store character under mask.
func (cpu *cpu) opSTCM(step *stepInfo) uint16 {
	// Fetch register
	reg := cpu.regs[step.R1]
	bits := step.R2
	// If no bits are set quick return
	if bits == 0 {
		return 0
	}

	shift := 24 // Shift count
	// Scan from bit 12 to bit 15
	for i := uint8(0x8); i != 0; i >>= 1 {
		// If the bit is one, read in the byte
		if (bits & i) != 0 {
			by := (reg >> shift) & 0xff
			if err := cpu.writeByte(step.address1, by); err != 0 {
				return err
			}
			step.address1++
		}
		shift -= 8
	}
	return 0
}

// Test and set.
func (cpu *cpu) opTS(step *stepInfo) uint16 {
	// Read original
	orig, err := cpu.readByte(step.address1)
	if err != 0 {
		return err
	}

	// Write back 0xff
	if err = cpu.writeByte(step.address1, 0xff); err != 0 {
		return err
	}

	// Set flag based on original value
	if (orig & 0x80) != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Compare character under mask.
func (cpu *cpu) opTM(step *stepInfo) uint16 {
	by, err := cpu.readByte(step.address1)
	if err != 0 {
		return err
	}
	by &= uint32(step.reg)
	if by != 0 {
		if uint32(step.reg) == by {
			cpu.cc = 3
		} else {
			cpu.cc = 1
		}
	} else {
		cpu.cc = 0
	}
	return 0
}

// Shift right logical.
func (cpu *cpu) opSRL(step *stepInfo) uint16 {
	value := cpu.regs[step.R1]
	shift := step.address1 & 0x3f
	if shift > 31 {
		value = 0
	} else {
		value >>= shift
	}
	cpu.regs[step.R1] = value
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Shift left logical.
func (cpu *cpu) opSLL(step *stepInfo) uint16 {
	value := cpu.regs[step.R1]
	shift := step.address1 & 0x3f
	if shift > 31 {
		value = 0
	} else {
		value <<= shift
	}
	cpu.regs[step.R1] = value
	cpu.perRegMod |= 1 << step.R1
	return 0
}

// Shift Arithmatic right.
func (cpu *cpu) opSRA(step *stepInfo) uint16 {
	value := cpu.regs[step.R1]
	shift := step.address1 & 0x3f
	if shift > 31 {
		if (value & MSIGN) != 0 {
			value = FMASK
		} else {
			value = 0
		}
	} else {
		value = uint32(int32(value) >> shift)
	}
	cpu.regs[step.R1] = value
	cpu.perRegMod |= 1 << step.R1
	cpu.setCC(value)
	return 0
}

// Shift Arithmatic left.
func (cpu *cpu) opSLA(step *stepInfo) uint16 {
	value := cpu.regs[step.R1]
	sign := value & MSIGN
	cpu.cc = 0
	for range step.address1 & 0x3f {
		value <<= 1
		if (value & MSIGN) != sign {
			cpu.cc = 3
		}
	}
	value &= ^MSIGN
	value |= sign
	cpu.regs[step.R1] = value
	cpu.perRegMod |= 1 << step.R1
	if cpu.cc != 3 {
		cpu.setCC(value)
	}
	return 0
}

// Shift Double left logical.
func (cpu *cpu) opSLDL(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return ircSpec
	}
	shift := step.address1 & 0x3f
	value := cpu.loadDouble(step.R1)
	value <<= shift
	cpu.storeDouble(step.R1, value)
	return 0
}

// Shift Double right logical.
func (cpu *cpu) opSRDL(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return ircSpec
	}
	shift := step.address1 & 0x3f
	value := cpu.loadDouble(step.R1)
	value >>= shift
	cpu.storeDouble(step.R1, value)
	return 0
}

// Shift Double left arithmatic.
func (cpu *cpu) opSLDA(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return ircSpec
	}
	cpu.cc = 0
	value := cpu.loadDouble(step.R1)
	sign := value & MSIGNL
	for range step.address1 & 0x3f {
		value <<= 1
		if (value & MSIGNL) != sign {
			cpu.cc = 3
		}
	}
	value &= ^MSIGNL
	value |= sign
	cpu.storeDouble(step.R1, value)
	if cpu.cc != 3 {
		if value != 0 {
			if sign != 0 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		}
	} else {
		if (cpu.progMask & FIXOVER) != 0 {
			return ircFixOver
		}
	}
	return 0
}

// Shift Double Arithmatic right.
func (cpu *cpu) opSRDA(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return ircSpec
	}
	cpu.cc = 0
	value := cpu.loadDouble(step.R1)
	sign := value & MSIGNL
	for range step.address1 & 0x3f {
		value >>= 1
		value |= sign
	}
	cpu.storeDouble(step.R1, value)
	if value != 0 {
		if (value & MSIGNL) != 0 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	return 0
}

// Load multiple registers.
func (cpu *cpu) opLM(step *stepInfo) uint16 {
	reg := step.R2
	regEnd := step.R1
	endAddr := step.address1
	for {
		if regEnd == reg {
			break
		}
		regEnd++
		regEnd &= 0xf
		endAddr += 4
	}
	// If we can access end, first fault will then be on end
	if (step.address1 & PMASK) != (endAddr & PMASK) {
		// Translate address
		physAddr, err := cpu.transAddr(endAddr)
		if err != 0 {
			return err
		}
		if cpu.checkProtect(physAddr, false) {
			return ircProt
		}
	}

	reg = step.R2
	for {
		word, err := cpu.readFull(step.address1)
		if err != 0 {
			return err
		}
		cpu.regs[step.R1] = word
		cpu.perCode |= 1 << step.R1
		if step.R1 == reg {
			return 0
		}
		step.R1++
		step.R1 &= 0xf
		step.address1 += 4
	}
}

// Load multiple registers.
func (cpu *cpu) opSTM(step *stepInfo) uint16 {
	endReg := step.R2

	for {
		if err := cpu.writeFull(step.address1, cpu.regs[step.R1]); err != 0 {
			return err
		}
		if step.R1 == endReg {
			return 0
		}
		step.R1++
		step.R1 &= 0xf
		step.address1 += 4
	}
}

// Handle memory to memory instructions.
func (cpu *cpu) opMem(step *stepInfo) uint16 {
	if err := cpu.testAccess(step.address1, uint32(step.reg), true); err != 0 {
		return err
	}
	if err := cpu.testAccess(step.address2, uint32(step.reg), false); err != 0 {
		return err
	}
	opcode := step.opcode
	if opcode == OpNC || opcode == OpOC || opcode == OpXC {
		cpu.cc = 0
	}

	for {
		var source, dest uint32
		var err uint16
		source, err = cpu.readByte(step.address2)
		if err != 0 {
			return err
		}
		if opcode != OpMVC {
			dest, err = cpu.readByte(step.address1)
			if err != 0 {
				return err
			}
			switch opcode {
			case OpMVZ:
				dest = (dest & 0x0f) | (source & 0xf0)
			case OpMVN:
				dest = (dest & 0xf0) | (source & 0x0f)
			case OpNC:
				dest &= source
				if dest != 0 {
					cpu.cc = 1
				}
			case OpOC:
				dest |= source
				if dest != 0 {
					cpu.cc = 1
				}
			case OpXC:
				dest ^= source
				if dest != 0 {
					cpu.cc = 1
				}
			}
		} else {
			dest = source
		}
		if err = cpu.writeByte(step.address1, dest); err != 0 {
			return err
		}
		step.address1++
		step.address2++
		step.reg--
		if step.reg == 0xff {
			return 0
		}
	}
}

// Compare memory to memory.
func (cpu *cpu) opCLC(step *stepInfo) uint16 {
	if err := cpu.testAccess(step.address1, uint32(step.reg), false); err != 0 {
		return err
	}
	if err := cpu.testAccess(step.address2, uint32(step.reg), false); err != 0 {
		return err
	}
	cpu.cc = 0

	for {
		var source1, source2 uint32
		var err uint16
		source2, err = cpu.readByte(step.address2)
		if err != 0 {
			return err
		}

		source1, err = cpu.readByte(step.address1)
		if err != 0 {
			return err
		}

		if source1 != source2 {
			if source1 > source2 {
				cpu.cc = 2
			} else {
				cpu.cc = 1
			}
			return 0
		}

		step.address1++
		step.address2++
		step.reg--
		if step.reg == 0xff {
			return 0
		}
	}
}

// Translate memory and Translate and Test.
func (cpu *cpu) opTR(step *stepInfo) uint16 {
	err := cpu.testAccess(step.address1, uint32(step.reg), true)
	if err != 0 {
		return err
	}
	err = cpu.testAccess(step.address2, 256, false)
	if err != 0 {
		return err
	}

	if step.opcode == OpTRT {
		cpuState.cc = 0
	}

	for {
		var source, xlatValue uint32
		source, err = cpu.readByte(step.address1)
		if err != 0 {
			return err
		}

		xlatValue, err = cpu.readByte(step.address2 + (source & 0xff))
		if err != 0 {
			return err
		}
		if step.opcode == OpTRT {
			if xlatValue != 0 {
				cpu.regs[1] &= 0xff000000
				cpu.regs[1] |= step.address1 & AMASK
				cpu.regs[2] &= 0xffffff00
				cpu.regs[2] |= xlatValue & 0xff
				cpu.perRegMod |= 6
				if step.reg == 0 {
					cpu.cc = 2
				} else {
					cpu.cc = 1
				}
				return 0
			}
		} else {
			err = cpu.writeByte(step.address1, xlatValue)
			if err != 0 {
				return err
			}
		}
		step.address1++
		step.reg--
		if step.reg == 0xff {
			return 0
		}
	}
}

// Move with offset.
func (cpu *cpu) opMVO(step *stepInfo) uint16 {

	err := cpu.testAccess(step.address1, uint32(step.R2), true)
	if err != 0 {
		return err
	}
	err = cpu.testAccess(step.address2, uint32(step.R1), false)
	if err != 0 {
		return err
	}

	step.address1 += uint32(step.R1)
	step.address2 += uint32(step.R2)

	var source1, result uint32
	source1, err = cpu.readByte(step.address1)
	if err != 0 {
		return err
	}

	result, err = cpu.readByte(step.address2)
	if err != 0 {
		return err
	}
	step.address2--

	source1 = (source1 & 0xf) | ((result << 4) & 0xf0)
	err = cpu.writeByte(step.address1, source1)
	if err != 0 {
		return err
	}
	step.address1--

	for step.R1 != 0 {
		source1 = (result >> 4) & 0xf
		if step.R2 != 0 {
			result, err = cpu.readByte(step.address2)
			if err != 0 {
				return err
			}
			step.address2--
			step.R2--
		} else {
			result = 0
		}
		source1 |= (result << 4) & 0xf0
		err = cpu.writeByte(step.address1, source1)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	return 0
}

// Move character inverse.
func (cpu *cpu) opMVCIN(step *stepInfo) uint16 {
	if err := cpu.testAccess(step.address1, uint32(step.reg), true); err != 0 {
		return err
	}
	if err := cpu.testAccess(step.address2-uint32(step.reg), uint32(step.reg), false); err != 0 {
		return err
	}

	for {
		source, err := cpu.readByte(step.address2)
		if err != 0 {
			return err
		}
		err = cpu.writeByte(step.address1, source)
		if err != 0 {
			return err
		}
		step.address2--
		step.address1++
		step.reg--
		if step.reg == 0xff {
			return 0
		}
	}
}

// Move Character Long.
func (cpu *cpu) opMVCL(step *stepInfo) uint16 {
	var err uint16
	var temp uint32

	// Check register alignment
	if (step.R2&1) != 0 || (step.R1&1) != 0 {
		return ircSpec
	}

	err = 0
	// extract parameters
	addr1 := cpu.regs[step.R1] & AMASK
	len1 := cpu.regs[step.R1|1] & AMASK
	addr2 := cpu.regs[step.R2] & AMASK
	len2 := cpu.regs[step.R2|1] & AMASK
	fill := (cpu.regs[step.R2|1] >> 24) & 0xff

	// Handle overlap
	if len1 > 1 && len2 > 1 {
		if addr2 < addr1 {
			temp = addr2 + len2 - 1
		} else {
			temp = addr2 + len1 - 1
		}
		temp &= AMASK
		if (temp > addr2 && (addr1 > addr2 && addr1 <= temp)) ||
			(temp <= addr2 && (addr1 > addr2 || addr1 <= temp)) {
			cpu.cc = 3
			return 0
		}
	}

	// Set condition codes
	if len1 < len2 {
		cpu.cc = 1
	} else if len1 > len2 {
		cpu.cc = 2
	} else {
		cpu.cc = 0
	}

	// Preform actual move
	for len1 != 0 {
		if len2 == 0 {
			temp = fill
		} else {
			temp, err = cpu.readByte(addr2)
			if err != 0 {
				break
			}
		}

		err = cpu.writeByte(addr1, temp)
		if err != 0 {
			break
		}

		if len2 != 0 {
			addr2++
			addr2 &= AMASK
		}
		addr1++
		addr1 &= AMASK
		len1--
	}
	// Save registers back
	cpu.regs[step.R1] = addr1
	cpu.regs[step.R1|1] &= ^AMASK
	cpu.regs[step.R1|1] |= len1 & AMASK
	cpu.perRegMod |= 3 << step.R1
	cpu.regs[step.R2] = addr2
	cpu.regs[step.R2|1] &= ^AMASK
	cpu.regs[step.R2|1] |= len1 & AMASK
	cpu.perRegMod |= 3 << step.R2
	return err
}

// Compare logical long.
func (cpu *cpu) opCLCL(step *stepInfo) uint16 {
	var err uint16
	var source1, source2 uint32

	// Check register alignment
	if (step.R2&1) != 0 || (step.R1&1) != 0 {
		return ircSpec
	}

	// extract parameters
	addr1 := cpu.regs[step.R1] & AMASK
	len1 := cpu.regs[step.R1|1] & AMASK
	addr2 := cpu.regs[step.R2] & AMASK
	len2 := cpu.regs[step.R2|1] & AMASK
	fill := (cpu.regs[step.R2|1] >> 24) & 0xff
	cpu.cc = 0

	// Preform compare
	for len1 != 0 || len2 != 0 {
		if len1 == 0 {
			source1 = fill
		} else {
			source1, err = cpu.readByte(addr1)
			if err != 0 {
				break
			}
		}

		if len2 == 0 {
			source2 = fill
		} else {
			source2, err = cpu.readByte(addr2)
			if err != 0 {
				break
			}
		}

		// Do compare
		if source1 != source2 {
			if source1 > source2 {
				cpu.cc = 2
			} else {
				cpu.cc = 1
			}
			break
		}

		// Adjust pointers to next item
		if len2 != 0 {
			addr2++
			addr2 &= AMASK
			len2--
		}

		if len1 != 0 {
			addr1++
			addr1 &= AMASK
			len1--
		}
	}
	// Save registers back
	cpu.regs[step.R1] = addr1
	cpu.regs[step.R1|1] &= ^AMASK
	cpu.regs[step.R1|1] |= len1 & AMASK
	cpu.perRegMod |= 3 << step.R1
	cpu.regs[step.R2] = addr2
	cpu.regs[step.R2|1] &= ^AMASK
	cpu.regs[step.R2|1] |= len2 & AMASK
	cpu.perRegMod |= 3 << step.R2
	return err
}

// Pack characters into digits.
func (cpu *cpu) opPACK(step *stepInfo) uint16 {
	var source, result uint32

	err := cpu.testAccess(step.address1, 0, true)
	if err != 0 {
		return err
	}
	err = cpu.testAccess(step.address2, 0, false)
	if err != 0 {
		return err
	}

	step.address1 += uint32(step.R1)
	step.address2 += uint32(step.R2)
	// Flip first location
	source, err = cpu.readByte(step.address2)
	if err != 0 {
		return err
	}
	source = ((source >> 4) & 0xf) | ((source << 4) & 0xf0)
	err = cpu.writeByte(step.address1, source)
	if err != 0 {
		return err
	}

	step.address1--
	step.address2--
	for step.R1 != 0 && step.R2 != 0 {
		source, err = cpu.readByte(step.address2)
		if err != 0 {
			return err
		}
		source &= uint32(0xf)
		step.address2--
		step.R2--
		if step.R2 != 0 {
			result, err = cpu.readByte(step.address2)
			if err != 0 {
				return err
			}
			source |= (result << 4) & 0xf0
			step.address2--
			step.R2--
		}
		err = cpu.writeByte(step.address1, source)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	source = 0
	for step.R1 != 0 {
		err = cpu.writeByte(step.address1, source)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	return 0
}

// Unpack packed BCD to character BCD.
func (cpu *cpu) opUNPK(step *stepInfo) uint16 {
	var source uint32

	err := cpu.testAccess(step.address1, 0, true)
	if err != 0 {
		return err
	}
	err = cpu.testAccess(step.address2, 0, false)
	if err != 0 {
		return err
	}

	// Point to end
	step.address1 += uint32(step.R1)
	step.address2 += uint32(step.R2)

	// Flip first location
	source, err = cpu.readByte(step.address2)
	if err != 0 {
		return err
	}
	source = ((source >> 4) & 0xf) | ((source << 4) & 0xf0)
	err = cpu.writeByte(step.address1, source)
	if err != 0 {
		return err
	}
	step.address1--
	step.address2--
	for step.R1 != 0 && step.R2 != 0 {
		source, err = cpu.readByte(step.address2)
		if err != 0 {
			return err
		}
		step.address2--
		step.R2--
		t2 := (source & 0xf) | 0xf0
		err = cpu.writeByte(step.address1, t2)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
		if step.R1 != 0 {
			t2 = ((source >> 4) & 0xf) | 0xf0
			err = cpu.writeByte(step.address1, t2)
			if err != 0 {
				return err
			}
			step.address1--
			step.R1--
		}
	}
	for step.R1 != 0 {
		err = cpu.writeByte(step.address1, 0xf0)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	return 0
}

// Convert packed decimal to binary.
func (cpu *cpu) opCVB(step *stepInfo) uint16 {
	var err uint16
	var sourceLow, sourceHigh uint32
	var sign uint32
	var result uint64

	// Read in number to convert
	sourceLow, err = cpu.readFull(step.address1)
	if err != 0 {
		return err
	}
	sourceHigh, err = cpu.readFull(step.address1 + 4)
	if err != 0 {
		return err
	}

	// Grab sign.
	sign = sourceHigh & uint32(0xf)

	// If sign invalid
	if sign < 0xa {
		return ircData
	}
	result = 0

	// Convert upper
	for i := 28; i >= 0; i -= 4 {
		d := (sourceLow >> i) & uint32(0xf)
		if d >= 0xa {
			return ircData
		}
		result = (result * 10) + uint64(d)
	}

	// Convert lower
	for i := 28; i > 0; i -= 4 {
		d := (sourceHigh >> i) & uint32(0xf)
		if d >= 0xa {
			return ircData
		}
		result = (result * 10) + uint64(d)
	}

	r := uint16(0)
	// Check if too big
	if (result&OMASKL) != 0 && result != uint64(MSIGN) {
		r = ircFixDiv
	}

	// two's compliment if needed
	if sign == 0xb || sign == 0xd {
		result = ^result + 1
	}

	cpu.regs[step.R1] = uint32(result & LMASKL)
	cpu.perRegMod |= 1 << step.R1
	return r
}

// Convert binary to packed decimal.
func (cpu *cpu) opCVD(step *stepInfo) uint16 {
	value := cpu.regs[step.R1]

	// Save sign
	sign := false
	if (value & MSIGN) != 0 {
		value = ^value + 1
		sign = true
	}

	// Convert to packed decimal
	result := uint64(0)
	for i := 4; value != 0; i += 4 {
		d := value % 10
		value /= 10
		result |= uint64(d) << i
	}

	// Fill in sign
	if sign {
		result |= 0xd
	} else {
		result |= 0xc
	}

	// Store packed value
	value = uint32((result >> 32) & LMASKL)
	err := cpu.writeFull(step.address1, value)
	if err != 0 {
		return err
	}
	value = uint32(result & LMASKL)
	return cpu.writeFull(step.address1+4, value)
}

// Edit string, mark saves address of significant digit.
func (cpu *cpu) opED(step *stepInfo) uint16 {
	var err uint16
	var src1f, src2f uint32 // Full word source
	var src1, src2 uint8    // Working source digit
	var fill, digit uint8   // Fill character and digit
	var cctemp uint8        // Temporary CC value
	var sig bool            // Signifigance indicator
	var need bool           // Need another digit

	src1f, err = cpu.readFull(step.address1 & WMASK)
	if err != 0 {
		return err
	}

	src1 = uint8((src1f >> (8 * (3 - (step.address1 & 0x3)))) & 0xff)
	fill = src1
	digit = src1
	cctemp = 0
	sig = false
	need = true
	cpu.cc = 0

	src2f, err = cpu.readFull(step.address2 & WMASK)
	if err != 0 {
		return err
	}
	src2 = uint8((src2f >> (8 * (3 - (step.address2 & 0x3)))) & 0xff)

	for {
		var temp uint8

		switch digit {
		case 0x21, 0x20: // Significance starter, digit selector

			// If we have not run of source, grab next pair
			if need {
				if (step.address2 & 3) == 0 {
					src2f, err = cpu.readFull(step.address2)
					if err != 0 {
						return err
					}
				}
				src2 = uint8((src2f >> (8 * (3 - (step.address2 & 0x3)))) & 0xff)
				step.address2++
				// Check if valid
				if src2 >= 0xa0 {
					return ircData
				}
			}

			// Split apart
			temp = (src2 >> 4) & 0xf
			need = !need

			// Prepare for next trip
			src2 = (src2 & 0xf) << 4
			if step.opcode == OpEDMK && !sig && temp != 0 {
				cpu.regs[1] &= 0xff000000
				cpu.regs[1] |= step.address1 & AMASK
				cpu.perRegMod |= 2
			}

			// Found non-zero
			if temp != 0 {
				cctemp = 2
			}

			// Select digit of fill
			if temp != 0 || sig {
				digit = 0xf0 | temp
			} else {
				digit = fill
			}

			if src1 == 0x21 || temp != 0 {
				sig = true
			}

			// If sign, update status
			if !need { // Check if found sign
				switch src2 {
				case 0xa0, 0xc0, 0xe0, 0xf0: // Minus
					sig = false
					fallthrough
				case 0xb0, 0xd0:
					need = true
				}
			}
		case 0x22: // Field separator
			sig = false
			digit = fill
			cctemp = 0 // set zero
		default: // Anything else
			if !sig {
				digit = fill
			}
		}

		// Save result
		err = cpu.writeByte(step.address1, uint32(digit))
		if err != 0 {
			return err
		}
		step.address1++
		if step.reg == 0 {
			break
		}
		step.reg--
		if (step.address1 & 3) == 0 {
			src1f, err = cpu.readFull(step.address1)
			if err != 0 {
				return err
			}
		}
		src1 = uint8((src1f >> (8 * (3 - (step.address1 & 0x3)))) & 0xff)
		digit = src1
	}
	cpu.cc = cctemp
	if sig && cpu.cc == 2 {
		cpu.cc = 1
	}

	return 0
}
