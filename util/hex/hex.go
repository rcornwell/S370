/*
 * S370 - Convert Hex to strings.
 *
 * Copyright 2024, Richard Cornwell
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

package hex

import "strings"

var hexMap = "0123456789ABCDEF"

func FormatWord(str *strings.Builder, word []uint32) {
	for _, full := range word {
		shift := 28
		for range 8 {
			str.WriteByte(hexMap[(full>>shift)&0xf])

			shift -= 4
		}
		str.WriteByte(' ')
	}
}

func FormatHalf(str *strings.Builder, space bool, half []uint16) {
	for _, word := range half {
		shift := 12
		for range 4 {
			str.WriteByte(hexMap[(word>>shift)&0xf])
			shift -= 4
		}
		if space {
			str.WriteByte(' ')
		}
	}
	if !space {
		str.WriteByte(' ')
	}
}

func FormatDisp(str *strings.Builder, disp []byte) {
	str.WriteByte(hexMap[disp[0]&0xf])
	str.WriteByte(hexMap[(disp[1]>>4)&0xf])
	str.WriteByte(hexMap[disp[1]&0xf])
}

func FormatAddr(str *strings.Builder, disp []byte) {
	str.WriteByte(hexMap[(disp[0]>>4)&0xf])
	str.WriteByte(' ')
	str.WriteByte(hexMap[disp[0]&0xf])
	str.WriteByte(hexMap[(disp[1]>>4)&0xf])
	str.WriteByte(hexMap[disp[1]&0xf])
}

func FormatBytes(str *strings.Builder, space bool, data []uint8) {
	for _, by := range data {
		str.WriteByte(hexMap[(by>>4)&0xf])
		str.WriteByte(hexMap[by&0xf])
		if space {
			str.WriteByte(' ')
		}
	}
}

func FormatByte(str *strings.Builder, data byte) {
	str.WriteByte(hexMap[(data>>4)&0xf])
	str.WriteByte(hexMap[data&0xf])
}

func FormatDigit(str *strings.Builder, data byte) {
	str.WriteByte(hexMap[data&0xf])
}

func FormatDecimal(str *strings.Builder, num byte) {
	if num >= 100 {
		str.WriteByte(hexMap[num/100])
		num %= 100
	}
	if num >= 10 {
		str.WriteByte(hexMap[num/10])
		num %= 10
	}
	str.WriteByte(hexMap[num])
}
