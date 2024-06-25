/*
 * Generic Card read/punch conversion routines.
 *
 * Copyright (c) 2021-2024, Richard Cornwell
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 */

package card

// Convert EBCDIC character into hollerith code.
func EBCDICToHol(v uint8) uint16 {
	return ebcdicToHolTable[v]
}

// Return hollerith code for ebcdic character.
func HolToEBCDIC(v uint16) uint16 {
	return holToEBCDICTable[v]
}

// Returns the ASCII character of a given Hollerith code.
func HolToASCII(v uint16) uint8 {
	return holToASCIITable29[v]
}

// Return hollerith code for ascii character.
func ASCIIToHol(v uint8) uint16 {
	return asciiToHol29[v]
}

func ASCIIToSix(v uint8) uint8 {
	return uint8(asciiToSix[v])
}

// Convert BCD character into hollerith code.
func BcdToHol(bcd uint8) uint16 {
	// Handle spacce correctly
	if bcd == 0 {
		return 0x82 // 0 to 82 punch
	}

	if bcd == 0o20 {
		return 0 // 20 no punch
	}

	hol := uint16(0)
	// Convert top row
	switch bcd & 0o60 {
	default:
		hol = 0o0000
	case 0o00:
		hol = 0o0000
	case 0o20:
		hol = 0o1000
	case 0o40:
		hol = 0o2000
	case 0o60:
		hol = 0o4000
	}

	// Handle case of 10 special
	// only 032 is punched as 8-2
	if (bcd&017) == 10 && (bcd&060) != 0o20 {
		hol |= 1 << 9
		return hol
	}

	// Convert to 0-9 row
	bcd &= 0o17
	if bcd > 9 {
		hol |= 0o2 // Col 8
		bcd -= 8
	}
	if bcd != 0 {
		hol |= 1 << (9 - bcd)
	}
	return hol
}

// Returns the BCD of the hollerith code or 0x7f if error.
func HolToBcd(hol uint16) uint8 {
	var bcd uint8

	// Convert rows 10,11,12
	switch hol & 0o7000 {
	case 0o0000:
		bcd = 0
	case 0o1000: // 10 Punch
		if (hol & 0x1ff) == 0 {
			return 10
		}
		bcd = 0o20
	case 0o2000: // 11 punch
		bcd = 0o40
	case 0o3000: // 11-10 punch
		bcd = 0o52
	case 0o4000: // 12 punch
		bcd = 0o60
	case 0o5000: // 12-10 Punch
		bcd = 0o72
	default: // Punch in 10,11,12 rows
		return 0x7f
	}

	hol &= 0o777         // Mask rows 0-9
	if (hol & 02) != 0 { //Check if row 8 punched
		bcd += 8
		hol &= 0o775 // Clear row 8.
	}

	// Convert rows 0-9
	for hol != 0 && (hol&0o1000) == 0 {
		bcd++
		hol <<= 1
	}

	// Any more columns punched?
	if (hol & 0o777) != 0 {
		return 0o177
	}
	return bcd
}
