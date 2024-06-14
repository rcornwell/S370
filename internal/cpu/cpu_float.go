package cpu

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

// Floating point half register
func (cpu *CPU) op_hd(step *stepInfo) uint16 {
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
			if (cpu.pmask & EXPUNDER) != 0 {
				return IRC_EXPUND
			} else {
				sign = false
				step.fsrc2 = 0
				e1 = 0
			}
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
	return cpu.op_ld(step)
}

// Floating load register
func (cpu *CPU) op_ld(step *stepInfo) uint16 {
	if (step.opcode & 0x10) == 0 {
		cpu.fpregs[step.R1] = step.fsrc2
	} else {
		cpu.fpregs[step.R1] = (step.fsrc2 & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return 0
}

// Floating point load register with sign change
func (cpu *CPU) op_lcs(step *stepInfo) uint16 {
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

// Floating point store register double
func (cpu *CPU) op_std(step *stepInfo) uint16 {
	var error uint16

	t := uint32(step.fsrc1 & LMASKL)
	if error = cpu.writeFull(step.address1+4, t); error != 0 {
		return error
	}
	t = uint32((step.fsrc1 >> 32) & LMASKL)
	return cpu.writeFull(step.address1, t)
}

// Floating point store register short
func (cpu *CPU) op_ste(step *stepInfo) uint16 {
	t := uint32((step.fsrc1 >> 32) & LMASKL)
	return cpu.writeFull(step.address1, t)
}

// Floating point compare short
func (cpu *CPU) op_ce(step *stepInfo) uint16 {
	var e1, e2 int
	var s1, s2 bool
	var d uint32

	// Extract number and adjust
	e1 = int((step.fsrc1 & EMASKL) >> 56)
	e2 = int((step.fsrc2 & EMASKL) >> 56)
	s1 = (step.fsrc1 & MSIGNL) != 0
	s2 = (step.fsrc2 & MSIGNL) != 0

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

// Floating point short add and subtract
func (cpu *CPU) op_sh_as(step *stepInfo) uint16 {
	// SER 3B
	// SUR 3F
	// SE  7B
	// SU  7F
	// AER 3A
	// AUR 3E
	// AE  7A
	// AU  7E
	var e1, e2 int
	var s1, s2 bool
	var d uint32
	var error uint16 = 0

	// If subrtact change sign
	if (step.opcode & 1) != 0 {
		step.fsrc2 ^= MSIGNL
	}

	// Extract number and adjust
	e1 = int((step.fsrc1 & EMASKL) >> 56)
	e2 = int((step.fsrc2 & EMASKL) >> 56)
	s1 = (step.fsrc1 & MSIGNL) != 0
	s2 = (step.fsrc2 & MSIGNL) != 0

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

	// Exponents should be equal now.
	// Add results
	if s1 != s2 {
		// Different signs do subtract
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
		e1++
		if e1 >= 128 {
			return IRC_EXPOVR
		}
	}

	// Set condition codes
	cpu.cc = 0
	// If not unnormalize opcode
	if (step.opcode & 0x0e) != 0x0e {
		if d != 0 {
			if s1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.pmask & SIGMASK) == 0 {
				e1 = 0
			}
			s1 = false
		}
	} else {
		if (d & 0xffffff0) != 0 {
			if s1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.pmask & SIGMASK) == 0 {
				e1 = 0
			}
			d = 0
			s1 = false
		}
	}

	// Check signifigance exceptions
	if cpu.cc == 0 && (cpu.pmask&SIGMASK) != 0 {
		error = IRC_SIGNIF
	} else {
		// Check if we are normalized addition
		if (step.opcode & 0x0e) != 0x0e {
			if cpu.cc != 0 { // Only if non-zero result
				for (d & SNMASK) == 0 {
					d <<= 4
					e1 = 0
				}
				// Check if underflow
				if e1 < 0 {
					if (cpu.pmask & EXPUNDER) != 0 {
						return IRC_EXPUND
					} else {
						d = 0
						s1 = false
						e1 = 0
					}
				}
			}

			// Remove guard digit
			d >>= 4
		}
	}
	d |= uint32(e1<<24) & EMASK
	if cpu.cc != 0 && s1 {
		d |= MSIGN
	}
	// Store result
	cpu.fpregs[step.R1] = (uint64(d) << 32) | (cpu.fpregs[step.R1] & LMASKL)
	return error
}

// Double floating compare
func (cpu *CPU) op_cd(step *stepInfo) uint16 {
	// OP_CD	0x69
	// OP_CDR	0x29
	var e1, e2 int
	var s1, s2 bool
	var d uint64

	// Extract number and adjust
	e1 = int((step.fsrc1 & EMASKL) >> 56)
	e2 = int((step.fsrc2 & EMASKL) >> 56)
	s1 = (step.fsrc1 & MSIGNL) != 0
	s2 = (step.fsrc2 & MSIGNL) != 0

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

// Floating point double add and subtract
func (cpu *CPU) op_db_as(step *stepInfo) uint16 {
	// SDR 3B
	// SWR 3F
	// SD  6B
	// SW  6F
	// ADR 3A
	// AWR 3E
	// AD  6A
	// AW  6E
	var e1, e2 int
	var s1, s2 bool
	var d uint64
	var error uint16 = 0

	// If subrtact change sign
	if (step.opcode & 1) != 0 {
		step.fsrc2 ^= MSIGNL
	}

	// Extract number and adjust
	e1 = int((step.fsrc1 & EMASKL) >> 56)
	e2 = int((step.fsrc2 & EMASKL) >> 56)
	s1 = (step.fsrc1 & MSIGNL) != 0
	s2 = (step.fsrc2 & MSIGNL) != 0

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
	// Add results
	if s1 != s2 {
		// Different signs do subtract
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
		e1++
		if e1 >= 128 {
			return IRC_EXPOVR
		}
	}

	// Set condition codes
	cpu.cc = 0
	// If not unnormalize opcode
	if (step.opcode & 0x0e) != 0x0e {
		if d != 0 {
			if s1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.pmask & SIGMASK) == 0 {
				e1 = 0
			}
			s1 = false
		}
	} else {
		if (d & 0xffffff0) != 0 {
			if s1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.pmask & SIGMASK) == 0 {
				e1 = 0
			}
			d = 0
			s1 = false
		}
	}

	// Check signifigance exceptions
	if cpu.cc == 0 && (cpu.pmask&SIGMASK) != 0 {
		error = IRC_SIGNIF
	} else {
		// Check if we are normalized addition
		if (step.opcode & 0x0e) != 0x0e {
			if cpu.cc != 0 { // Only if non-zero result
				for (d & UMASKL) == 0 {
					d <<= 4
					e1 = 0
				}
				// Check if underflow
				if e1 < 0 {
					if (cpu.pmask & EXPUNDER) != 0 {
						return IRC_EXPUND
					} else {
						d = 0
						s1 = false
						e1 = 0
					}
				}
			}

			// Remove guard digit
			d >>= 4
		}
	}
	d |= uint64(e1<<56) & EMASKL
	if cpu.cc != 0 && s1 {
		d |= MSIGNL
	}
	// Store result
	cpu.fpregs[step.R1] = d
	return error
}

// Floating point multiply
func (cpu *CPU) op_fp_mpy(step *stepInfo) uint16 {
	// MDR	2c
	// MER  3c
	// ME   7c
	// MD   6c
	var e1, e2 int
	var s1 bool
	var d uint64
	var error uint16 = 0

	// Extract number and adjust
	e1 = int((step.fsrc1 & EMASKL) >> 56)
	e2 = int((step.fsrc2 & EMASKL) >> 56)
	s1 = (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

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
	d = 0

	// Do actual multiply
	for i := 0; i < 60; i++ {
		// Add if we need too
		if (v1 & 1) != 0 {
			d += v2
		}
		// Shift right by one
		v1 >>= 1
		d >>= 1
	}

	// If overflow, shift right 4 bits
	if (d & EMASKL) != 0 {
		d >>= 4
		e1++
	}

	// Check for overflow
	if e1 >= 128 {
		error = IRC_EXPOVR
	}

	// Align the results
	if d != 0 {
		for (d & NMASKL) == 0 {
			d <<= 4
			e1--
		} // Make 32 bit and create guard digit
		// Check if underflow
		if e1 < 0 {
			if (cpu.pmask & EXPUNDER) != 0 {
				error = IRC_EXPUND
			} else {
				d = 0
				s1 = false
				e1 = 0
			}
		} else {
			e1 = 0
			s1 = false
		}
	}

	// Store result'
	d |= (uint64(e1) << 56) & EMASKL
	if s1 {
		d |= MSIGNL
	}
	if (step.opcode&0x10) == 0 || (step.opcode&0xf) == 0xc {
		cpu.fpregs[step.R1] = d
	} else {
		cpu.fpregs[step.R1] = (d & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return error
}

// Floating point divide
func (cpu *CPU) op_fp_div(step *stepInfo) uint16 {
	// MDR	2c
	// MER  3c
	// ME   7c
	// MD   6c
	var e1, e2 int
	var s1 bool
	var d uint64
	var error uint16 = 0

	// Extract number and adjust
	e1 = int((step.fsrc1 & EMASKL) >> 56)
	e2 = int((step.fsrc2 & EMASKL) >> 56)
	s1 = (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	v1 := step.fsrc1 & MMASKL
	v2 := step.fsrc2 & MMASKL

	if v2 == 0 {
		return IRC_FPDIV
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
	d = 0

	// Do divide
	for i := 57; i > 0; i-- {
		// Shift left by one
		v1 <<= 1
		// Subtract remainder from dividend
		t := v1 + v2
		// Shift quotent left one bit
		d <<= 1
		// If remainder larger then divsor replace
		if (t & CMASKL) != 0 {
			v1 = t
			d |= 1
		}
		v1 &= XMASKL
	}

	if d == 0x01ffffffffffffff {
		d++
	}
	d >>= 1

	// If overflow, shift right 4 bits
	if (d & EMASKL) != 0 {
		d >>= 4
		e1++
	}

	// Check for overflow
	if e1 >= 128 {
		error = IRC_EXPOVR
	}

	// Align the results
	if d != 0 {
		for (d & NMASKL) == 0 {
			d <<= 4
			e1--
		} // Make 32 bit and create guard digit
		// Check if underflow
		if e1 < 0 {
			if (cpu.pmask & EXPUNDER) != 0 {
				error = IRC_EXPUND
			} else {
				d = 0
				s1 = false
				e1 = 0
			}
		} else {
			e1 = 0
			s1 = false
		}
	}

	// Store results
	d |= (uint64(e1) << 56) & EMASKL
	if s1 {
		d |= MSIGNL
	}
	if (step.opcode&0x10) == 0 || (step.opcode&0xf) == 0xc {
		cpu.fpregs[step.R1] = d
	} else {
		cpu.fpregs[step.R1] = (d & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return error
}

func (cpu *CPU) op_mxr(step *stepInfo) uint16  { return 0 }
func (cpu *CPU) op_mxdr(step *stepInfo) uint16 { return 0 }
func (cpu *CPU) op_mx(step *stepInfo) uint16   { return 0 }

// Extended precision load round
func (cpu *CPU) op_lre(step *stepInfo) uint16 {
	var error uint16 = 0
	var v uint64 = step.fsrc2

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
				error = IRC_EXPOVR
			}
		}
		// Store results
		v |= (uint64(e) << 56) & EMASKL
		if s {
			v |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = (v & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	return error
}

func (cpu *CPU) op_lrd(step *stepInfo) uint16 {
	if (step.R2 & 0xb) != 0 {
		return IRC_SPEC
	}
	var error uint16 = 0
	var v uint64 = cpu.fpregs[step.R2]
	if (cpu.fpregs[step.R2|2] & 0x0080000000000000) != 0 {
		// Extract numbers and adjust
		e := int((v & EMASKL) >> 56)
		s := (v & MSIGNL) != 0
		v = (v & MMASKL) + 1
		if (v & SNMASKL) != 0 {
			v >>= 4
			e++
			if e > 128 {
				error = IRC_EXPOVR
			}
		}
		// Store results
		v |= (uint64(e) << 56) & EMASKL
		if s {
			v |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = v
	return error
}

func (cpu *CPU) op_axr(step *stepInfo) uint16 {
	if (step.R1&0xb) != 0 || (step.R2&0xb) != 0 {
		return IRC_SPEC
	}
	var error uint16 = 0
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
		v1l = v1l + v2l
		v1h = v1h + v2h
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
	v1l = v1l + v2l
	v1h = v1h + v2h
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
			error = IRC_EXPOVR
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
	if cpu.cc == 0 && (cpu.pmask&SIGMASK) != 0 {
		cpu.fpregs[step.R1] = 0
		cpu.fpregs[step.R1|2] = 0
		return IRC_SIGNIF
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
			if (cpu.pmask & EXPUNDER) != 0 {
				error = IRC_EXPUND
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

	return error
}

//         case OP_MXD:
//                 if ((cpu_unit[0].flags & FEAT_EFP) == 0) {
//                     storepsw(OPPSW, IRC_OPR);
//                     goto supress;
//                 }
//                 if ((reg1 & 0xB) != 0) {
//                     storepsw(OPPSW, IRC_SPEC);
//                     goto supress;
//                 }

//                 /* src2L already has DP number */

//                 /* Fall through */
//         case OP_MXDR:
//                 if ((cpu_unit[0].flags & FEAT_EFP) == 0) {
//                     storepsw(OPPSW, IRC_OPR);
//                     goto supress;
//                 }
//                 if ((reg1 & 0xB) != 0) {
//                     storepsw(OPPSW, IRC_SPEC);
//                     goto supress;
//                 }

//                 /* Extract numbers and adjust */
//                 e1 = (src1L & EMASKL) >> 56;
//                 e2 = (src2L & EMASKL) >> 56;
//                 fill = 0;
//                 if ((src1L & MSIGNL) != (src2L & MSIGNL))
//                    fill = 1;
//                 src1L &= MMASKL;
//                 src2L &= MMASKL;

//                 /* Pre-nomalize src2 and src1 */
//                 if (src2L != 0) {
//                     while ((src2L & NMASKL) == 0) {
//                        src2L <<= 4;
//                        e2 --;
//                     }
//                 }
//                 if (src1L != 0) {
//                     while ((src1L & NMASKL) == 0) {
//                        src1L <<= 4;
//                        e1 --;
//                     }
//                 }

//                 /* Compute exponent */
//                 e1 = e1 + e2 - 64;

//                 destL = 0;
//                 /* Do multiply */
//                 for (temp = 0; temp < 56; temp++) {
//                      /* Add if we need too */
//                      if (src1L & 1)
//                          destL += src2L;
//                      /* Shift right by one */
//                      src1L >>= 1;
//                      if (destL & 1)
//                         src1L |= MSIGNL;
//                      destL >>= 1;
//                 }

//                 /* If overflow, shift right 4 bits */
//                 if (destL & EMASKL) {
//                    src1L >>= 4;
//                    src1L |= (destL & 0xF) << 60;
//                    destL >>= 4;
//                    e1 ++;
//                    if (e1 >= 128) {
//                        storepsw(OPPSW, IRC_EXPOVR);
//                    }
//                 }

//                 /* Align the results */
//                 if ((destL | src1L) != 0) {
//                     while ((destL & NMASKL) == 0) {
//                         destL <<= 4;
//                         destL |= (src1L >> 60) & 0xf;
//                         src1L <<= 4;
//                         e1 --;
//                     }
//                     /* Check if underflow */
//                     if (e1 < 0) {
//                         if (pmsk & EXPUND) {
//                             storepsw(OPPSW, IRC_EXPUND);
//                         } else {
//                             destL = src1L = 0;
//                             fill = e1 = 0;
//                         }
//                     }
//                 } else
//                     e1 = fill = 0;
//                 if (e1) {
//                     destL |= (((t_uint64)e1) << 56) & EMASKL;
//                     src1L |= ((t_uint64)(e1 - 14) << 56) & EMASKL;
//                     if (fill) {
//                        destL |= MSIGNL;
//                        src1L |= MSIGNL;
//                     }
//                 }
//                 fpregs[reg1] = destL;
//                 fpregs[reg1|2] = src1L;
//                 break;

//         case OP_MXR:
//                 if ((cpu_unit[0].flags & FEAT_EFP) == 0) {
//                     storepsw(OPPSW, IRC_OPR);
//                     goto supress;
//                 }
//                 if ((reg1 & 0xBB) != 0) {
//                     storepsw(OPPSW, IRC_SPEC);
//                     goto supress;
//                 }

//                 /* Extract numbers and adjust */
//                 e1 = (src1L & EMASKL) >> 56;
//                 e2 = (src2L & EMASKL) >> 56;
//                 fill = 0;
//                 if ((src1L & MSIGNL) != (src2L & MSIGNL))
//                    fill = 1;
//                 src1L &= MMASKL;
//                 src2L = fpregs[reg1|2] & MMASKL;
//                 /* Normalize first operand */
//                 if (src1L != 0) {
//                     while ((src1L & NMASKL) == 0) {
//                        src1L <<= 4;
//                        src1L |= (src2L >> 56) & 0xf;
//                        src2L <<= 4;
//                        e1 --;
//                     }
//                 }
//                 src2L <<= 4;
//                 src2L &= UMASKL;

//                 /* Normalize second operand. */
//                 fpregs[reg1|2] = fpregs[R2(reg)|2] & MMASKL;
//                 fpregs[reg1] = fpregs[R2(reg)] & MMASKL;
//                 /* Save second operand in result */
//                 destL = fpregs[reg1] | fpregs[reg1|2];
//                 if (destL != 0) {
//                     while ((fpregs[reg1] & 0x00f00000000000LL) == 0) {
//                        fpregs[reg1] <<= 4;
//                        fpregs[reg1|2] <<= 4;
//                        fpregs[reg1] |= fpregs[reg1|2] >> 56;
//                        e2--;
//                     }
//                     fpregs[reg1|2] &= MMASKL;
//                 }

//                 /* Compute exponent */
//                 e1 = e1 + e2 - 64;

//                 /* Do multiply */
//                 destL = 0;
//                 dest2L = 0;
//                 for (temp = 0; temp < 112; temp++) {
//                      /* Add if we need too */
//                      if (fpregs[reg1|2] & 1) {
//                          destL += src1L;
//                          dest2L += src2L;
//                          if (dest2L & CMASKL)
//                              destL ++;
//                          dest2L &= XMASKL;
//                       }
//                       /* Shift right by one */
//                       dest2L >>= 1;
//                       destL >>= 1;
//                       if (destL & 0x8) {
//                           dest2L |= 0x0800000000000000LL;
//                       }
//                       if (fpregs[reg1] & 1) {
//                           fpregs[reg1|2] |= CMASKL >> 4;
//                       }
//                       fpregs[reg1|2] >>= 1;
//                       fpregs[reg1] >>= 1;
//                 }
//                 /* If overflow, shift right 4 bits */
//                 if (destL & EMASKL) {
//                    src1L >>= 4;
//                    src1L |= (destL & 0xF) << 60;
//                    destL >>= 4;
//                    e1 ++;
//                    if (e1 >= 128) {
//                        storepsw(OPPSW, IRC_EXPOVR);
//                    }
//                 }
//                 src1L >>= 4;
//                 if (e1) {
//                     destL |= (((t_uint64)e1) << 56) & EMASKL;
//                     src1L |= ((t_uint64)(e1 - 14) << 56) & EMASKL;
//                     if (fill) {
//                        destL |= MSIGNL;
//                        src1L |= MSIGNL;
//                     }
//                 }
//                 fpregs[reg1] = destL;
//                 fpregs[reg1|2] = src1L;
//                 break;

//         default:   /* Unknown op code */
//                 storepsw(OPPSW, IRC_OPR);
//                 goto supress;
//         }
//         if (per_en && (cregs[9] & 0x10000000) != 0 && (cregs[9] & 0xffff & per_mod) != 0)
//            per_code |= 0x1000;

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
