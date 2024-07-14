/*
   IBM 370 Floating point instructions

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
func (cpu *cpuState) opFPHalf(step *stepInfo) uint16 {
	var err uint16

	// Split number apart
	exponent := int((step.fsrc2 & EMASKL) >> 56)
	sign := (step.fsrc2 & MSIGNL) != 0
	// if (step.opcode & 0x10) != 0 {
	// 	step.fsrc2 &= HMASKL
	// }
	// Create guard digit
	step.fsrc2 = (step.fsrc2 & MMASKL) << 4
	// Divide by 2
	step.fsrc2 >>= 1
	// If not zero normalize result
	if step.fsrc2 != 0 {
		for (step.fsrc2 & SNMASKL) == 0 {
			step.fsrc2 <<= 4
			exponent--
		}
		// Check if underflow
		if exponent < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				sign = false
				step.fsrc2 = 0
				exponent = 0
			}
		}

		// Remove guard digit
		step.fsrc2 >>= 4
	}

	// Check for zero
	if step.fsrc2 == 0 {
		sign = false
		exponent = 0
	}

	// Restore result
	step.fsrc2 |= (uint64(exponent) << 56) & EMASKL
	if sign {
		step.fsrc2 |= MSIGNL
	}

	// Store results.
	if (step.opcode & 0x10) == 0 {
		cpu.fpregs[step.R1] = step.fsrc2
	} else {
		cpu.fpregs[step.R1] = (step.fsrc2 & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return err
}

// Floating load register.
func (cpu *cpuState) opFPLoad(step *stepInfo) uint16 {
	if (step.opcode & 0x10) == 0 {
		cpu.fpregs[step.R1] = step.fsrc2
	} else {
		cpu.fpregs[step.R1] = (step.fsrc2 & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return 0
}

// Floating point load register with sign change.
func (cpu *cpuState) opFPLCS(step *stepInfo) uint16 {
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
	if (fsrc1 & MMASKL) != 0 {
		if (step.fsrc2 & MSIGNL) != 0 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	return 0
}

// Floating point store register double.
func (cpu *cpuState) opSTD(step *stepInfo) uint16 {
	t := uint32(step.fsrc1 & LMASKL)
	if err := cpu.writeFull(step.address1+4, t); err != 0 {
		return err
	}
	t = uint32((step.fsrc1 >> 32) & LMASKL)
	return cpu.writeFull(step.address1, t)
}

// Floating point store register short.
func (cpu *cpuState) opSTE(step *stepInfo) uint16 {
	t := uint32((step.fsrc1 >> 32) & LMASKL)
	return cpu.writeFull(step.address1, t)
}

// Floating point compare short.
func (cpu *cpuState) opCE(step *stepInfo) uint16 {
	// Extract number and adjust
	exponent1 := int((step.fsrc1 & EMASKL) >> 56)
	exponent2 := int((step.fsrc2 & EMASKL) >> 56)
	sign1 := (step.fsrc1 & MSIGNL) != 0
	sign2 := (step.fsrc2 & MSIGNL) != 0

	// Make 32 bit and create guard digit
	value1 := uint32(step.fsrc1>>28) & XMASK
	value2 := uint32(step.fsrc2>>28) & XMASK

	expDiff := exponent1 - exponent2
	if expDiff > 0 {
		if expDiff > 8 {
			value2 = 0
		} else {
			// Shift v2 right if v1 larger expo - expo
			value2 >>= 4 * expDiff
		}
	} else {
		if expDiff < 0 {
			if expDiff < -8 {
				value1 = 0
			} else {
				// Shift v1 right if v2 larger expo - expo
				value1 >>= 4 * -expDiff
			}
		}
	}

	// Exponents should be equal now.

	// Subtract results
	var diff uint32

	if sign1 == sign2 {
		// Same signs do subtract
		value2 ^= XMASK
		diff = value1 + value2 + 1
		if (diff & CMASK) != 0 {
			diff &= XMASK
		} else {
			sign1 = !sign1
			diff ^= XMASK
			diff++
		}
	} else {
		diff = value1 + value2
	}

	// If v1 not normal shift left + expo
	if (diff & CMASK) != 0 {
		diff >>= 4
	}

	// Set condition code
	cpu.cc = 0
	if diff != 0 {
		if sign1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	return 0
}

// Floating point short add and subtract.
func (cpu *cpuState) opFPAdd(step *stepInfo) uint16 {
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
	exponent1 := int((step.fsrc1 & EMASKL) >> 56)
	exponent2 := int((step.fsrc2 & EMASKL) >> 56)
	sign1 := (step.fsrc1 & MSIGNL) != 0
	sign2 := (step.fsrc2 & MSIGNL) != 0

	// Make 32 bit and create guard digit
	value1 := uint32(step.fsrc1>>28) & XMASK
	value2 := uint32(step.fsrc2>>28) & XMASK

	expDifg := exponent1 - exponent2
	if expDifg > 0 {
		if expDifg > 8 {
			value2 = 0
		} else {
			// Shift v2 right if v1 larger expo - expo
			value2 >>= 4 * expDifg
		}
	} else if expDifg < 0 {
		if expDifg < -8 {
			value1 = 0
		} else {
			// Shift v1 right if v2 larger expo - expo
			value1 >>= 4 * -expDifg
		}
		exponent1 = exponent2
	}

	var sum uint32

	// Exponents should be equal now.
	// Add results
	if sign1 != sign2 {
		// Different signs do subtract
		value2 ^= XMASK
		sum = value1 + value2 + 1
		if (sum & CMASK) != 0 {
			sum &= XMASK
		} else {
			sign1 = !sign1
			sum ^= XMASK
			sum++
		}
	} else {
		sum = value1 + value2
	}

	var err uint16
	// If v1 not normal shift left + expo
	if (sum & CMASK) != 0 {
		sum >>= 4
		exponent1++
		if exponent1 >= 128 {
			err = ircExpOver
		}
	}

	// Set condition codes
	cpu.cc = 0
	// If not unnormalize opcode
	if (step.opcode & 0x0e) != 0x0e {
		if sum != 0 {
			if sign1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.progMask & SIGMASK) == 0 {
				exponent1 = 0
			}
			sign1 = false
		}
	} else {
		if (sum & 0xffffff0) != 0 {
			if sign1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.progMask & SIGMASK) == 0 {
				exponent1 = 0
			}
			sum = 0
			sign1 = false
		}
	}

	// Check signifigance exceptions
	if cpu.cc == 0 && (cpu.progMask&SIGMASK) != 0 {
		err = ircSignif
	} else
	// Check if we are normalized addition
	if (step.opcode & 0x0e) != 0x0e {
		if cpu.cc != 0 { // Only if non-zero result
			for (sum & SNMASK) == 0 {
				sum <<= 4
				exponent1--
			}
			// Check if underflow
			if exponent1 < 0 {
				if (cpu.progMask & EXPUNDER) != 0 {
					err = ircExpUnder
				} else {
					sum = 0
					sign1 = false
					exponent1 = 0
				}
			}
		}
	}

	// Remove guard digit
	sum >>= 4

	sum |= uint32(exponent1<<24) & EMASK
	if cpu.cc != 0 && sign1 {
		sum |= MSIGN
	}
	// Store result
	cpu.fpregs[step.R1] = (uint64(sum) << 32) | (cpu.fpregs[step.R1] & LMASKL)
	return err
}

// Double floating compare.
func (cpu *cpuState) opCD(step *stepInfo) uint16 {
	// OP_CD	0x69
	// OP_CDR	0x29

	// Extract number and adjust
	exponent1 := int((step.fsrc1 & EMASKL) >> 56)
	exponent2 := int((step.fsrc2 & EMASKL) >> 56)
	sign1 := (step.fsrc1 & MSIGNL) != 0
	sign2 := (step.fsrc2 & MSIGNL) != 0

	// Make 32 bit and create guard digit
	value1 := step.fsrc1 & MMASKL
	value2 := step.fsrc2 & MMASKL

	// Add guard digit
	value1 <<= 4
	value2 <<= 4

	// Align based on exponent difference
	expDiff := exponent1 - exponent2
	if expDiff > 0 {
		if expDiff > 17 {
			value2 = 0
		} else {
			// Shift v2 right if v1 larger expo - expo
			value2 >>= 4 * expDiff
		}
	} else if expDiff < 0 {
		if expDiff < -17 {
			value1 = 0
		} else {
			// Shift v1 right if v2 larger expo - expo
			value1 >>= 4 * -expDiff
		}
	}

	// Exponents should be equal now.

	// Subtract results
	var diff uint64
	if sign1 == sign2 {
		// Same signs do subtract
		value2 ^= XMASKL
		diff = value1 + value2 + 1
		if (diff & CMASKL) != 0 {
			diff &= XMASKL
		} else {
			sign1 = !sign1
			diff ^= XMASKL
			diff++
		}
	} else {
		diff = value1 + value2
	}

	// If v1 not normal shift left + expo
	// if (d & CMASKL) != 0 {
	// 	d >>= 4
	// }

	// Set condition code
	cpu.cc = 0
	if diff != 0 {
		if sign1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	return 0
}

// Floating point double add and subtract.
func (cpu *cpuState) opFPAddD(step *stepInfo) uint16 {
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
	exponent1 := int((step.fsrc1 & EMASKL) >> 56)
	exponent2 := int((step.fsrc2 & EMASKL) >> 56)
	sign1 := (step.fsrc1 & MSIGNL) != 0
	sign2 := (step.fsrc2 & MSIGNL) != 0

	// Make 32 bit and create guard digit
	value1 := step.fsrc1 & MMASKL
	value2 := step.fsrc2 & MMASKL

	// Add guard digit
	value1 <<= 4
	value2 <<= 4

	// Align based on exponent difference
	expDiff := exponent1 - exponent2
	if expDiff > 0 {
		if expDiff > 17 {
			value2 = 0
		} else {
			// Shift v2 right if v1 larger expo - expo
			value2 >>= 4 * expDiff
		}
	} else if expDiff < 0 {
		if expDiff < -17 {
			value1 = 0
		} else {
			// Shift v1 right if v2 larger expo - expo
			value1 >>= 4 * -expDiff
		}
		exponent1 = exponent2
	}

	// Exponents should be equal now.
	var sum uint64 // Add results

	if sign1 != sign2 {
		// Different signs do subtract
		value2 ^= XMASKL
		sum = value1 + value2 + 1
		if (sum & CMASKL) != 0 {
			sum &= XMASKL
		} else {
			sign1 = !sign1
			sum ^= XMASKL
			sum++
		}
	} else {
		sum = value1 + value2
	}

	var err uint16
	// If v1 not normal shift left + expo
	if (sum & CMASKL) != 0 {
		sum >>= 4
		exponent1++
		if exponent1 >= 128 {
			err = ircExpOver
		}
	}

	// Set condition codes
	cpu.cc = 0
	// If not unnormalize opcode
	if (step.opcode & 0x0e) != 0x0e {
		if sum != 0 {
			if sign1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.progMask & SIGMASK) == 0 {
				exponent1 = 0
			}
			sign1 = false
		}
	} else {
		if (sum & UMASKL) != 0 {
			if sign1 {
				cpu.cc = 1
			} else {
				cpu.cc = 2
			}
		} else {
			if (cpu.progMask & SIGMASK) == 0 {
				exponent1 = 0
			}
			sum = 0
			sign1 = false
		}
	}

	// Check signifigance exceptions
	if cpu.cc == 0 && (cpu.progMask&SIGMASK) != 0 {
		err = ircSignif
		goto fpstore
	}

	// Check if we are normalized addition
	if (step.opcode & 0x0e) != 0x0e {
		if cpu.cc != 0 { // Only if non-zero result
			for (sum & SNMASKL) == 0 {
				sum <<= 4
				exponent1--
			}
			// Check if underflow
			if exponent1 < 0 {
				if (cpu.progMask & EXPUNDER) != 0 {
					err = ircExpUnder
				} else {
					sum = 0
					sign1 = false
					exponent1 = 0
				}
			}
		}
	}

	// Return true zero
	if cpu.cc == 0 {
		sign1 = false
		sum = 0
	}

	// Remove guard digit
	sum >>= 4

fpstore:
	// Save result
	sum |= uint64(exponent1<<56) & EMASKL
	if cpu.cc != 0 && sign1 {
		sum |= MSIGNL
	}
	// Store result
	cpu.fpregs[step.R1] = sum
	return err
}

// Floating point multiply.
func (cpu *cpuState) opFPMul(step *stepInfo) uint16 {
	// MDR	2c
	// MER  3c
	// ME   7c
	// MD   6c

	// Extract number and adjust
	exponent1 := int((step.fsrc1 & EMASKL) >> 56)
	exponent2 := int((step.fsrc2 & EMASKL) >> 56)
	sign := (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	// Make 32 bit and create guard digit
	value1 := step.fsrc1 & MMASKL
	value2 := step.fsrc2 & MMASKL

	// Pre-nomalize v1 and v2 */
	if value1 != 0 {
		for (value1 & NMASKL) == 0 {
			value1 <<= 4
			exponent1--
		}
	}
	if value2 != 0 {
		for (value2 & NMASKL) == 0 {
			value2 <<= 4
			exponent2--
		}
	}
	// Compute exponent
	exponent1 = exponent1 + exponent2 - 65

	// Add in guard digits
	value1 <<= 4
	value2 <<= 4
	var product uint64 // Result

	// Do actual multiply
	for range 60 {
		// Add if we need too
		if (value1 & 1) != 0 {
			product += value2
		}
		// Shift right by one
		value1 >>= 1
		product >>= 1
	}

	// If overflow, shift right 4 bits
	if (product & EMASKL) != 0 {
		product >>= 4
		exponent1++
	}

	var err uint16
	// Check for overflow
	if exponent1 >= 128 {
		err = ircExpOver
	}

	// Align the results
	if product != 0 {
		for (product & NMASKL) == 0 {
			product <<= 4
			exponent1--
		}

		// Check if underflow
		if exponent1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				product = 0
				sign = false
				exponent1 = 0
			}
		}
	} else {
		exponent1 = 0
		sign = false
	}

	// Store result'
	product |= (uint64(exponent1) << 56) & EMASKL
	if sign {
		product |= MSIGNL
	}
	cpu.fpregs[step.R1] = product
	return err
}

// Floating point divide.
func (cpu *cpuState) opFPDiv(step *stepInfo) uint16 {
	// MDR	2c
	// MER  3c
	// ME   7c
	// MD   6c

	// Extract number and adjust
	exponent1 := int((step.fsrc1 & EMASKL) >> 56)
	exponent2 := int((step.fsrc2 & EMASKL) >> 56)
	sign := (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	value1 := step.fsrc1 & MMASKL
	value2 := step.fsrc2 & MMASKL

	if value2 == 0 {
		return ircFPDiv
	}

	// Pre-nomalize v1 and v2 */
	if value1 != 0 {
		for (value1 & NMASKL) == 0 {
			value1 <<= 4
			exponent1--
		}
	}
	if value2 != 0 {
		for (value2 & NMASKL) == 0 {
			value2 <<= 4
			exponent2--
		}
	}
	// Compute exponent
	exponent1 = exponent1 - exponent2 + 64

	// Shift numbers up 4 bits so as not to lose precision below
	value1 <<= 4
	value2 <<= 4

	// Check if we need to adjust divsor if it larger then dividend
	if value1 > value2 {
		value1 >>= 4
		exponent1++
	}

	// Change sign of v2 so we can add
	value2 ^= XMASKL
	value2++
	var quotent uint64 // Result

	// Do divide
	for range 57 {
		// Shift left by one
		value1 <<= 1
		// Subtract remainder from dividend
		temp := value1 + value2
		// Shift quotent left one bit
		quotent <<= 1
		// If remainder larger then divsor replace
		if (temp & CMASKL) != 0 {
			value1 = temp
			quotent |= 1
		}
		value1 &= XMASKL
	}

	if quotent == 0x01ffffffffffffff {
		quotent++
	}
	quotent >>= 1

	// If overflow, shift right 4 bits
	if (quotent & EMASKL) != 0 {
		quotent >>= 4
		exponent1++
	}

	var err uint16
	// Check for overflow
	if exponent1 >= 128 {
		err = ircExpOver
	}

	// Align the results
	if quotent != 0 {
		for (quotent & NMASKL) == 0 {
			quotent <<= 4
			exponent1--
		}

		// Check if underflow
		if exponent1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				quotent = 0
				sign = false
				exponent1 = 0
			}
		}
	} else {
		exponent1 = 0
		sign = false
	}

	// Store result'
	quotent |= (uint64(exponent1) << 56) & EMASKL
	if sign {
		quotent |= MSIGNL
	}

	if (step.opcode & 0x10) == 0 {
		cpu.fpregs[step.R1] = quotent
	} else {
		cpu.fpregs[step.R1] = (quotent & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	}
	return err
}

// Extended precision load round.
func (cpu *cpuState) opLRER(step *stepInfo) uint16 {
	var err uint16
	value := step.fsrc2

	// Check if round bit is one.
	if (value & RMASKL) != 0 {
		// Extract number and adjust
		exponent := int((value & EMASKL) >> 56)
		sign := (value & MSIGNL) != 0
		value = (value & MMASKL) + RMASKL

		// Normalize if needed
		if (value & SNMASKL) != 0 {
			value >>= 4
			exponent++
			if exponent > 128 {
				err = ircExpOver
			}
		}
		// Store results
		value |= (uint64(exponent) << 56) & EMASKL
		if sign {
			value |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = (value & HMASKL) | (cpu.fpregs[step.R1] & LMASKL)
	return err
}

func (cpu *cpuState) opLRDR(step *stepInfo) uint16 {
	if (step.R2 & 0xb) != 0 {
		return ircSpec
	}
	var err uint16
	value := cpu.fpregs[step.R2]
	if (cpu.fpregs[step.R2|2] & 0x0080000000000000) != 0 {
		// Extract numbers and adjust
		exponent := int((value & EMASKL) >> 56)
		sign := (value & MSIGNL) != 0
		value = (value & MMASKL) + 1
		if (value & SNMASKL) != 0 {
			value >>= 4
			exponent++
			if exponent > 128 {
				err = ircExpOver
			}
		}
		// Store results
		value |= (uint64(exponent) << 56) & EMASKL
		if sign {
			value |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = value
	return err
}

// Handle extended floating point add.
func (cpu *cpuState) opAXR(step *stepInfo) uint16 {
	if (step.R1&0xb) != 0 || (step.R2&0xb) != 0 {
		return ircSpec
	}
	var err uint16
	value1Low := cpu.fpregs[step.R1]
	value1High := cpu.fpregs[step.R1|2] & MMASKL
	value2Low := cpu.fpregs[step.R2]
	value2High := cpu.fpregs[step.R2|2] & MMASKL

	// Extract numbers
	exponent1 := int((value1Low & EMASKL) >> 56)
	sign1 := (value1Low & MSIGNL) != 0
	value1Low = (value1Low & MMASKL) + 1
	exponent2 := int((value2Low & EMASKL) >> 56)
	sign2 := (value2Low & MSIGNL) != 0
	value2Low = (value2Low & MMASKL) + 1
	if (step.opcode & 1) != 0 {
		sign2 = !sign2
	}

	// Create Guard digits.
	value1Low <<= 4
	value2Low <<= 4

	// Align values
	expDiff := exponent1 - exponent2
	if expDiff > 0 {
		if expDiff > 15 {
			value2Low = 0
			value2High = 0
		} else {
			for range expDiff {
				value2Low >>= 4
				value2Low |= (value2High & 0xf) << 60
				value2High >>= 4
			}
		}
	} else if expDiff < 0 {
		if expDiff < -15 {
			value1Low = 0
			value1High = 0
		} else {
			for range -expDiff {
				value1Low >>= 4
				value1Low |= (value1High & 0xf) << 60
				value1High >>= 4
			}
		}
		exponent1 = exponent2
	}
	// Exponents should be equal now.

	// Add results
	if sign1 != sign2 {
		// Different signs do subtract
		value2High ^= XMASKL
		value2Low ^= XMASKL
		if value2Low == XMASKL {
			value2High++
		}
		value2Low++
		// Do actual add
		value1Low += value2Low
		value1High += value2High
		// Check if overflow lower value
		if (value1Low & CMASKL) != 0 {
			value1Low &= XMASKL
			value1High++
		}
		// Check if carry out, if not change sign of result
		if (value1High & CMASKL) != 0 {
			value1High &= XMASKL
		} else {
			sign1 = !sign1
			value1Low ^= XMASKL
			value1High ^= XMASKL
			if value1Low == XMASKL {
				value1High++
			}
			value1Low++
		}
	} else {
		// Do add
		value1Low += value2Low
		value1High += value2High
		// If lower overflowed, increment upper
		if (value1Low & CMASKL) != 0 {
			value1Low &= XMASKL
			value1High++
		}
	}
	value1Low += value2Low
	value1High += value2High
	if (value1Low & CMASKL) != 0 {
		value1Low &= XMASKL
		value1High++
	}

	// If overflow shift right 4 bits
	if (value1High & NMASKL) != 0 {
		value1Low >>= 4
		value1Low |= (value1High & 0xf) << 60
		value1High >>= 4
		exponent1++
		if exponent1 >= 128 {
			err = ircExpOver
		}
	}

	// Set condition codes
	cpu.cc = 0
	if (value1Low | value1High) != 0 {
		if sign1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	} else {
		sign1 = false
		exponent1 = 0
	}

	// Check signifigance exception
	if cpu.cc == 0 && (cpu.progMask&SIGMASK) != 0 {
		cpu.fpregs[step.R1] = 0
		cpu.fpregs[step.R1|2] = 0
		return ircSignif
	}

	// Check if we need to normalize results
	if cpu.cc != 0 { // Only if non-zero result
		for (value1High & NMASKL) == 0 {
			value1High <<= 4
			value1High |= (value1Low >> 60) & 0xf
			value1Low <<= 4
			value1Low &= UMASKL
			exponent1--
		}
		// Check if underflow
		if exponent1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				value1Low = 0
				value1High = 0
				exponent1 = 0
				sign1 = false
			}
		}
	} else { // true zero
		value1Low = 0
		value1High = 0
		exponent1 = 0
		sign1 = false
	}

	// Remmove the guard digit
	value1Low >>= 4

	// Store result
	if exponent1 != 0 {
		value1High |= (uint64(exponent1) << 56) & EMASKL
		value1Low |= (uint64(exponent1-14) << 56) & EMASKL
		if sign1 {
			value1Low |= MSIGNL
			value1High |= MMASKL
		}
	}

	cpu.fpregs[step.R1] = value1High
	cpu.fpregs[step.R1|2] = value1Low

	return err
}

// Floating Point Multiply producing extended result.
func (cpu *cpuState) opMXD(step *stepInfo) uint16 {
	// Check if registers are valid.
	if (step.R1 & 0xb) != 0 { // 0 or 4
		return ircSpec
	}

	// Extract number and adjust
	exponent1 := int((step.fsrc1 & EMASKL) >> 56)
	exponent2 := int((step.fsrc2 & EMASKL) >> 56)
	sign1 := (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	// Make 32 bit and create guard digit
	value1 := step.fsrc1 & MMASKL
	value2 := step.fsrc2 & MMASKL

	// Pre-nomalize v1 and v2 */
	if value1 != 0 {
		for (value1 & NMASKL) == 0 {
			value1 <<= 4
			exponent1--
		}
	}
	if value2 != 0 {
		for (value2 & NMASKL) == 0 {
			value2 <<= 4
			exponent2--
		}
	}
	// Compute exponent
	exponent1 = exponent1 + exponent2 - 65

	// Add in guard digits
	value1 <<= 4
	value2 <<= 4
	var product uint64

	// Do actual multiply
	for range 56 {
		// Add if we need too
		if (value1 & 1) != 0 {
			product += value2
		}
		value1 >>= 1
		// Shift right by one
		if (product & 1) != 0 {
			value1 |= MSIGNL
		}
		product >>= 1
	}

	var err uint16
	// If overflow, shift right 4 bits
	if (product & EMASKL) != 0 {
		value1 >>= 4
		value1 |= (product & 0xf) << 60
		product >>= 4
		exponent1++

		// Check for overflow
		if exponent1 >= 128 {
			err = ircExpOver
		}
	}

	// Align the results
	if product != 0 {
		for (product & NMASKL) == 0 {
			product <<= 4
			product |= (value1 >> 60) & 0xf
			value1 <<= 4
			exponent1--
		}
		// Make 32 bit and create guard digit
		// Check if underflow
		if exponent1 < 0 {
			if (cpu.progMask & EXPUNDER) != 0 {
				err = ircExpUnder
			} else {
				product = 0
				value1 = 0
				sign1 = false
				exponent1 = 0
			}
		} else {
			exponent1 = 0
			sign1 = false
		}
	}

	// Store result
	if exponent1 == 0 {
		product |= (uint64(exponent1) << 56) & EMASKL
		value1 |= (uint64(exponent1-14) << 56) & EMASKL
		if sign1 {
			product |= MSIGNL
			value1 |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = product
	cpu.fpregs[step.R1|2] = value1
	return err
}

func (cpu *cpuState) opMXR(step *stepInfo) uint16 {
	if (step.R1&0xb) != 0 || (step.R2&0xb) != 0 {
		return ircSpec
	}
	// Extract number and adjust
	exponent1 := int((step.fsrc1 & EMASKL) >> 56)
	exponent2 := int((step.fsrc2 & EMASKL) >> 56)
	sign := (step.fsrc1 & MSIGNL) != (step.fsrc2 & MSIGNL)

	// Make 32 bit and create guard digit
	value1High := step.fsrc1 & MMASKL
	value1Low := cpu.fpregs[step.R1|2] & MMASKL
	value2High := step.fsrc2 & MMASKL
	value2Low := cpu.fpregs[step.R2|2] & MMASKL

	// Pre-nomalize v1 and v2 */
	if value1High != 0 {
		for (value1High & NMASKL) == 0 {
			value1High <<= 4
			value1High |= (value1Low >> 56) & 0xf
			value1Low <<= 4
			exponent1--
		}
	}
	if value2High != 0 {
		for (value2High & NMASKL) == 0 {
			value2High <<= 4
			value2High |= (value2Low >> 56) & 0xf
			value2Low <<= 4
			exponent2--
		}
	}

	// Create guard digit
	value1Low <<= 4
	value1Low &= UMASKL

	// Compute exponent
	exponent1 = exponent1 + exponent2 - 64

	// Do multiply
	productLow := uint64(0)
	productHigh := uint64(0)
	for range 112 {
		if (value1Low & 1) != 0 {
			productLow += value2Low
			productHigh += value2High
			if (productLow & CMASKL) != 0 {
				productHigh++
			}
			productLow &= XMASKL
		}
		// Shift right by one.
		productLow >>= 1
		productHigh >>= 1
		if (productLow & 0x8) != 0 {
			productHigh |= CMASKL >> 4
		}
		if (value1High & 1) != 0 {
			value1Low |= CMASKL >> 4
		}
	}

	var err uint16
	// If overflow, shift right 4 bits
	if (productHigh & EMASKL) != 0 {
		productLow >>= 4
		productLow |= (productHigh & 0xf) << 60
		productHigh >>= 4
		exponent1++
		if exponent1 >= 128 {
			err = ircExpOver
		}
	}
	// Remove guard digit
	productLow >>= 4
	if exponent1 != 0 {
		productHigh |= (uint64(exponent1) << 56) & EMASKL
		productLow |= (uint64(exponent1-14) << 56) & EMASKL
		if sign {
			productHigh |= MSIGNL
			productLow |= MSIGNL
		}
	}
	cpu.fpregs[step.R1] = productHigh
	cpu.fpregs[step.R1|2] = productLow
	return err
}
