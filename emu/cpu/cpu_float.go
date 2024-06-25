/* IBM 370 Floating point instructions

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

// Floating point half register.
func (cpu *cpu) opFPHalf(step *stepInfo) uint16 {
	var e1 int
	var sign bool

	// Split number apart
	e1 = int((step.fsrc2 & EMASKL) >> 56)
	if (step.opcode & 0x10) != 0 {
		step.fsrc2 &= HMASKL
	}
	sign = (step.fsrc2 & MSIGNL) != 0
	// Create guard digit
	step.fsrc2 = (step.fsrc2 & MMASKL) << 4
	// Divide by 2
	step.fsrc2 >>= 1
	// If not zero normalize result
	if step.fsrc2 != 0 {
		for (step.fsrc2 & SNMASKL) == 0 {
			step.fsrc2 <<= 4
			e1--
		}
		// Check if underflow
		if e1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				return ircExpUnder
			}
			sign = false
			step.fsrc2 = 0
			e1 = 0
		}

		// Remove guard digit
		step.fsrc2 >>= 4
	}

	// Check for zero
	if step.fsrc2 == 0 {
		sign = false
		e1 = 0
	}

	// Restore result
	step.fsrc2 |= (uint64(e1) << 56) & EMASKL
	if sign {
		step.fsrc2 |= MSIGNL
	}
	return cpu.opFPLoad(step)
}

// Floating load register.
func (cpu *cpu) opFPLoad(step *stepInfo) uint16 {
	if (step.opcode & 0x10) == 0 {
		cpu.fpregs[step.R1] = step.fsrc2
	} else {
		cpu.fpregs[step.R1] = (step.fsrc2 & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return 0
}

// Floating point load register with sign change.
func (cpu *cpu) opLcs(step *stepInfo) uint16 {
	if (step.opcode & 0x2) == 0 { // LP, LN
		step.fsrc2 &= ^MSIGNL
	}
	if (step.opcode & 0x1) != 0 { // LN, LC
		step.fsrc2 ^= MSIGNL
	}
	cpu.cc = 0
	fsrc1 := step.fsrc2 & ^MSIGNL
	if (step.opcode & 0x10) == 0 {
		cpu.fpregs[step.R1] = step.fsrc2
	} else {
		cpu.fpregs[step.R1] = (step.fsrc2 & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	if fsrc1 != 0 {
		if (step.fsrc2 & MSIGNL) != 0 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	return 0
}

// Floating point store register double.
func (cpu *cpu) opSTD(step *stepInfo) uint16 {
	t := uint32(step.fsrc1 & LMASKL)
	if err := cpu.writeFull(step.address1+4, t); err != 0 {
		return err
	}
	t = uint32((step.fsrc1 >> 32) & LMASKL)
	return cpu.writeFull(step.address1, t)
}

// Floating point store register short.
func (cpu *cpu) opSTE(step *stepInfo) uint16 {
	t := uint32((step.fsrc1 >> 32) & LMASKL)
	return cpu.writeFull(step.address1, t)
}

// Floating point compare short.
func (cpu *cpu) opCE(step *stepInfo) uint16 {
	// Extract number and adjust
	e1 := int((step.fsrc1 & EMASKL) >> 56)
	e2 := int((step.fsrc2 & EMASKL) >> 56)
	s1 := (step.fsrc1 & MSIGNL) != 0
	s2 := (step.fsrc2 & MSIGNL) != 0

	// Make 32 bit and create guard digit
	v1 := uint32(step.fsrc1>>28) & XMASK
	v2 := uint32(step.fsrc2>>28) & XMASK

	t := e1 - e2
	if t > 0 {
		if t > 8 {
			v2 = 0
		} else {
			// Shift v2 right if v1 larger expo - expo
			v2 >>= 4 * t
		}
	} else if t < 0 {
		if t < -8 {
			v1 = 0
		} else {
			// Shift v1 right if v2 larger expo - expo
			v1 >>= 4 * -t
		}
	}

	// Exponents should be equal now.

	// Subtract results
	var d uint32

	if s1 == s2 {
		// Same signs do subtract
		v2 ^= XMASK
		d = v1 + v2 + 1
		if (d & CMASK) != 0 {
			d &= XMASK
		} else {
			s1 = !s1
			d ^= XMASK
			d++
		}
	} else {
		d = v1 + v2
	}

	// If v1 not normal shift left + expo
	if (d & CMASK) != 0 {
		d >>= 4
	}

	// Set condition code
	cpu.cc = 0
	if d != 0 {
		if s1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	return 0
}

// Floating point short add and subtract.
func (cpu *cpu) opFPAdd(step *stepInfo) uint16 {
	// SER 3B
	// SUR 3F
	// SE  7B
	// SU  7F
	// AER 3A
	// AUR 3E
	// AE  7A
	// AU  7E

	// If subrtact change sign
	if (step.opcode & 1) != 0 {
		step.fsrc2 ^= MSIGNL
	}

	// Extract number and adjust
	e1 := int((step.fsrc1 & EMASKL) >> 56)
	e2 := int((step.fsrc2 & EMASKL) >> 56)
	s1 := (step.fsrc1 & MSIGNL) != 0
	s2 := (step.fsrc2 & MSIGNL) != 0

	// Make 32 bit and create guard digit
	v1 := uint32(step.fsrc1>>28) & XMASK
	v2 := uint32(step.fsrc2>>28) & XMASK

	t := e1 - e2
	if t > 0 {
		if t > 8 {
			v2 = 0
		} else {
			// Shift v2 right if v1 larger expo - expo
			v2 >>= 4 * t
		}
	} else if t < 0 {
		if t < -8 {
			v1 = 0
		} else {
			// Shift v1 right if v2 larger expo - expo
			v1 >>= 4 * -t
		}
		e1 = e2
	}

	var r uint32

	// Exponents should be equal now.
	// Add results
	if s1 != s2 {
		// Different signs do subtract
		v2 ^= XMASK
		r = v1 + v2 + 1
		if (r & CMASK) != 0 {
			r &= XMASK
		} else {
			s1 = !s1
			r ^= XMASK
			r++
		}
	} else {
		r = v1 + v2
	}

	// If v1 not normal shift left + expo
	if (r & CMASK) != 0 {
		r >>= 4
		e1++
		if e1 >= 128 {
			return ircExpOver
		}
	}

	// Set condition codes
	cpu.cc = 0
	// If not unnormalize opcode
	if (step.opcode & 0x0e) != 0x0e {
		if r != 0 {
			if s1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.progMask & SIGMASK) == 0 {
				e1 = 0
			}
			s1 = false
		}
	} else {
		if (r & 0xffffff0) != 0 {
			if s1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.progMask & SIGMASK) == 0 {
				e1 = 0
			}
			r = 0
			s1 = false
		}
	}

	var err uint16
	// Check signifigance exceptions
	if cpu.cc == 0 && (cpu.progMask&SIGMASK) != 0 {
		err = ircSignif
	} else
	// Check if we are normalized addition
	if (step.opcode & 0x0e) != 0x0e {
		if cpu.cc != 0 { // Only if non-zero result
			for (r & SNMASK) == 0 {
				r <<= 4
				e1 = 0
			}
			// Check if underflow
			if e1 < 0 {
				if (cpu.progMask & EXPUNDER) != 0 {
					return ircExpUnder
				}
				r = 0
				s1 = false
				e1 = 0
			}
		}

		// Remove guard digit
		r >>= 4
	}
	r |= uint32(e1<<24) & EMASK
	if cpu.cc != 0 && s1 {
		r |= MSIGN
	}
	// Store result
	cpu.fpregs[step.R1] = (uint64(r) << 32) | (cpu.fpregs[step.R1] & LMASKL)
	return err
}

// Double floating compare.
func (cpu *cpu) opCD(step *stepInfo) uint16 {
	// OP_CD	0x69
	// OP_CDR	0x29

	// Extract number and adjust
	e1 := int((step.fsrc1 & EMASKL) >> 56)
	e2 := int((step.fsrc2 & EMASKL) >> 56)
	s1 := (step.fsrc1 & MSIGNL) != 0
	s2 := (step.fsrc2 & MSIGNL) != 0

	// Make 32 bit and create guard digit
	v1 := step.fsrc1 & MMASKL
	v2 := step.fsrc2 & MMASKL

	t := e1 - e2
	if t > 0 {
		if t > 17 {
			v2 = 0
		} else {
			// Shift v2 right if v1 larger expo - expo
			v2 >>= 4 * t
		}
	} else if t < 0 {
		if t < -17 {
			v1 = 0
		} else {
			// Shift v1 right if v2 larger expo - expo
			v1 >>= 4 * -t
		}
	}

	// Exponents should be equal now.

	// Subtract results
	var d uint64
	if s1 != s2 {
		// Same signs do subtract
		v2 ^= XMASKL
		d = v1 + v2 + 1
		if (d & CMASKL) != 0 {
			d &= XMASKL
		} else {
			s1 = !s1
			d ^= XMASKL
			d++
		}
	} else {
		d = v1 + v2
	}

	// If v1 not normal shift left + expo
	if (d & CMASKL) != 0 {
		d >>= 4
	}

	// Set condition code
	cpu.cc = 0
	if d != 0 {
		if s1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	return 0
}

// Floating point double add and subtract.
func (cpu *cpu) opFPAddD(step *stepInfo) uint16 {
	// SDR 3B
	// SWR 3F
	// SD  6B
	// SW  6F
	// ADR 3A
	// AWR 3E
	// AD  6A
	// AW  6E

	// If subrtact change sign
	if (step.opcode & 1) != 0 {
		step.fsrc2 ^= MSIGNL
	}

	// Extract number and adjust
	e1 := int((step.fsrc1 & EMASKL) >> 56)
	e2 := int((step.fsrc2 & EMASKL) >> 56)
	s1 := (step.fsrc1 & MSIGNL) != 0
	s2 := (step.fsrc2 & MSIGNL) != 0

	// Make 32 bit and create guard digit
	v1 := step.fsrc1 & MMASKL
	v2 := step.fsrc2 & MMASKL

	t := e1 - e2
	if t > 0 {
		if t > 17 {
			v2 = 0
		} else {
			// Shift v2 right if v1 larger expo - expo
			v2 >>= 4 * t
		}
	} else if t < 0 {
		if t < -17 {
			v1 = 0
		} else {
			// Shift v1 right if v2 larger expo - expo
			v1 >>= 4 * -t
		}
		e1 = e2
	}

	// Exponents should be equal now.
	var r uint64 // Add results

	if s1 != s2 {
		// Different signs do subtract
		v2 ^= XMASKL
		r = v1 + v2 + 1
		if (r & CMASKL) != 0 {
			r &= XMASKL
		} else {
			s1 = !s1
			r ^= XMASKL
			r++
		}
	} else {
		r = v1 + v2
	}

	// If v1 not normal shift left + expo
	if (r & CMASKL) != 0 {
		r >>= 4
		e1++
		if e1 >= 128 {
			return ircExpOver
		}
	}

	// Set condition codes
	cpu.cc = 0
	// If not unnormalize opcode
	if (step.opcode & 0x0e) != 0x0e {
		if r != 0 {
			if s1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.progMask & SIGMASK) == 0 {
				e1 = 0
			}
			s1 = false
		}
	} else {
		if (r & 0xffffff0) != 0 {
			if s1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.progMask & SIGMASK) == 0 {
				e1 = 0
			}
			r = 0
			s1 = false
		}
	}

	var err uint16
	// Check signifigance exceptions
	if cpu.cc == 0 && (cpu.progMask&SIGMASK) != 0 {
		err = ircSignif
	} else
	// Check if we are normalized addition
	if (step.opcode & 0x0e) != 0x0e {
		if cpu.cc != 0 { // Only if non-zero result
			for (r & UMASKL) == 0 {
				r <<= 4
				e1 = 0
			}
			// Check if underflow
			if e1 < 0 {
				if (cpu.progMask & EXPUNDER) != 0 {
					return ircExpUnder
				}
				r = 0
				s1 = false
				e1 = 0
			}
		}

		// Remove guard digit
		r >>= 4
	}
	r |= uint64(e1<<56) & EMASKL
	if cpu.cc != 0 && s1 {
		r |= MSIGNL
	}
	// Store result
	cpu.fpregs[step.R1] = r
	return err
}

// Floating point multiply.
func (cpu *cpu) opFPMul(step *stepInfo) uint16 {
	// MDR	2c
	// MER  3c
	// ME   7c
	// MD   6c

	// Extract number and adjust
	e1 := int((step.fsrc1 & EMASKL) >> 56)
	e2 := int((step.fsrc2 & EMASKL) >> 56)
	s1 := (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	// Make 32 bit and create guard digit
	v1 := step.fsrc1 & MMASKL
	v2 := step.fsrc2 & MMASKL

	// Pre-nomalize v1 and v2 */
	if v1 != 0 {
		for (v1 & NMASKL) == 0 {
			v1 <<= 4
			e1--
		}
	}
	if v2 != 0 {
		for (v2 & NMASKL) == 0 {
			v2 <<= 4
			e2--
		}
	}
	// Compute exponent
	e1 = e1 + e2 - 65

	// Add in guard digits
	v1 <<= 4
	v2 <<= 4
	var r uint64 // Result

	// Do actual multiply
	for range 60 {
		// Add if we need too
		if (v1 & 1) != 0 {
			r += v2
		}
		// Shift right by one
		v1 >>= 1
		r >>= 1
	}

	// If overflow, shift right 4 bits
	if (r & EMASKL) != 0 {
		r >>= 4
		e1++
	}

	var err uint16
	// Check for overflow
	if e1 >= 128 {
		err = ircExpOver
	}

	// Align the results
	if r != 0 {
		for (r & NMASKL) == 0 {
			r <<= 4
			e1--
		} // Make 32 bit and create guard digit
		// Check if underflow
		if e1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				r = 0
				s1 = false
				e1 = 0
			}
		} else {
			e1 = 0
			s1 = false
		}
	}

	// Store result'
	r |= (uint64(e1) << 56) & EMASKL
	if s1 {
		r |= MSIGNL
	}
	if (step.opcode&0x10) == 0 || (step.opcode&0xf) == 0xc {
		cpu.fpregs[step.R1] = r
	} else {
		cpu.fpregs[step.R1] = (r & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return err
}

// Floating point divide.
func (cpu *cpu) opFPDiv(step *stepInfo) uint16 {
	// MDR	2c
	// MER  3c
	// ME   7c
	// MD   6c

	// Extract number and adjust
	e1 := int((step.fsrc1 & EMASKL) >> 56)
	e2 := int((step.fsrc2 & EMASKL) >> 56)
	s1 := (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	v1 := step.fsrc1 & MMASKL
	v2 := step.fsrc2 & MMASKL

	if v2 == 0 {
		return ircFPDiv
	}

	// Pre-nomalize v1 and v2 */
	if v1 != 0 {
		for (v1 & NMASKL) == 0 {
			v1 <<= 4
			e1--
		}
	}
	if v2 != 0 {
		for (v2 & NMASKL) == 0 {
			v2 <<= 4
			e2--
		}
	}
	// Compute exponent
	e1 = e1 - e2 + 64

	// Shift numbers up 4 bits so as not to lose precision below
	v1 <<= 4
	v2 <<= 4

	// Check if we need to adjust divsor if it larger then dividend
	if v1 > v2 {
		v1 >>= 4
		e1++
	}

	// Change sign of v2 so we can add
	v2 ^= XMASKL
	v2++
	var r uint64 // Result

	// Do divide
	for i := 57; i > 0; i-- {
		// Shift left by one
		v1 <<= 1
		// Subtract remainder from dividend
		t := v1 + v2
		// Shift quotent left one bit
		r <<= 1
		// If remainder larger then divsor replace
		if (t & CMASKL) != 0 {
			v1 = t
			r |= 1
		}
		v1 &= XMASKL
	}

	if r == 0x01ffffffffffffff {
		r++
	}
	r >>= 1

	// If overflow, shift right 4 bits
	if (r & EMASKL) != 0 {
		r >>= 4
		e1++
	}

	var err uint16
	// Check for overflow
	if e1 >= 128 {
		err = ircExpOver
	}

	// Align the results
	if r != 0 {
		for (r & NMASKL) == 0 {
			r <<= 4
			e1--
		} // Make 32 bit and create guard digit
		// Check if underflow
		if e1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				r = 0
				s1 = false
				e1 = 0
			}
		} else {
			e1 = 0
			s1 = false
		}
	}

	// Store results
	r |= (uint64(e1) << 56) & EMASKL
	if s1 {
		r |= MSIGNL
	}
	if (step.opcode&0x10) == 0 || (step.opcode&0xf) == 0xc {
		cpu.fpregs[step.R1] = r
	} else {
		cpu.fpregs[step.R1] = (r & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return err
}

// Extended precision load round.
func (cpu *cpu) opLRER(step *stepInfo) uint16 {
	var err uint16
	v := step.fsrc2

	// Check if round bit is one.
	if (v & RMASKL) != 0 {
		// Extract number and adjust
		e := int((v & EMASKL) >> 56)
		s := (v & MSIGNL) != 0
		v = (v & MMASKL) + RMASKL

		// Normalize if needed
		if (v & SNMASKL) != 0 {
			v >>= 4
			e++
			if e > 128 {
				err = ircExpOver
			}
		}
		// Store results
		v |= (uint64(e) << 56) & EMASKL
		if s {
			v |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = (v & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	return err
}

func (cpu *cpu) opLRDR(step *stepInfo) uint16 {
	if (step.R2 & 0xb) != 0 {
		return ircSpec
	}
	var err uint16
	v := cpu.fpregs[step.R2]
	if (cpu.fpregs[step.R2|2] & 0x0080000000000000) != 0 {

		// Extract numbers and adjust
		e := int((v & EMASKL) >> 56)
		s := (v & MSIGNL) != 0
		v = (v & MMASKL) + 1
		if (v & SNMASKL) != 0 {
			v >>= 4
			e++
			if e > 128 {
				err = ircExpOver
			}
		}
		// Store results
		v |= (uint64(e) << 56) & EMASKL
		if s {
			v |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = v
	return err
}

// Handle extended floating point add.
func (cpu *cpu) opAXR(step *stepInfo) uint16 {
	if (step.R1&0xb) != 0 || (step.R2&0xb) != 0 {
		return ircSpec
	}
	var err uint16
	v1l := cpu.fpregs[step.R1]
	v1h := cpu.fpregs[step.R1|2] & MMASKL
	v2l := cpu.fpregs[step.R2]
	v2h := cpu.fpregs[step.R2|2] & MMASKL

	// Extract numbers
	e1 := int((v1l & EMASKL) >> 56)
	s1 := (v1l & MSIGNL) != 0
	v1l = (v1l & MMASKL) + 1
	e2 := int((v2l & EMASKL) >> 56)
	s2 := (v2l & MSIGNL) != 0
	v2l = (v2l & MMASKL) + 1
	if (step.opcode & 1) != 0 {
		s2 = !s2
	}

	// Create Guard digits.
	v1l <<= 4
	v2l <<= 4

	// Align values
	diff := e1 - e2
	if diff > 0 {
		if diff > 15 {
			v2l = 0
			v2h = 0
		} else {
			for range diff {
				v2l >>= 4
				v2l |= (v2h & 0xf) << 60
				v2h >>= 4
			}
		}
	} else if diff < 0 {
		if diff < -15 {
			v1l = 0
			v1h = 0
		} else {
			for range -diff {
				v1l >>= 4
				v1l |= (v1h & 0xf) << 60
				v1h >>= 4
			}
		}
		e1 = e2
	}
	// Exponents should be equal now.

	// Add results
	if s1 != s2 {
		// Different signs do subtract
		v2h ^= XMASKL
		v2l ^= XMASKL
		if v2l == XMASKL {
			v2h++
		}
		v2l++
		// Do actual add
		v1l += v2l
		v1h += v2h
		// Check if overflow lower value
		if (v1l & CMASKL) != 0 {
			v1l &= XMASKL
			v1h++
		}
		// Check if carry out, if not change sign of result
		if (v1h & CMASKL) != 0 {
			v1h &= XMASKL
		} else {
			s1 = !s1
			v1l ^= XMASKL
			v1h ^= XMASKL
			if v1l == XMASKL {
				v1h++
			}
			v1l++
		}
	} else {
		// Do add
		v1l = v1l + v2l
		v1h = v1h + v2h
		// If lower overflowed, increment upper
		if (v1l & CMASKL) != 0 {
			v1l &= XMASKL
			v1h++
		}
	}
	v1l += v2l
	v1h += v2h
	if (v1l & CMASKL) != 0 {
		v1l &= XMASKL
		v1h++
	}

	// If overflow shift right 4 bits
	if (v1h & NMASKL) != 0 {
		v1l >>= 4
		v1l |= (v1h & 0xf) << 60
		v1h >>= 4
		e1++
		if e1 >= 128 {
			err = ircExpOver
		}
	}

	// Set condition codes
	cpu.cc = 0
	if (v1l | v1h) != 0 {
		if s1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	} else {
		s1 = false
		e1 = 0
	}

	// Check signifigance exception
	if cpu.cc == 0 && (cpu.progMask&SIGMASK) != 0 {
		cpu.fpregs[step.R1] = 0
		cpu.fpregs[step.R1|2] = 0
		return ircSignif
	}

	// Check if we need to normalize results
	if cpu.cc != 0 { // Only if non-zero result
		for (v1h & NMASKL) == 0 {
			v1h <<= 4
			v1h |= (v1l >> 60) & 0xf
			v1l <<= 4
			v1l &= UMASKL
			e1--
		}
		// Check if underflow
		if e1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				v1l = 0
				v1h = 0
				e1 = 0
				s1 = false
			}
		}
	} else { // true zero
		v1l = 0
		v1h = 0
		e1 = 0
		s1 = false
	}

	// Remmove the guard digit
	v1l >>= 4

	// Store result
	if e1 != 0 {
		v1h |= (uint64(e1) << 56) & EMASKL
		v1l |= (uint64(e1-14) << 56) & EMASKL
		if s1 {
			v1l |= MSIGNL
			v1h |= MMASKL
		}
	}

	cpu.fpregs[step.R1] = v1h
	cpu.fpregs[step.R1|2] = v1l

	return err
}

// Floating Point Multiply producing extended result.
func (cpu *cpu) opMXD(step *stepInfo) uint16 {
	// Check if registers are valid.
	if (step.R1 & 0xb) != 0 { // 0 or 4
		return ircSpec
	}

	// Extract number and adjust
	e1 := int((step.fsrc1 & EMASKL) >> 56)
	e2 := int((step.fsrc2 & EMASKL) >> 56)
	s1 := (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	// Make 32 bit and create guard digit
	v1 := step.fsrc1 & MMASKL
	v2 := step.fsrc2 & MMASKL

	// Pre-nomalize v1 and v2 */
	if v1 != 0 {
		for (v1 & NMASKL) == 0 {
			v1 <<= 4
			e1--
		}
	}
	if v2 != 0 {
		for (v2 & NMASKL) == 0 {
			v2 <<= 4
			e2--
		}
	}
	// Compute exponent
	e1 = e1 + e2 - 65

	// Add in guard digits
	v1 <<= 4
	v2 <<= 4
	var r uint64

	// Do actual multiply
	for range 56 {
		// Add if we need too
		if (v1 & 1) != 0 {
			r += v2
		}
		v1 >>= 1
		// Shift right by one
		if (r & 1) != 0 {
			v1 |= MSIGNL
		}
		r >>= 1
	}

	var err uint16
	// If overflow, shift right 4 bits
	if (r & EMASKL) != 0 {
		v1 >>= 4
		v1 |= (r & 0xf) << 60
		r >>= 4
		e1++

		// Check for overflow
		if e1 >= 128 {
			err = ircExpOver
		}
	}

	// Align the results
	if r != 0 {
		for (r & NMASKL) == 0 {
			r <<= 4
			r |= (v1 >> 60) & 0xf
			v1 <<= 4
			e1--
		}
		// Make 32 bit and create guard digit
		// Check if underflow
		if e1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				r = 0
				v1 = 0
				s1 = false
				e1 = 0
			}
		} else {
			e1 = 0
			s1 = false
		}
	}

	// Store result
	if e1 == 0 {
		r |= (uint64(e1) << 56) & EMASKL
		v1 |= (uint64(e1-14) << 56) & EMASKL
		if s1 {
			r |= MSIGNL
			v1 |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = r
	cpu.fpregs[step.R1|2] = v1
	return err
}

func (cpu *cpu) opMXR(step *stepInfo) uint16 {
	if (step.R1&0xb) != 0 || (step.R2&0xb) != 0 {
		return ircSpec
	}
	// Extract number and adjust
	e1 := int((step.fsrc1 & EMASKL) >> 56)
	e2 := int((step.fsrc2 & EMASKL) >> 56)
	s1 := (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	// Make 32 bit and create guard digit
	v1h := step.fsrc1 & MMASKL
	v1l := cpu.fpregs[step.R1|2] & MMASKL
	v2h := step.fsrc2 & MMASKL
	v2l := cpu.fpregs[step.R2|2] & MMASKL

	// Pre-nomalize v1 and v2 */
	if v1h != 0 {
		for (v1h & NMASKL) == 0 {
			v1h <<= 4
			v1h |= (v1l >> 56) & 0xf
			v1l <<= 4
			e1--
		}
	}
	if v2h != 0 {
		for (v2h & NMASKL) == 0 {
			v2h <<= 4
			v2h |= (v2l >> 56) & 0xf
			v2l <<= 4
			e2--
		}
	}

	// Create guard digit
	v1l <<= 4
	v1l &= UMASKL

	// Compute exponent
	e1 = e1 + e2 - 64

	// Do multiply
	rl := uint64(0)
	rh := uint64(0)
	for range 112 {
		if (v1l & 1) != 0 {
			rl += v2l
			rh += v2h
			if (rl & CMASKL) != 0 {
				rh++
			}
			rl &= XMASKL
		}
		// Shift right by one.
		rl >>= 1
		rh >>= 1
		if (rl & 0x8) != 0 {
			rh |= CMASKL >> 4
		}
		if (v1h & 1) != 0 {
			v1l |= CMASKL >> 4
		}
	}

	var err uint16
	// If overflow, shift right 4 bits
	if (rh & EMASKL) != 0 {
		rl >>= 4
		rl |= (rh & 0xf) << 60
		rh >>= 4
		e1++
		if e1 >= 128 {
			err = ircExpOver
		}
	}
	// Remove guard digit
	rl >>= 4
	if e1 != 0 {
		rh |= (uint64(e1) << 56) & EMASKL
		rl |= (uint64(e1-14) << 56) & EMASKL
		if s1 {
			rh |= MSIGNL
			rl |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = rh
	cpu.fpregs[step.R1|2] = rl
	return err
}
