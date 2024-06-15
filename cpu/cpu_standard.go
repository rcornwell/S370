package cpu

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

// Handle an unknown instruction
func (cpu *CPU) opUnk(step *stepInfo) uint16 {
	return IRC_OPR
}

// Set program mask
func (cpu *CPU) opSPM(step *stepInfo) uint16 {
	cpu.pmask = uint8(step.src1>>24) & 0xf
	cpu.cc = uint8(step.src1>>28) & 0x3
	return 0
}

// Branch and save
func (cpu *CPU) opBAS(step *stepInfo) uint16 {
	dest := cpu.PC
	if step.opcode != OP_BASR || step.R2 != 0 {
		if cpu.per_en && (cpu.cregs[9]&0x80000000) != 0 {
			cpu.per_code |= 0x8000 /* Set PER branch */
		}
		cpu.PC = step.address1 & AMASK
	}
	cpu.regs[step.R1] = dest
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Branch and link
func (cpu *CPU) opBAL(step *stepInfo) uint16 {
	dest := (uint32(cpu.ilc) << 30) |
		(uint32(cpu.cc) << 28) |
		(uint32(cpu.pmask) << 24) |
		cpu.PC
	if step.opcode != OP_BALR || step.R2 != 0 {
		if cpu.per_en && (cpu.cregs[9]&0x80000000) != 0 {
			cpu.per_code |= 0x8000 /* Set PER branch */
		}
		cpu.PC = step.address1 & AMASK
	}
	cpu.regs[step.R1] = dest
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Branch and count register
func (cpu *CPU) opBCT(step *stepInfo) uint16 {
	dest := step.src1 - 1

	if dest != 0 && (step.opcode != OP_BCTR || step.R2 != 0) {
		if cpu.per_en && (cpu.cregs[9]&0x80000000) != 0 {
			cpu.per_code |= 0x8000 /* Set PER branch */
		}
		cpu.PC = step.address1 & AMASK
	}
	cpu.regs[step.R1] = dest
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Branch conditional
func (cpu *CPU) opBC(step *stepInfo) uint16 {
	if ((0x8>>cpu.cc)&step.R1) != 0 && (step.opcode != OP_BCR || step.R2 != 0) {
		if cpu.per_en && (cpu.cregs[9]&0x80000000) != 0 {
			cpu.per_code |= 0x8000 /* Set PER branch */
		}
		cpu.PC = step.address1 & AMASK
	}
	return 0
}

// Branch on index high
func (cpu *CPU) opBXH(step *stepInfo) uint16 {
	src1 := cpu.regs[step.R2|1]
	dest := cpu.regs[step.R1] + cpu.regs[step.R2]
	cpu.regs[step.R1] = dest
	cpu.per_mod |= 1 << step.R1
	if int32(dest) > int32(src1) {
		if cpu.per_en && (cpu.cregs[9]&0x80000000) != 0 {
			cpu.per_code |= 0x8000 /* Set PER branch */
		}
		cpu.PC = step.address1 & AMASK
	}
	return 0
}

// Branch of index low or equal
func (cpu *CPU) op_BXLE(step *stepInfo) uint16 {
	src1 := cpu.regs[step.R2|1]
	dest := cpu.regs[step.R1] + cpu.regs[step.R2]
	cpu.regs[step.R1] = dest
	cpu.per_mod |= 1 << step.R1
	if int32(dest) <= int32(src1) {
		if cpu.per_en && (cpu.cregs[9]&0x80000000) != 0 {
			cpu.per_code |= 0x8000 /* Set PER branch */
		}
		cpu.PC = step.address1 & AMASK
	}
	return 0
}

// Set the condition code based on value provided
func (cpu *CPU) setCC(value uint32) {
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

// Load register and make positive
func (cpu *CPU) opLPR(step *stepInfo) uint16 {
	if step.src2 == MSIGN {
		cpu.cc = 3
	} else if (step.src2 & MSIGN) != 0 {
		step.src2 = (FMASK ^ step.src2) + 1
		cpu.setCC(step.src2)
	} else {
		cpu.setCC(step.src2)
	}
	cpu.regs[step.R1] = step.src2
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Load complement of register
func (cpu *CPU) opLCR(step *stepInfo) uint16 {
	if step.src2 != MSIGN {
		step.src2 = (FMASK ^ step.src2) + 1
		cpu.setCC(step.src2)
	} else {
		cpu.cc = 3
	}
	cpu.regs[step.R1] = step.src2
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Load and test register
func (cpu *CPU) opLTR(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src2
	cpu.per_mod |= 1 << step.R1
	cpu.setCC(step.src2)
	return 0
}

// Load number and make it negative
func (cpu *CPU) opLNR(step *stepInfo) uint16 {
	if (step.src2 & MSIGN) == 0 {
		step.src2 = (FMASK ^ step.src2) + 1
	}
	cpu.regs[step.R1] = step.src2
	cpu.per_mod |= 1 << step.R1
	cpu.setCC(step.src2)
	return 0
}

// Load value into register, handle RR and RX
// Also handle LH and LA
func (cpu *CPU) opL(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src2
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Load character into register
func (cpu *CPU) opIC(step *stepInfo) uint16 {
	if t, error := cpu.readByte(step.address1); error != 0 {
		return error
	} else {
		cpu.regs[step.R1] = (step.src1 & 0xffffff00) | (t & 0xff)
		cpu.per_mod |= 1 << step.R1
	}
	return 0
}

// Insert character into register under mask
func (cpu *CPU) opICM(step *stepInfo) uint16 {
	// Fetch register
	d := cpu.regs[step.R1]
	bits := step.R2
	cpu.cc = 0
	// If no bits are set, read first byte to check for trap
	if bits == 0 {
		_, err := cpu.readByte(step.address1)
		return err
	}

	// Set flag to check first bit
	high := uint32(0x80)
	m := uint32(0xff000000)
	s := 24 // Shift count

	// Scan from bit 12 to bit 15
	for i := uint8(0x8); i != 0; i >>= 1 {
		// If the bit is one, read in the byte
		if (bits & i) != 0 {
			if t, err := cpu.readByte(step.address1); err != 0 {
				return err
			} else {

				// Put byte in place
				d = (d & ^m) | ((t << s) & m)

				// If byte is not zero, adjust CC
				if t != 0 {
					if (t & high) != 0 {
						cpu.cc = 1
					}
					if cpu.cc == 0 {
						cpu.cc = 2
					}
				}
				high = 0
				step.address1++
			}
		}
		s -= 8
		m >>= 8
	}
	cpu.regs[step.R1] = d
	cpu.per_mod |= 1 << step.reg
	return 0
}

// Add value to register, handle RR, RX and RH
func (cpu *CPU) opAdd(step *stepInfo) uint16 {
	sum := step.src1 + step.src2
	carry := (step.src1 & step.src2) | ((step.src1 ^ step.src2) & ^sum)
	cpu.regs[step.R1] = sum
	cpu.per_mod |= 1 << step.R1
	if (((carry << 1) ^ carry) & MSIGN) != 0 {
		cpu.cc = 3
		if (cpu.pmask & FIXOVER) != 0 {
			return IRC_FIXOVR
		}
		return 0
	}
	cpu.setCC(sum)
	return 0
}

// Subtract value from register, handle RR, RX and RH
func (cpu *CPU) opSub(step *stepInfo) uint16 {
	s2 := step.src2 ^ FMASK
	diff := step.src1 + s2 + 1
	carry := (step.src1 & s2) | ((step.src1 ^ s2) & ^diff)
	cpu.regs[step.R1] = diff
	cpu.per_mod |= 1 << step.R1
	if (((carry << 1) ^ carry) & MSIGN) != 0 {
		cpu.cc = 3
		if (cpu.pmask & FIXOVER) != 0 {
			return IRC_FIXOVR
		}
		return 0
	}
	cpu.setCC(diff)
	return 0
}

// Compare register with value, handle RR, RX and RH
func (cpu *CPU) opCmp(step *stepInfo) uint16 {
	if int32(step.src1) > int32(step.src2) {
		cpu.cc = 2
	} else if step.src1 != step.src2 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical add value to register, handle RR, RX
func (cpu *CPU) opAddL(step *stepInfo) uint16 {
	sum := step.src1 + step.src2
	cpu.regs[step.R1] = sum
	isum := FMASK ^ sum
	carry := (step.src1 & step.src2) | ((step.src1 ^ step.src2) & isum)
	cpu.per_mod |= 1 << step.R1
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

// Subtract logical value to register, handle RR, RX
func (cpu *CPU) opSubL(step *stepInfo) uint16 {
	step.src2 ^= FMASK
	sum := step.src1 + step.src2 + 1
	cpu.regs[step.R1] = sum
	isum := FMASK ^ sum
	carry := (step.src1 & step.src2) | ((step.src1 ^ step.src2) & isum)
	cpu.per_mod |= 1 << step.R1
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

// Logical compare with register
func (cpu *CPU) opCmpL(step *stepInfo) uint16 {
	if step.src1 > step.src2 {
		cpu.cc = 2
	} else if step.src1 != step.src2 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Compare character under mask
func (cpu *CPU) opCLM(step *stepInfo) uint16 {
	// Fetch register
	d := cpu.regs[step.R1]
	bits := step.R2
	cpu.cc = 0
	// If no bits are set, read first byte to check for trap
	if bits == 0 {
		_, err := cpu.readByte(step.address1)
		return err
	}

	s := 24 // Shift count

	// Scan from bit 12 to bit 15
	for i := uint8(0x8); i != 0; i >>= 1 {
		// If the bit is one, read in the byte
		if (bits & i) != 0 {
			if t, err := cpu.readByte(step.address1); err != 0 {
				return err
			} else {
				// Put byte in place
				t2 := (d >> s) & 0xff

				// Compare values
				if t2 != t {
					if t2 < t {
						cpu.cc = 1
					} else {
						cpu.cc = 2
					}
					return 0
				}
				step.address1++
			}
		}
		s -= 8
	}
	return 0
}

// Multiply register by value, handle RR and RX
func (cpu *CPU) opMul(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return IRC_SPEC
	}
	src1 := int64(int32(cpu.regs[step.R1|1]))
	src1 = src1 * int64(int32(step.src2))
	cpu.storeDouble(step.R1, uint64(src1))
	return 0
}

// Multiply half word.
func (cpu *CPU) opMulH(step *stepInfo) uint16 {
	src1 := int64(int32(cpu.regs[step.R1]))
	src1 = src1 * int64(int32(step.src2))
	cpu.regs[step.R1] = uint32(uint64(src1) & LMASKL)
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Divide register by value, handle RR and RX
func (cpu *CPU) opDiv(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return IRC_SPEC
	}
	if step.src2 == 0 {
		return IRC_FIXDIV
	}
	sign := 0
	srcl := cpu.regs[step.R1]
	srch := cpu.regs[step.R1|1]
	if (srcl & MSIGN) != 0 {
		sign = 3
		srch ^= FMASK
		srcl ^= FMASK
		if srch == FMASK {
			srcl++
		}
		srch++
	}
	if (step.src2 & MSIGN) != 0 {
		sign ^= 1
		step.src2 = (step.src2 ^ FMASK) + 1
	}
	var result uint32
	result = 0
	for range 32 {
		srcl <<= 1
		if (srch & MSIGN) != 0 {
			srcl |= 1
		}
		srch <<= 1
		temp := srcl - step.src2
		result <<= 1
		if (temp & MSIGN) == 0 {
			srcl = temp
			result |= 1
		}
	}

	if (result&MSIGN) != 0 && result != MSIGN {
		return IRC_FIXDIV
	}
	if (sign & 1) != 0 {
		result = (result ^ FMASK) + 1
	}
	if (sign & 2) != 0 {
		srcl = (srcl ^ FMASK) + 1
	}
	cpu.regs[step.R1] = srcl
	cpu.regs[step.R1|1] = result
	cpu.per_mod |= 3 << step.R1
	return 0
}

// Logical And register with value, handle RR and RX
func (cpu *CPU) opAnd(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src1 & step.src2
	cpu.per_mod |= 1 << step.R1
	if cpu.regs[step.R1] != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical Or register with value, handle RR and RX
func (cpu *CPU) opOr(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src1 | step.src2
	cpu.per_mod |= 1 << step.R1
	if cpu.regs[step.R1] != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical Exclusive Or register with value, handle RR and RX
func (cpu *CPU) opXor(step *stepInfo) uint16 {
	cpu.regs[step.R1] = step.src1 ^ step.src2
	cpu.per_mod |= 1 << step.R1
	if cpu.regs[step.R1] != 0 {
		cpu.cc = 1
	} else {
		cpu.cc = 0
	}
	return 0
}

// Logical And immeditate to memory
func (cpu *CPU) opNI(step *stepInfo) uint16 {
	if t, err := cpu.readByte(step.address1); err != 0 {
		return err
	} else {
		t &= uint32(step.reg)
		if err = cpu.writeByte(step.address1, t); err != 0 {
			return err
		}
		if t != 0 {
			cpu.cc = 1
		} else {
			cpu.cc = 0
		}
	}
	return 0
}

// Logical Or immediate to memory
func (cpu *CPU) opOI(step *stepInfo) uint16 {
	if t, err := cpu.readByte(step.address1); err != 0 {
		return err
	} else {
		t |= uint32(step.reg)
		if err = cpu.writeByte(step.address1, t); err != 0 {
			return err
		}
		if t != 0 {
			cpu.cc = 1
		} else {
			cpu.cc = 0
		}
	}
	return 0
}

// Logical Exclusive Or immediate to memory
func (cpu *CPU) opXI(step *stepInfo) uint16 {
	if t, err := cpu.readByte(step.address1); err != 0 {
		return err
	} else {
		t ^= uint32(step.reg)
		if err = cpu.writeByte(step.address1, t); err != 0 {
			return err
		}
		if t != 0 {
			cpu.cc = 1
		} else {
			cpu.cc = 0
		}
	}
	return 0
}

// Compare immediate with memory
func (cpu *CPU) opCLI(step *stepInfo) uint16 {
	if t, err := cpu.readByte(step.address1); err != 0 {
		return err
	} else {
		if t != uint32(step.reg) {
			if t < uint32(step.reg) {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			cpu.cc = 0
		}
	}
	return 0
}

// Move immediate value to memory
func (cpu *CPU) opMVI(step *stepInfo) uint16 {
	return cpu.writeByte(step.address1, uint32(step.reg))
}

// Store full word into memory
func (cpu *CPU) opST(step *stepInfo) uint16 {
	return cpu.writeFull(step.address1, step.src1)
}

// Store half word into memory
func (cpu *CPU) opSTH(step *stepInfo) uint16 {
	return cpu.writeHalf(step.address1, step.src1)
}

// Store character into memory
func (cpu *CPU) opSTC(step *stepInfo) uint16 {
	return cpu.writeByte(step.address1, step.src1)
}

// Store character under mask
func (cpu *CPU) opSTCM(step *stepInfo) uint16 {
	// Fetch register
	d := cpu.regs[step.R1]
	bits := step.R2
	// If no bits are set quick return
	if bits == 0 {
		return 0
	}

	s := 24 // Shift count
	// Scan from bit 12 to bit 15
	for i := uint8(0x8); i != 0; i >>= 1 {
		// If the bit is one, read in the byte
		if (bits & i) != 0 {
			t := (d >> s) & 0xff
			if err := cpu.writeByte(step.address1, t); err != 0 {
				return err
			}
			step.address1++
		}
		s -= 8
	}
	return 0
}

// Test and set.
func (cpu *CPU) opTS(step *stepInfo) uint16 {
	if t2, err := cpu.readByte(step.address1); err != 0 {
		return err
	} else {
		if err = cpu.writeByte(step.address1, step.src1&0xff); err != 0 {
			return err
		}
		if (t2 & 0x80) != 0 {
			cpu.cc = 1
		} else {
			cpu.cc = 0
		}
	}
	return 0
}

// Compare character under mask
func (cpu *CPU) opTM(step *stepInfo) uint16 {
	if t, err := cpu.readByte(step.address1); err != 0 {
		return err
	} else {
		t &= uint32(step.reg)
		if t != 0 {
			if uint32(step.reg) == t {
				cpu.cc = 3
			} else {
				cpu.cc = 1
			}
		} else {
			cpu.cc = 0
		}
	}
	return 0
}

// Shift right logical
func (cpu *CPU) opSRL(step *stepInfo) uint16 {
	t := cpu.regs[step.R1]
	s := step.address1 & 0x3f
	if s > 31 {
		t = 0
	} else {
		t = t >> s
	}
	cpu.regs[step.R1] = t
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Shift left logical
func (cpu *CPU) opSLL(step *stepInfo) uint16 {
	t := cpu.regs[step.R1]
	s := step.address1 & 0x3f
	if s > 31 {
		t = 0
	} else {
		t = t << s
	}
	cpu.regs[step.R1] = t
	cpu.per_mod |= 1 << step.R1
	return 0
}

// Shift Arithmatic right
func (cpu *CPU) opSRA(step *stepInfo) uint16 {
	t := cpu.regs[step.R1]
	s := step.address1 & 0x3f
	if s > 31 {
		if (t & MSIGN) != 0 {
			t = FMASK
		} else {
			t = 0
		}
	} else {
		t = uint32(int32(t) >> s)
	}
	cpu.regs[step.R1] = t
	cpu.per_mod |= 1 << step.R1
	cpu.setCC(t)
	return 0
}

// Shift Arithmatic left
func (cpu *CPU) opSLA(step *stepInfo) uint16 {
	t := cpu.regs[step.R1]
	si := t & MSIGN
	cpu.cc = 0
	for s := step.address1 & 0x3f; s > 0; s-- {
		t <<= 1
		if (t & MSIGN) != si {
			cpu.cc = 3
		}
	}
	t &= MSIGN
	t |= si
	cpu.regs[step.R1] = t
	cpu.per_mod |= 1 << step.R1
	if cpu.cc != 3 {
		cpu.setCC(t)
	}
	return 0
}

// Shift Double left logical
func (cpu *CPU) opSLDL(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return IRC_SPEC
	}
	s := step.address1 & 0x3f
	v := cpu.loadDouble(step.R1)
	v <<= s
	cpu.storeDouble(step.R1, v)
	return 0
}

// Shift Double right logical
func (cpu *CPU) opSRDL(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return IRC_SPEC
	}
	s := step.address1 & 0x3f
	v := cpu.loadDouble(step.R1)
	v >>= s
	cpu.storeDouble(step.R1, v)
	return 0
}

// Shift Double left arithmatic
func (cpu *CPU) opSLDA(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return IRC_SPEC
	}
	cpu.cc = 0
	t := cpu.loadDouble(step.R1)
	si := t & MSIGNL
	for s := step.address1 & 0x3f; s > 0; s-- {
		t <<= 1
		if (t & MSIGNL) != si {
			cpu.cc = 3
		}
	}
	t &= MSIGNL
	t |= si
	cpu.storeDouble(step.R1, t)
	if cpu.cc != 3 {
		if t != 0 {
			if (t & MSIGNL) != 0 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		}
	} else {
		if (cpu.pmask & FIXOVER) != 0 {
			return IRC_FIXOVR
		}
	}
	return 0
}

// Shift Double Arithmatic right
func (cpu *CPU) opSRDA(step *stepInfo) uint16 {
	if (step.R1 & 1) != 0 {
		return IRC_SPEC
	}
	cpu.cc = 0
	t := cpu.loadDouble(step.R1)
	si := t & MSIGNL
	for s := step.address1 & 0x3f; s > 0; s-- {
		t >>= 1
		t |= si
	}
	cpu.storeDouble(step.R1, t)
	if t != 0 {
		if (t & MSIGNL) != 0 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	return 0
}

// Load multiple registers
func (cpu *CPU) opLM(step *stepInfo) uint16 {
	r := step.R2
	tr := step.R1
	a2 := step.address1
	for {
		if tr == r {
			break
		}
		tr++
		tr &= 0xf
		a2 += 4
	}
	// If we can access end, first fault will then be on end
	if (step.address1 & PMASK) != (a2 & PMASK) {
		// Translate address
		if pa, err := cpu.transAddr(a2); err != 0 {
			return err
		} else {
			if cpu.checkProtect(pa, false) {
				return IRC_PROT
			}
		}
	}

	r = step.R2
	for {
		if t, err := cpu.readFull(step.address1); err != 0 {
			return err
		} else {
			cpu.regs[step.R1] = t
		}
		cpu.per_code |= 1 << step.R1
		if step.R1 == r {
			return 0
		}
		step.R1++
		step.R1 &= 0xf
		step.address1 += 4
	}
}

// Load multiple registers
func (cpu *CPU) opSTM(step *stepInfo) uint16 {
	r := step.R2

	for {
		if err := cpu.writeFull(step.address1, cpu.regs[step.R1]); err != 0 {
			return err
		}
		if step.R1 == r {
			return 0
		}
		step.R1++
		step.R1 &= 0xf
		step.address1 += 4
	}
}

// Handle memory to memory instructions
func (cpu *CPU) opMem(step *stepInfo) uint16 {
	if err := cpu.testAccess(step.address1, uint32(step.reg), true); err != 0 {
		return err
	}
	if err := cpu.testAccess(step.address2, uint32(step.reg), false); err != 0 {
		return err
	}
	o := step.opcode
	if o == OP_NC || o == OP_OC || o == OP_XC {
		cpu.cc = 0
	}

	for {
		var ts, td uint32
		var err uint16
		ts, err = cpu.readByte(step.address2)
		if err != 0 {
			return err
		}
		if o != OP_MVC {
			td, err = cpu.readByte(step.address1)
			if err != 0 {
				return err
			}
			switch o {
			case OP_MVZ:
				td = (td & 0x0f) | (ts & 0xf0)
			case OP_MVN:
				td = (td & 0xf0) | (ts & 0x0f)
			case OP_NC:
				td &= ts
				if td != 0 {
					cpu.cc = 1
				}
			case OP_OC:
				td |= ts
				if td != 0 {
					cpu.cc = 1
				}
			case OP_XC:
				td ^= ts
				if td != 0 {
					cpu.cc = 1
				}
			}
		} else {
			td = ts
		}
		if err = cpu.writeByte(step.address1, td); err != 0 {
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

// Compare memory to memory
func (cpu *CPU) opCLC(step *stepInfo) uint16 {
	if err := cpu.testAccess(step.address1, uint32(step.reg), false); err != 0 {
		return err
	}
	if err := cpu.testAccess(step.address2, uint32(step.reg), false); err != 0 {
		return err
	}
	cpu.cc = 0

	for {
		var t1, t2 uint32
		var err uint16
		t2, err = cpu.readByte(step.address2)
		if err != 0 {
			return err
		}

		t1, err = cpu.readByte(step.address1)
		if err != 0 {
			return err
		}

		if t1 != t2 {
			if t1 > t2 {
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

// Translate memory and Translate and Test
func (cpu *CPU) opTR(step *stepInfo) uint16 {
	if err := cpu.testAccess(step.address1, uint32(step.reg), true); err != 0 {
		return err
	}
	if err := cpu.testAccess(step.address2, 256, false); err != 0 {
		return err
	}

	for {
		var t1, t2 uint32
		var err uint16
		t1, err = cpu.readByte(step.address1)
		if err != 0 {
			return err
		}

		t2, err = cpu.readByte(step.address2 + (t1 & 0xff))
		if err != 0 {
			return err
		}
		if step.opcode == OP_TRT {
			if t2 != 0 {
				cpu.regs[1] &= 0xff000000
				cpu.regs[1] |= step.address1 & AMASK
				cpu.regs[2] &= 0xffffff00
				cpu.regs[2] |= t2 & 0xff
				cpu.per_mod |= 6
				if step.reg == 0 {
					cpu.cc = 2
				} else {
					cpu.cc = 1
				}
				return 0
			}
		} else {
			err = cpu.writeByte(step.address1, t1)
			if err != 0 {
				return err
			}
			step.address2++
		}
		step.address1++
		step.reg--
		if step.reg == 0xff {
			return 0
		}
	}
}

// Move with offset
func (cpu *CPU) opMVO(step *stepInfo) uint16 {
	var err uint16
	var t1, t2 uint32

	err = cpu.testAccess(step.address1, uint32(step.R2), true)
	if err != 0 {
		return err
	}
	err = cpu.testAccess(step.address2, uint32(step.R1), false)
	if err != 0 {
		return err
	}

	step.address1 += uint32(step.R1)
	step.address2 += uint32(step.R2)

	t1, err = cpu.readByte(step.address1)
	if err != 0 {
		return err
	}

	t2, err = cpu.readByte(step.address2)
	if err != 0 {
		return err
	}
	step.address2--

	t1 = (t1 & 0xf) | ((t2 << 4) & 0xf0)
	err = cpu.writeByte(step.address1, t1)
	if err != 0 {
		return err
	}
	step.address1--

	for step.R1 != 0 {
		t1 = (t2 >> 4) & 0xf
		if step.R2 != 0 {
			t2, err = cpu.readByte(step.address2)
			if err != 0 {
				return err
			}
			step.address2--
			step.R2--
		} else {
			t2 = 0
		}
		t1 |= (t2 << 4) & 0xf0
		err = cpu.writeByte(step.address1, t1)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	return 0
}

// Move character inverse
func (cpu *CPU) opMVCIN(step *stepInfo) uint16 {
	if err := cpu.testAccess(step.address1, uint32(step.reg), true); err != 0 {
		return err
	}
	if err := cpu.testAccess(step.address2-uint32(step.reg), uint32(step.reg), false); err != 0 {
		return err
	}

	for {
		if t, err := cpu.readByte(step.address1); err != 0 {
			return err
		} else {
			err = cpu.writeByte(step.address2, t)
			if err != 0 {
				return err
			}
		}
		step.address2--
		step.address1++
		step.reg--
		if step.reg == 0xff {
			return 0
		}
	}
}

// Move Character Long
func (cpu *CPU) opMVCL(step *stepInfo) uint16 {
	var err uint16 = 0
	var d uint32

	// Check register alignment
	if (step.R2&1) != 0 || (step.R1&1) != 0 {
		return IRC_SPEC
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
			d = addr2 + len2 - 1
		} else {
			d = addr2 + len1 - 1
		}
		d &= AMASK
		if (d > addr2 && (addr1 > addr2 && addr1 <= d)) ||
			(d <= addr2 && (addr1 > addr2 || addr1 <= d)) {
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
			d = fill
		} else {
			d, err = cpu.readByte(addr2)
			if err != 0 {
				break
			}
		}

		err = cpu.writeByte(addr1, d)
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
	cpu.per_mod |= 3 << step.R1
	cpu.regs[step.R2] = addr2
	cpu.regs[step.R2|1] &= ^AMASK
	cpu.regs[step.R2|1] |= len1 & AMASK
	cpu.per_mod |= 3 << step.R2
	return err
}

// Compare logical long
func (cpu *CPU) opCLCL(step *stepInfo) uint16 {
	var err uint16 = 0
	var d1, d2 uint32

	// Check register alignment
	if (step.R2&1) != 0 || (step.R1&1) != 0 {
		return IRC_SPEC
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
			d1 = fill
		} else {
			d1, err = cpu.readByte(addr1)
			if err != 0 {
				break
			}
		}

		if len2 == 0 {
			d2 = fill
		} else {
			d2, err = cpu.readByte(addr2)
			if err != 0 {
				break
			}
		}

		// Do compare
		if d1 != d2 {
			if d1 > d2 {
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
	cpu.per_mod |= 3 << step.R1
	cpu.regs[step.R2] = addr2
	cpu.regs[step.R2|1] &= ^AMASK
	cpu.regs[step.R2|1] |= len2 & AMASK
	cpu.per_mod |= 3 << step.R2
	return err
}

// Pack characters into digits.
func (cpu *CPU) opPACK(step *stepInfo) uint16 {
	var err uint16
	var t, t2 uint32

	err = cpu.testAccess(step.address1, 0, true)
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
	t, err = cpu.readByte(step.address2)
	if err != 0 {
		return err
	}
	t = ((t >> 4) & 0xf) | ((t << 4) & 0xf0)
	err = cpu.writeByte(step.address1, t)
	if err != 0 {
		return err
	}

	step.address1--
	step.address2--
	for step.R1 != 0 && step.R2 != 0 {
		t, err = cpu.readByte(step.address2)
		if err != 0 {
			return err
		}
		t &= uint32(0xf)
		step.address2--
		step.R2--
		if step.R1 != 0 {
			t2, err = cpu.readByte(step.address2)
			if err != 0 {
				return err
			}
			t |= (t2 << 4) & 0xf0
			step.address2--
			step.R1--
		}
		err = cpu.writeByte(step.address1, t)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	t = 0
	for step.R1 != 0 {
		err = cpu.writeByte(step.address1, t)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	return 0
}

// Unpack packed BCD to character BCD
func (cpu *CPU) opUNPK(step *stepInfo) uint16 {
	var err uint16
	var t, t2 uint32

	err = cpu.testAccess(step.address1, 0, true)
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
	t, err = cpu.readByte(step.address2)
	if err != 0 {
		return err
	}
	t = ((t >> 4) & 0xf) | ((t << 4) & 0xf0)
	err = cpu.writeByte(step.address1, t)
	if err != 0 {
		return err
	}
	step.address1--
	step.address2--
	for step.R1 != 0 && step.R2 != 0 {
		t, err = cpu.readByte(step.address2)
		if err != 0 {
			return err
		}
		step.address2--
		step.R2--
		t2 = (t & 0xf) | 0xf0
		err = cpu.writeByte(step.address1, t2)
		if err != 0 {
			return err
		}
		if step.R1 != 0 {
			t2 = ((t >> 4) & 0xf) | 0xf0
			err = cpu.writeByte(step.address1, t2)
			if err != 0 {
				return err
			}
			step.address2--
			step.R1--
		}
		err = cpu.writeByte(step.address1, t)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	t = 0xf0
	for step.R1 != 0 {
		err = cpu.writeByte(step.address1, t)
		if err != 0 {
			return err
		}
		step.address1--
		step.R1--
	}
	return 0
}

// Convert packed decimal to binary
func (cpu *CPU) opCVB(step *stepInfo) uint16 {
	var err uint16
	var t1, t2 uint32
	var s uint32
	var v uint64

	t1, err = cpu.readFull(step.address1)
	if err != 0 {
		return err
	}
	t2, err = cpu.readFull(step.address1 + 4)
	if err != 0 {
		return err
	}
	s = t2 & uint32(0xf)
	if s < 0xa {
		return IRC_DATA
	}
	v = 0

	// Convert upper
	for i := 28; i >= 0; i -= 4 {
		d := (t1 >> i) & uint32(0xf)
		if d >= 0xa {
			return IRC_DATA
		}
		v = (v * 10) + uint64(d)
	}

	// Convert lower
	for i := 28; i >= 0; i -= 4 {
		d := (t2 >> i) & uint32(0xf)
		if d >= 0xa {
			return IRC_DATA
		}
		v = (v * 10) + uint64(d)
	}

	// Check if too big
	if (v&OMASKL) != 0 && v != uint64(MSIGN) {
		return IRC_FIXDIV
	}

	// two's compliment if needed
	if s == 0xb || s == 0xd {
		v = ^v + 1
	}

	cpu.regs[step.R1] = uint32(v & LMASKL)
	return 0
}

// Convert binary to packed decimal
func (cpu *CPU) opCVD(step *stepInfo) uint16 {
	var err uint16
	var v uint32
	var t uint64
	var s bool

	v = cpu.regs[step.R1]

	// Save sign
	if (v & MSIGN) != 0 {
		v = ^v + 1
		s = true
	} else {
		s = false
	}

	// Convert to packed decimal
	t = 0
	for i := 4; v != 0; i += 4 {
		d := v % 10
		v /= 10
		t |= uint64(d) << i
	}

	// Fill in sign
	if s {
		t |= 0xd
	} else {
		t |= 0xc
	}

	v = uint32((t >> 32) & LMASKL)
	err = cpu.writeFull(step.address1, v)
	if err != 0 {
		return err
	}
	v = uint32(t & LMASKL)
	return cpu.writeFull(step.address1, v)
}

// Edit string, mark saves address of significant digit
func (cpu *CPU) opED(step *stepInfo) uint16 {
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
		var t uint8

		switch digit {
		case 0x21, 0x20: // Significance starter, digit selector

			// If we have not run of of source, grab next pair
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
					return IRC_DATA
				}
			}

			// Split apart
			t = (src2 >> 4) & 0xf
			need = !need

			// Prepare for next trip
			src2 = (src2 & 0xf) << 4
			if step.opcode == OP_EDMK && !sig && t != 0 {
				cpu.regs[1] &= 0xff000000
				cpu.regs[1] |= step.address1 & AMASK
				cpu.per_mod |= 2
			}

			// Found non-zero
			if t != 0 {
				cctemp = 2
			}

			// Select digit of fill
			if t != 0 || sig {
				digit = 0xf0 | t
			} else {
				digit = fill
			}

			if src1 == 0x21 || t != 0 {
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
