/*
   IBM 370 Decimal instructions

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

// Load decimal number into temp storage
// return error or zero.
func (cpu *cpu) decLoad(data *[32]uint8, addr uint32, length int, sign *bool) uint16 {
	var err uint16

	// Point to end, and read backwards
	addr += uint32(length)
	j := 0
	// Read into data backwards
	for i := 0; i <= length; i++ {
		digitPair, errm := cpu.readByte(addr)
		if errm != 0 {
			return errm
		}
		digit := uint8(digitPair & 0xf)
		if j != 0 && digit > 0x9 {
			err = ircData
		}
		data[j] = digit
		j++
		digit = uint8((digitPair >> 4) & 0xf)
		if digit > 0x9 {
			err = ircData
		}
		data[j] = digit
		j++
		addr--
	}

	// Check if sign value and return it
	if data[0] == 0xb || data[0] == 0xd {
		*sign = true
	} else {
		*sign = false
		if data[0] < 0xa {
			return ircData
		}
	}
	return err
}

// Store decimal number into memory
// return error code.
func (cpu *cpu) decStore(data [32]uint8, addr uint32, length int) uint16 {
	addr += uint32(length)
	j := 0
	for i := 0; i <= length; i++ {
		digit := data[j] & 0xf
		j++
		digit |= (data[j] & 0xf) << 4
		j++
		err := cpu.writeByte(addr, uint32(digit))
		if err != 0 {
			return err
		}
		addr--
	}
	return 0
}

// Add or subtract a pair of BCD numbers.
func decAdd(l int, addsub bool, value1 *[32]uint8, value2 *[32]uint8) (uint8, bool) {
	var cy uint8
	var zero bool

	// Set carry
	if addsub {
		cy = 1
	} else {
		cy = 0
	}

	// Look for zero value
	zero = true
	for i := 1; i <= l; i++ {
		digit := value1[i]
		if addsub {
			digit = 0x9 - digit
		}
		acc := value2[i] + digit + cy
		if acc > 0x9 {
			acc += 0x6
		}
		value1[i] = acc & 0xf
		cy = (acc >> 4) & 0xf
		if (acc & 0xf) != 0 {
			zero = false
		}
	}
	return cy, zero
}

// Recomplement a number for decimal add.
func decRecomp(l int, value *[32]uint8) bool {
	// We need to recomplent the result
	cy := uint8(1)
	zero := true
	for i := 1; i <= l; i++ {
		acc := (0x9 - value[i]) + cy
		if acc > 0x9 {
			acc += 0x6
		}
		value[i] = acc & 0xf
		cy = (acc >> 4) & 0xf
		if value[i] != 0 {
			zero = false
		}
	}
	return zero
}

// Handle AP, SP, CP and ZAP instructions.
func (cpu *cpu) opDecAdd(step *stepInfo) uint16 {
	// ZAP = F8    00
	// CP  = F9    01
	// AP  = FA    10
	// SP  = FB    11
	var err uint16
	var value1 [32]uint8
	var value2 [32]uint8
	var sign1, sign2 bool
	var overflow bool

	addr1 := step.address1
	addr2 := step.address2
	len1 := int(step.R1)
	len2 := int(step.R2)

	length := len1
	if len2 > len1 {
		length = len2
	}
	// Always load second operand
	err = cpu.decLoad(&value2, addr2, len2, &sign2)
	if err != 0 {
		return err
	}

	// Subtract, change the sign
	if (step.opcode & 1) != 0 {
		sign2 = !sign2
	}

	// Length is 1 plus number of digits times two, including sign nibble
	length = 2*(length+1) - 1
	// On all but ZAP load first operand
	if (step.opcode & 3) != 0 {
		err = cpu.decLoad(&value1, addr1, len1, &sign1)
		if err != 0 {
			return err
		}
	} else {
		// For ZAP clear everything
		// for i := range 32 {
		// 	v1[i] = 0
		// }
		sign1 = false
	}

	addsub := sign1 != sign2

	cy, zero := decAdd(length, addsub, &value1, &value2)
	if cy != 0 {
		if addsub {
			sign1 = !sign1
		} else {
			overflow = true
		}
	} else {
		if addsub {
			// We need to recomplent the result
			zero = decRecomp(length, &value1)
		}
	}

	// Set flags
	if zero && !overflow {
		sign1 = false
	}
	cpu.cc = 0
	if !zero {
		if sign1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}

	// Do not store results for compare
	if step.opcode == OpCP {
		return err
	}

	// Not compare set status.
	if !zero && !overflow {
		// Start at l1 and go to l2 and see if any non-zero digits
		for i := (len1 + 1) * 2; i <= length; i++ {
			if value1[i] != 0 {
				overflow = true
				break
			}
		}
	}
	// Set sign
	if sign1 {
		value1[0] = 0xd
	} else {
		value1[0] = 0xc
	}
	// Store result
	err = cpu.decStore(value1, addr1, len1)
	if err != 0 {
		return err
	}
	// If overflow, set CC 3, if want overflows trigger trap
	if overflow {
		cpu.cc = 3
		if (cpu.progMask & DECOVER) != 0 {
			err = ircDecOver
		}
	}
	return err
}

// Handle SRP instruction.
func (cpu *cpu) opSRP(step *stepInfo) uint16 {
	var err uint16
	var value [32]uint8
	var sign bool
	var cy uint8

	overflow := false
	zero := true
	addr := step.address1
	length := int(step.R1)
	shift := int(step.address2 & 0x3f)

	// Load operand
	err = cpu.decLoad(&value, addr, length, &sign)
	if err != 0 {
		return err
	}

	if (shift & 0x20) != 0 { // shift to right
		var i, j int

		shift = 0x3f & (^shift + 1)
		if (value[shift] + step.R2) > 0x9 {
			cy = 1
		}
		j = shift + 1
		for i = 1; i < length; i++ {
			var acc uint8
			if j > length {
				acc = cy
			} else {
				acc = value[j] + cy
			}
			if acc > 0x9 {
				acc += 0x6
			}
			value[i] = acc & 0xf
			cy = (acc >> 4) & 0xf
			if value[i] != 0 {
				zero = false
			}
			j++
		}
	} else if shift != 0 { // Shift to left
		var i, j int

		// Check if we would move out of any non-zero digits
		for j = length; j > shift; j-- {
			if value[j] != 0 {
				overflow = true
			}
		}
		// Now shift digits
		for i = length; j > 0; i-- {
			value[i] = value[j]
			if value[i] != 0 {
				zero = false
			}
			j--
		}
		// Now fill zeros until at bottom
		for i > 0 {
			value[i] = 0
			i--
		}
	} else {
		// Check if number is zero
		for i := 1; i < length; i++ {
			if value[i] != 0 {
				zero = false
				break
			}
		}
	}

	if zero && !overflow {
		sign = false
	}
	cpu.cc = 0
	if !zero { // Really not zero
		if sign {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	if sign {
		value[0] = 0xd
	} else {
		value[0] = 0xc
	}
	err = cpu.decStore(value, addr, length)
	if err != 0 {
		return err
	}
	if overflow {
		cpu.cc = 3
		if (cpu.progMask & DECOVER) != 0 {
			err = ircDecOver
		}
	}
	return err
}

// Step for multiply decimal number.
func decMulstep(length int, pos int, value1 *[32]uint8, value2 *[32]uint8) {
	cy := uint8(0)
	resPos := 1
	for pos <= length {
		acc := value1[pos] + value2[resPos] + cy
		if acc > 0x9 {
			acc += 0x6
		}
		value1[pos] = acc & 0xf
		cy = (acc >> 4) & 0xf
		pos++
		resPos++
	}
}

// Decimal multiply.
func (cpu *cpu) opMP(step *stepInfo) uint16 {
	var err uint16
	var value1 [32]uint8
	var value2 [32]uint8
	var sign1, sign2 bool

	len1 := int(step.R1)
	len2 := int(step.R2)

	err = cpu.decLoad(&value2, step.address2, len2, &sign2)
	if err != 0 {
		return err
	}
	err = cpu.decLoad(&value1, step.address1, len1, &sign1)
	if err != 0 {
		return err
	}

	if len2 == len1 {
		return ircSpec
	}

	if len2 > 7 || len2 >= len1 {
		return ircData
	}

	len1 = (len1 + 1) * 2
	len2 = (len2 + 1) * 2

	// Verify that we have l2 zeros at start of v1
	for i := len1 - len2; i < len1; i++ {
		if value1[i] != 0 {
			return ircData
		}
	}

	// Compute sign
	if sign2 {
		sign1 = !sign1
	}

	// Start at end and work backwards
	for j := len1 - len2; j > 0; j-- {
		mul := value1[j]
		value1[j] = 0
		for mul != 0 {
			// Add multiplier to miltiplican
			decMulstep(len1, j, &value1, &value2)
			mul--
		}
	}
	if sign1 {
		value1[0] = 0xd
	} else {
		value1[0] = 0xc
	}
	return cpu.decStore(value1, step.address1, int(step.R1))
}

// BCD Packed Divide instruction.
func (cpu *cpu) opDP(step *stepInfo) uint16 {
	var err uint16
	var value1 [32]uint8
	var value2 [32]uint8
	var restor [32]uint8 // Restore holder
	var sign1, sign2 bool
	var cy uint8

	len1 := int(step.R1)
	len2 := int(step.R2)
	if len2 > 7 || len2 >= len1 {
		return ircSpec
	}

	err = cpu.decLoad(&value2, step.address2, len2, &sign2)
	if err != 0 {
		return err
	}

	err = cpu.decLoad(&value1, step.address1, len1, &sign1)
	if err != 0 {
		return err
	}

	len1 = (len1 + 1) * 2
	len2 = (len2 + 1) * 2

	// Compute sign
	if sign1 {
		sign2 = !sign2
	}

	for j := len1 - len2; j > 0; j-- {
		var k int

		// Current quotient digit
		q := uint8(0)
		for {
			// Subtract divisor
			cy = 1
			i := j
			k = 1
			for k < len2 {
				restor[i] = value1[i] // Save if we divide too far
				acc := value1[i] + (0x9 - value2[k]) + cy
				if acc > 0x9 {
					acc += 0x6
				}
				value1[i] = acc & 0xf
				cy = (acc >> 4) & 0xf
				k++
				i++
			}
			// Plus one more digit
			if i < 31 {
				acc := value1[i] + 9 + cy
				if acc > 0x9 {
					acc += 0x6
				}
				value1[i] = acc & 0xf
				cy = (acc >> 4) & 0xf
			}
			// If no borrow, so we are done with this digit
			if cy == 0 {
				// It is a no-no to have non-zero digit above size
				if q > 0 && (i+1) >= len1 {
					return ircDecDiv
				}
				if i < 31 {
					value1[i+1] = q // Save quotient digit
				}
				for i := j; k > 1; i++ {
					value1[i] = restor[i] // Restore previous
					k--
				}
			} else {
				q++
			}
			if q > 9 {
				return ircDecDiv
			}
			if cy == 0 {
				break
			}
		}
	}
	// Set sign of quotient.
	if sign2 {
		value1[len2] = 0xd
	} else {
		value1[len2] = 0xc
	}
	// Set sign of remainder.
	if sign1 {
		value1[0] = 0xd
	} else {
		value1[0] = 0xc
	}
	return cpu.decStore(value1, step.address1, int(step.R1))
}
