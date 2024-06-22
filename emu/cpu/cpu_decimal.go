package cpu

/* IBM 370 Decimal instructions

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

// Load decimal number into temp storage
// return error or zero
func (cpu *cpu) decLoad(data *[32]uint8, addr uint32, len uint8, sign *bool) uint16 {
	var err uint16 = 0
	var t uint32

	a := addr + uint32(len)
	// Clear result
	for i := range len {
		data[i] = 0
	}

	j := 0
	// Read into data backwards
	for range len {
		t, err = cpu.readByte(a)
		if err != 0 {
			return err
		}
		t2 := uint8(t & 0xf)
		if j != 0 && t2 > 0x9 {
			err = IRC_DATA
		}
		data[j] = t2
		j++
		t2 = uint8((t >> 4) & 0xf)
		if t2 > 0x9 {
			err = IRC_DATA
		}
		data[j] = t2
		j++
		a--
	}

	// Check if sign value and return it
	if data[0] == 0xb || data[0] == 0xd {
		*sign = true
	} else {
		*sign = false
	}
	if data[0] < 0xa {
		err = IRC_DATA
	}
	return err
}

// Store decimal number into memory
// return error code
func (cpu *cpu) decStore(data [32]uint8, addr uint32, len uint8) uint16 {
	a := addr + uint32(len)
	j := 0
	for range len {
		t := data[j] & 0xf
		j++
		t |= (data[j] & 0xf) << 4
		j++
		if err := cpu.writeByte(a, uint32(t)); err != 0 {
			return err
		}
		a--
	}
	return 0
}

// Add or subtract a pair of BCD numbers
func dec_add(l uint8, addsub bool, v1 *[32]uint8, v2 *[32]uint8) (uint8, bool) {
	var cy uint8
	var z bool
	if addsub {
		cy = 1
	} else {
		cy = 0
	}
	z = true
	for i := 1; i <= int(l); i++ {
		d := v1[1]
		if addsub {
			d = 0x9 - d
		}
		acc := v2[i] + d + cy
		if acc > 0x9 {
			acc += 0x6
		}
		v1[i] = acc & 0xf
		cy = (acc >> 4) & 0xf
		if (acc & 0xf) != 0 {
			z = false
		}
	}
	return cy, z
}

func dec_recomp(l uint8, v1 *[32]uint8) bool {
	// We need to recomplent the result
	cy := uint8(1)
	z := true
	for i := 1; i <= int(l); i++ {
		acc := (0x9 - v1[i]) + cy
		if acc > 0x9 {
			acc += 0x6
		}
		v1[i] = acc & 0xf
		cy = (acc >> 4) & 0xf
		if v1[i] != 0 {
			z = false
		}
	}
	return z
}

// Handle AP, SP, CP and ZAP instructions.
func (cpu *cpu) opDecAdd(step *stepInfo) uint16 {
	// ZAP = F8    00
	// CP  = F9    01
	// AP  = FA    10
	// SP  = FB    11
	var err uint16 = 0
	var v1 [32]uint8
	var v2 [32]uint8
	var s1, s2 bool
	var addsub bool
	var cy uint8
	var z bool
	var ov bool

	ov = false
	a1 := step.address1
	a2 := step.address2
	l1 := step.R1
	l2 := step.R2

	l := l1
	if l2 > l1 {
		l = l2
	}
	// Always load second operand
	if err = cpu.decLoad(&v2, a2, l2, &s2); err != 0 {
		return err
	}

	if (step.opcode & 1) != 0 {
		s2 = !s2
	}

	// Length is 1 plus number of digits times two, including sign nibble
	l = 2*(l+1) - 1
	// On all but ZAP load first operand
	if (step.opcode & 3) != 0 {
		if err = cpu.decLoad(&v1, a1, l1, &s1); err != 0 {
			return err
		} else {
			for i := range 32 {
				v1[i] = 0
			}
			s1 = false
		}
	}

	if s1 != s2 {
		addsub = true
	} else {
		addsub = false
	}

	cy, z = dec_add(l, addsub, &v1, &v2)

	if cy != 0 {
		if addsub {
			s1 = !s1
		} else {
			ov = true
		}
	} else {
		if addsub {
			// We need to recomplent the result
			z = dec_recomp(l, &v1)
		}
	}

	// Set flags
	if z && !ov {
		s1 = false
	}
	cpu.cc = 0
	if !z {
		if s1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}

	if (step.opcode & 3) != 1 {
		if !z && !ov {
			// Start at l1 and go to l2 and see if any non-zero digits
			for i := (l1 + 1) * 2; i <= l; i++ {
				if v1[i] != 0 {
					ov = true
					break
				}
			}
		}
		if s1 {
			v1[0] = 0xd
		} else {
			v1[0] = 0xc
		}
		if err = cpu.decStore(v1, a1, l1); err != 0 {
			return err
		}
		if ov {
			cpu.cc = 3
			if (cpu.progMask & DECOVER) != 0 {
				err = IRC_DECOVR
			}
		}
	}
	return err
}

// Handle SRP instruction
func (cpu *cpu) opSRP(step *stepInfo) uint16 {
	var err uint16 = 0
	var v1 [32]uint8
	var s1 bool
	var cy uint8 = 0
	var i, j int

	ov := false
	z := true
	a1 := step.address1
	l1 := step.R1
	shift := int(step.address2 & 0x3f)

	// Load operand
	if err = cpu.decLoad(&v1, a1, l1, &s1); err != 0 {
		return err
	}

	if (shift & 0x20) != 0 { // shift to right
		shift = 0x3f & (^shift + 1)
		if (v1[shift] + step.R2) > 0x9 {
			cy = 1
		}
		j = shift + 1
		for i = 1; i < int(l1); i++ {
			var acc uint8
			if j > int(l1) {
				acc = cy
			} else {
				acc = v1[j] + cy
			}
			if acc > 0x9 {
				acc += 0x6
			}
			v1[i] = acc & 0xf
			cy = (acc >> 4) & 0xf
			if v1[i] != 0 {
				z = false
			}
			j++
		}
	} else if shift != 0 { // Shift to left
		// Check if we would move out of any non-zero digits
		for j = int(l1); j > shift; j-- {
			if v1[j] != 0 {
				ov = true
			}
		}
		// Now shift digits
		for i = int(l1); j > 0; i-- {
			v1[i] = v1[j]
			if v1[i] != 0 {
				z = false
			}
			j--
		}
		// Now fill zeros until at bottom
		for i > 0 {
			v1[i] = 0
			i--
		}
	} else {
		// Check if number is zero
		for i = 1; i < int(l1); i++ {
			if v1[i] != 0 {
				z = false
				break
			}
		}
	}

	if z && !ov {
		s1 = false
	}
	cpu.cc = 0
	if !z { // Really not zero
		if s1 {
			cpu.cc = 1
		} else {
			cpu.cc = 2
		}
	}
	if s1 {
		v1[0] = 0xd
	} else {
		v1[0] = 0xc
	}
	if err = cpu.decStore(v1, a1, l1); err != 0 {
		return err
	}
	if ov {
		cpu.cc = 3
		if (cpu.progMask & DECOVER) != 0 {
			err = IRC_DECOVR
		}
	}
	return err
}

// Step for multiply decimal number
func dec_mulstep(l int, s1 int, v1 *[32]uint8, v2 *[32]uint8) {
	var cy uint8
	cy = 0
	s2 := 1
	for s1 <= l {
		acc := v1[s1] + v2[s2] + cy
		if acc > 0x9 {
			acc += 0x6
		}
		v1[s1] = acc & 0xf
		cy = (acc >> 4) & 0xf
		s1++
		s2++
	}
}

// Decimal multiply
func (cpu *cpu) opMP(step *stepInfo) uint16 {
	var err uint16 = 0
	var v1 [32]uint8
	var v2 [32]uint8
	var s1, s2 bool

	if step.R2 == step.R1 {
		return IRC_SPEC
	}

	if step.R2 > 7 || step.R2 >= step.R1 {
		return IRC_DATA
	}
	if err = cpu.decLoad(&v2, step.address2, step.R2, &s2); err != 0 {
		return err
	}
	if err = cpu.decLoad(&v1, step.address1, step.R1, &s1); err != 0 {
		return err
	}

	l1 := int(step.R1)
	l2 := int(step.R2)
	l1 = (l1 + 1) * 2
	l2 = (l2 + 1) * 2

	// Verify that we have l2 zeros at start of v1
	for i := l1 - l2; i < l1; i++ {
		if v1[i] != 0 {
			return IRC_DATA
		}
	}

	// Compute sign
	if s2 {
		s1 = !s1
	}

	// Start at end and work backwards
	for j := l1 - l2; j > 0; j-- {
		mul := v1[j]
		v1[j] = 0
		for mul != 0 {
			// Add multiplier to miltiplican
			dec_mulstep(l1, j, &v1, &v2)
			mul--
		}
	}
	if s1 {
		v1[0] = 0xd
	} else {
		v1[0] = 0xc
	}
	return cpu.decStore(v1, step.address1, uint8(l1))
}

// BCD Packed Divide instruction
func (cpu *cpu) opDP(step *stepInfo) uint16 {
	var err uint16 = 0
	var v1 [32]uint8
	var v2 [32]uint8
	var r [32]uint8
	var s1, s2 bool
	var cy uint8

	if step.R2 > 7 || step.R2 >= step.R1 {
		return IRC_DATA
	}
	if err = cpu.decLoad(&v2, step.address2, step.R2, &s2); err != 0 {
		return err
	}
	if err = cpu.decLoad(&v1, step.address1, step.R1, &s1); err != 0 {
		return err
	}

	// Clear result
	for i := range 32 {
		r[i] = 0
	}

	l1 := int(step.R1)
	l2 := int(step.R2)
	l1 = (l1 + 1) * 2
	l2 = (l2 + 1) * 2

	// Compute sign
	if s2 {
		s1 = !s1
	}

	for j := l1 - l2; j > 0; j-- {
		var k int
		q := uint8(0)
		cy = 1
		for cy != 0 {
			// Subtract divisor
			cy = 1
			i := j
			for k = 1; k < l2; k++ {
				r[i] = v1[i] // Save if we divide too far
				acc := v1[i] + (0x9 - v2[k]) + cy
				if acc > 0x9 {
					acc += 0x6
				}
				v1[i] = acc & 0xf
				cy = (acc >> 4) & 0xf
				i++
			}
			// Plus one more digit
			if i < 31 {
				acc := v1[i] + 9 + cy
				if acc > 0x9 {
					acc += 0x6
				}
				v1[i] = acc & 0xf
				cy = (acc >> 4) & 0xf
			}
			// If no borrow, so we are done with this digit
			if cy == 0 {
				// It is a no-no to have non-zero digit above size
				if q > 0 && (i+1) > l1 {
					return IRC_DECDIV
				}
				v1[i+1] = q // Save quotient digit
				for i := j; k > 1; i++ {
					v1[i] = r[i] // Restore previous
				}
			} else {
				q++
			}
			if q > 9 {
				return IRC_DECDIV
			}
		}
	}
	// Set sign of quotient
	if s2 {
		v1[l2] = 0xd
	} else {
		v1[l2] = 0xc
	}

	if s1 {
		v1[0] = 0xd
	} else {
		v1[0] = 0xc
	}
	return cpu.decStore(v1, step.address1, uint8(l1))
}
