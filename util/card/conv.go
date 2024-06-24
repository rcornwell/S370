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

// Convert EBCDIC character into hollerith code
func EbcdicToHol(v uint8) uint16 {
	return ebcdicToHolTable[v]
}

// Return hollerith code for ebcdic character
func HolToEbcdic(v uint16) uint16 {
	return holToEbcdicTable[v]
}

// Returns the ASCII character of a given Hollerith code
func HolToAscii(v uint16) uint8 {
	return holToAsciiTable29[v]
}

// Return hollerith code for ascii character
func AsciiToHol(v uint8) uint16 {
	return asciiToHol29[v]
}

func AsciiToSix(v uint8) uint8 {
	return uint8(asciiToSix[v])
}

// Convert BCD character into hollerith code
func BcdToHol(bcd uint8) uint16 {

	// Handle spacce correctly
	if bcd == 0 {
		return 0x82 // 0 to 82 punch
	}
	if bcd == 020 {
		return 0 // 20 no punch
	}

	hol := uint16(0)
	// Convert top row
	switch bcd & 060 {
	default:
		hol = 00000
	case 000:
		hol = 00000
	case 020:
		hol = 01000
	case 040:
		hol = 02000
	case 060:
		hol = 04000
	}

	// Handle case of 10 special
	// only 032 is punched as 8-2
	if (bcd&017) == 10 && (bcd&060) != 020 {
		hol |= 1 << 9
		return hol
	}

	// Convert to 0-9 row
	bcd &= 017
	if bcd > 9 {
		hol |= 02 // Col 8
		bcd -= 8
	}
	if bcd != 0 {
		hol |= 1 << (9 - bcd)
	}
	return hol
}

// Returns the BCD of the hollerith code or 0x7f if error
func HolToBcd(hol uint16) uint8 {
	var bcd uint8

	// Convert rows 10,11,12
	switch hol & 07000 {
	case 00000:
		bcd = 0
	case 01000: // 10 Punch
		if (hol & 0x1ff) == 0 {
			return 10
		}
		bcd = 020
	case 02000: // 11 punch
		bcd = 040
	case 03000: // 11-10 punch
		bcd = 052
	case 04000: // 12 punch
		bcd = 060
	case 05000: // 12-10 Punch
		bcd = 072
	default: // Punch in 10,11,12 rows
		return 0x7f
	}

	hol &= 0777          // Mask rows 0-9
	if (hol & 02) != 0 { //Check if row 8 punched
		bcd += 8
		hol &= 0775 // Clear row 8.
	}

	// Convert rows 0-9
	for hol != 0 && (hol&01000) == 0 {
		bcd++
		hol <<= 1
	}

	// Any more columns punched?
	if (hol & 0777) != 0 {
		return 0177
	}
	return bcd
}
