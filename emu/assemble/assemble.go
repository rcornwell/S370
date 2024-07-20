/*
	   IBM 370 Assembler

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
package assembler

import (
	"errors"
	"strings"
	"unicode"

	op "github.com/rcornwell/S370/emu/opcodemap"
)

const (
	tyRR = 1 + iota
	tyRX
	tyRS
	tySI
	tySS
	tyS
	ty370 // Specific 0xB2 370 instruction

	zeroOp = 1 + iota
	oneOp
	imdOp
	twoOp
	addrOp
)

type opcode struct {
	opCode  int // Opcode string.
	opType  int // Opcode type.
	opFlags int // Opcode flags.
}

// Length of opcode types.
var lenMap = map[int]int{
	tyRR:  2,
	tyRX:  4,
	tyRS:  4,
	tySI:  4,
	tySS:  6,
	tyS:   4,
	ty370: 4,
}

var opMap = map[string]opcode{
	"SPM":   {op.OpSPM, tyRR, oneOp},
	"BALR":  {op.OpBALR, tyRR, 0},
	"BCTR":  {op.OpBCTR, tyRR, 0},
	"BCR":   {op.OpBCR, tyRR, 0},
	"SSK":   {op.OpSSK, tyRR, 0},
	"ISK":   {op.OpISK, tyRR, 0},
	"SVC":   {op.OpSVC, tyRR, imdOp},
	"BASR":  {op.OpBASR, tyRR, 0},
	"LPR":   {op.OpLPR, tyRR, 0},
	"LNR":   {op.OpLNR, tyRR, 0},
	"LTR":   {op.OpLTR, tyRR, 0},
	"LCR":   {op.OpLCR, tyRR, 0},
	"NR":    {op.OpNR, tyRR, 0},
	"OR":    {op.OpOR, tyRR, 0},
	"XR":    {op.OpXR, tyRR, 0},
	"CLR":   {op.OpCLR, tyRR, 0},
	"CR":    {op.OpCR, tyRR, 0},
	"LR":    {op.OpLR, tyRR, 0},
	"AR":    {op.OpAR, tyRR, 0},
	"SR":    {op.OpSR, tyRR, 0},
	"MR":    {op.OpMR, tyRR, 0},
	"DR":    {op.OpDR, tyRR, 0},
	"ALR":   {op.OpALR, tyRR, 0},
	"SLR":   {op.OpSLR, tyRR, 0},
	"LPDR":  {op.OpLPDR, tyRR, 0},
	"LNDR":  {op.OpLNDR, tyRR, 0},
	"LTDR":  {op.OpLTDR, tyRR, 0},
	"LCDR":  {op.OpLCDR, tyRR, 0},
	"HDR":   {op.OpHDR, tyRR, 0},
	"LRDR":  {op.OpLRDR, tyRR, 0},
	"MXR":   {op.OpMXR, tyRR, 0},
	"MXDR":  {op.OpMXDR, tyRR, 0},
	"LDR":   {op.OpLDR, tyRR, 0},
	"CDR":   {op.OpCDR, tyRR, 0},
	"ADR":   {op.OpADR, tyRR, 0},
	"SDR":   {op.OpSDR, tyRR, 0},
	"MDR":   {op.OpMDR, tyRR, 0},
	"DDR":   {op.OpDDR, tyRR, 0},
	"AWR":   {op.OpAWR, tyRR, 0},
	"SWR":   {op.OpSWR, tyRR, 0},
	"LPER":  {op.OpLPER, tyRR, 0},
	"LNER":  {op.OpLNER, tyRR, 0},
	"LTER":  {op.OpLTER, tyRR, 0},
	"LCER":  {op.OpLCER, tyRR, 0},
	"HER":   {op.OpHER, tyRR, 0},
	"LRER":  {op.OpLRER, tyRR, 0},
	"AXR":   {op.OpAXR, tyRR, 0},
	"SXR":   {op.OpSXR, tyRR, 0},
	"LER":   {op.OpLER, tyRR, 0},
	"CER":   {op.OpCER, tyRR, 0},
	"AER":   {op.OpAER, tyRR, 0},
	"SER":   {op.OpSER, tyRR, 0},
	"MER":   {op.OpMER, tyRR, 0},
	"DER":   {op.OpDER, tyRR, 0},
	"AUR":   {op.OpAUR, tyRR, 0},
	"SUR":   {op.OpSUR, tyRR, 0},
	"STH":   {op.OpSTH, tyRX, 0},
	"LA":    {op.OpLA, tyRX, 0},
	"STC":   {op.OpSTC, tyRX, 0},
	"IC":    {op.OpIC, tyRX, 0},
	"EX":    {op.OpEX, tyRX, 0},
	"BAL":   {op.OpBAL, tyRX, 0},
	"BCT":   {op.OpBCT, tyRX, 0},
	"BC":    {op.OpBC, tyRX, 0},
	"LH":    {op.OpLH, tyRX, 0},
	"CH":    {op.OpCH, tyRX, 0},
	"AH":    {op.OpAH, tyRX, 0},
	"SH":    {op.OpSH, tyRX, 0},
	"MH":    {op.OpMH, tyRX, 0},
	"BAS":   {op.OpBAS, tyRX, 0},
	"CVD":   {op.OpCVD, tyRX, 0},
	"CVB":   {op.OpCVB, tyRX, 0},
	"ST":    {op.OpST, tyRX, 0},
	"N":     {op.OpN, tyRX, 0},
	"CL":    {op.OpCL, tyRX, 0},
	"O":     {op.OpO, tyRX, 0},
	"X":     {op.OpX, tyRX, 0},
	"L":     {op.OpL, tyRX, 0},
	"C":     {op.OpC, tyRX, 0},
	"A":     {op.OpA, tyRX, 0},
	"S":     {op.OpS, tyRX, 0},
	"M":     {op.OpM, tyRX, 0},
	"D":     {op.OpD, tyRX, 0},
	"AL":    {op.OpAL, tyRX, 0},
	"SL":    {op.OpSL, tyRX, 0},
	"STD":   {op.OpSTD, tyRX, 0},
	"MXD":   {op.OpMXD, tyRX, 0},
	"LD":    {op.OpLD, tyRX, 0},
	"CD":    {op.OpCD, tyRX, 0},
	"AD":    {op.OpAD, tyRX, 0},
	"SD":    {op.OpSD, tyRX, 0},
	"MD":    {op.OpMD, tyRX, 0},
	"DD":    {op.OpDD, tyRX, 0},
	"AW":    {op.OpAW, tyRX, 0},
	"SW":    {op.OpSW, tyRX, 0},
	"STE":   {op.OpSTE, tyRX, 0},
	"LE":    {op.OpLE, tyRX, 0},
	"CE":    {op.OpCE, tyRX, 0},
	"AE":    {op.OpAE, tyRX, 0},
	"SE":    {op.OpSE, tyRX, 0},
	"ME":    {op.OpME, tyRX, 0},
	"DE":    {op.OpDE, tyRX, 0},
	"AU":    {op.OpAU, tyRX, 0},
	"SU":    {op.OpSU, tyRX, 0},
	"SSM":   {op.OpSSM, tyS, 0},
	"LPSW":  {op.OpLPSW, tyS, 0},
	"DIAG":  {op.OpDIAG, tySI, 0},
	"BXH":   {op.OpBXH, tyRS, 0},
	"BXLE":  {op.OpBXLE, tyRS, 0},
	"SRL":   {op.OpSRL, tyRS, oneOp},
	"SLL":   {op.OpSLL, tyRS, oneOp},
	"SRA":   {op.OpSRA, tyRS, oneOp},
	"SLA":   {op.OpSLA, tyRS, oneOp},
	"SRDL":  {op.OpSRDL, tyRS, oneOp},
	"SLDL":  {op.OpSLDL, tyRS, oneOp},
	"SRDA":  {op.OpSRDA, tyRS, oneOp},
	"SLDA":  {op.OpSLDA, tyRS, oneOp},
	"STM":   {op.OpSTM, tyRS, 0},
	"TM":    {op.OpTM, tySI, 0},
	"MVI":   {op.OpMVI, tySI, 0},
	"TS":    {op.OpTS, tyS, 0},
	"NI":    {op.OpNI, tySI, 0},
	"CLI":   {op.OpCLI, tySI, 0},
	"OI":    {op.OpOI, tySI, 0},
	"XI":    {op.OpXI, tySI, 0},
	"LM":    {op.OpLM, tyRS, 0},
	"SIO":   {op.OpSIO, tyS, 0},
	"TIO":   {op.OpTIO, tyS, 0},
	"HIO":   {op.OpHIO, tyS, 0},
	"TCH":   {op.OpTCH, tyS, 0},
	"LRA":   {op.OpLRA, tyRX, 0},
	"MVN":   {op.OpMVN, tySS, 0},
	"MVC":   {op.OpMVC, tySS, 0},
	"MVZ":   {op.OpMVZ, tySS, 0},
	"NC":    {op.OpNC, tySS, 0},
	"CLC":   {op.OpCLC, tySS, 0},
	"OC":    {op.OpOC, tySS, 0},
	"XC":    {op.OpXC, tySS, 0},
	"TR":    {op.OpTR, tySS, 0},
	"TRT":   {op.OpTRT, tySS, 0},
	"ED":    {op.OpED, tySS, 0},
	"EDMK":  {op.OpEDMK, tySS, 0},
	"MVCIN": {op.OpMVCIN, tySS, 0},
	"MVO":   {op.OpMVO, tySS, twoOp},
	"PACK":  {op.OpPACK, tySS, twoOp},
	"UNPK":  {op.OpUNPK, tySS, twoOp},
	"ZAP":   {op.OpZAP, tySS, twoOp},
	"CP":    {op.OpCP, tySS, twoOp},
	"AP":    {op.OpAP, tySS, twoOp},
	"SP":    {op.OpSP, tySS, twoOp},
	"MP":    {op.OpMP, tySS, twoOp},
	"DP":    {op.OpDP, tySS, twoOp},
	"MVCL":  {op.OpMVCL, tyRR, twoOp},
	"CLCL":  {op.OpCLCL, tyRR, twoOp},
	"STNSM": {op.OpSTNSM, tySI, 0},
	"STOSM": {op.OpSTOSM, tySI, 0},
	"SIGP":  {op.OpSIGP, tyRS, 0},
	"MC":    {op.OpMC, tySI, 0},
	"STCTL": {op.OpSTCTL, tyRS, 0},
	"LCTL":  {op.OpLCTL, tyRS, 0},
	"CS":    {op.OpCS, tyRS, 0},
	"CDS":   {op.OpCDS, tyRS, 0},
	"CLM":   {op.OpCLM, tyRS, 0},
	"STCM":  {op.OpSTCM, tyRS, 0},
	"ICM":   {op.OpICM, tyRS, 0},
	"SRP":   {op.OpSRP, tySS, twoOp},
	"CONCS": {0x00, ty370, 0},
	"DISCS": {0x01, ty370, 0},
	"STIDP": {0x02, ty370, 0},
	"STIDC": {0x03, ty370, 0},
	"SCK":   {0x04, ty370, 0},
	"STCK":  {0x05, ty370, 0},
	"SCKC":  {0x06, ty370, 0},
	"STCKC": {0x07, ty370, 0},
	"SPT":   {0x08, ty370, 0},
	"STPT":  {0x09, ty370, 0},
	"SPKA":  {0x0A, ty370, 0},
	"IPK":   {0x0B, ty370, zeroOp},
	"PTLB":  {0x0D, ty370, zeroOp},
	"SPX":   {0x10, ty370, 0},
	"STPX":  {0x11, ty370, 0},
	"STAP":  {0x12, ty370, 0},
	"RRB":   {0x13, ty370, 0},
}

func Assemble(line string) ([]byte, error) {
	var err string
	var opName string
	var next byte
	var r1, r2, x2, b2, d2 int
	opName, line = getName(line) // Get opcode.
	opc, ok := opMap[strings.ToUpper(opName)]
	if !ok {
		return []byte{}, errors.New("undefined opcode " + opName)
	}
	op := byte(opc.opCode)
	inst := make([]byte, lenMap[opc.opType])
	inst[0] = op
	switch opc.opType {
	case tyRR: // RR syntax "Op r1,r2" "Op r1" "Op imd"
		if opc.opFlags == imdOp {
			line = skipSpace(line)
			if line == "" {
				return []byte{}, errors.New("invalid format for " + opName)
			}
			r1, line = getHex(line, 256)
			if r1 < 0 {
				return []byte{}, errors.New("Invalid immediate value for " + opName)
			}
			inst[1] = byte(r1)
			break
		}
		if opc.opFlags == oneOp {
			line = skipSpace(line)
			if line == "" {
				return []byte{}, errors.New("invalid format for " + opName)
			}
			r1, line = getHex(line, 16)
			if r1 < 0 {
				return []byte{}, errors.New("Invalid immediate value for " + opName)
			}
			inst[1] = byte(r1)
			break
		}
		r1, line = getNumber(line, 16)
		next, line = getNext(line)
		if next != ',' {
			return []byte{}, errors.New("invalid format for " + opName)
		}
		r2, line = getNumber(line, 16)
		if r1 < 0 || r2 < 0 {
			return []byte{}, errors.New("register values out of range " + opName)
		}
		inst[1] = (byte(r1) << 4) | byte(r2)

	case tyRX: // RX syntax is "Op r1,d2(x2,b2)" "Op r1,d2(b2)" "Op r1,d2"
		r1, line = getNumber(line, 16)
		next, line = getNext(line)
		if next != ',' {
			return []byte{}, errors.New("invalid format for " + opName)
		}
		if r1 < 0 {
			return []byte{}, errors.New("register values out of range " + opName)
		}
		b2, d2, x2, line, err = getAddr(line, true, 0)
		if err != "" {
			return []byte{}, errors.New(err + opName)
		}
		inst[1] = (byte(r1) << 4) | byte(x2)
		inst[2] = byte(b2<<4) | byte(d2>>8)
		inst[3] = byte(d2 & 0xff)

	case tyRS: // RS format is "Op r1,r3,d2(b2)" "Op r1,d2(b2)" b2 is optional, d2 is in hex.
		r3 := 0
		r1, line = getNumber(line, 16)
		next, line = getNext(line)
		if next != ',' {
			return []byte{}, errors.New("invalid format for " + opName)
		}
		if opc.opFlags != oneOp {
			r3, line = getNumber(line, 16)
			next, line = getNext(line)
			if next != ',' {
				return []byte{}, errors.New("invalid format for " + opName)
			}
		}
		if r1 < 0 || r3 < 0 {
			return []byte{}, errors.New("register values out of range " + opName)
		}
		b2, d2, _, line, err = getAddr(line, false, 0)
		if err != "" {
			return []byte{}, errors.New(err + opName)
		}

		inst[1] = (byte(r1) << 4) | byte(r3)
		inst[2] = byte(b2<<4) | byte(d2>>8)
		inst[3] = byte(d2 & 0xff)

	case tySI: // SI format is "op d(b),imm" "op d,imm"
		i2 := 0

		b2, d2, _, line, err = getAddr(line, false, 0)
		if err != "" {
			return []byte{}, errors.New(err + opName)
		}

		next, line = getNext(line)
		if next != ',' {
			return []byte{}, errors.New("invalid format for " + opName)
		}
		i2, line = getHex(line, 256)
		if i2 < 0 {
			return []byte{}, errors.New("immediate is out of range for " + opName)
		}
		inst[1] = byte(i2)
		inst[2] = byte(b2<<4) | byte(d2>>8)
		inst[3] = byte(d2 & 0xff)

	case tyS: // S format is "op d(b)" or "op d" or "op"
		if opc.opFlags != zeroOp {
			b2, d2, _, line, err = getAddr(line, false, 0)
			if err != "" {
				return []byte{}, errors.New(err + opName)
			}
		}
		inst[1] = 0
		inst[2] = byte(b2<<4) | byte(d2>>8)
		inst[3] = byte(d2 & 0xff)

	case tySS: // SS format is "op d1(l1,b1),d2(l2,b2)" or "op d1(l,b1),d2(b2)"
		var b1, d1, l1, l2 int
		if opc.opFlags == twoOp {
			b1, d1, l1, line, err = getAddr(line, false, 16)
			l1 <<= 4
		} else {
			b1, d1, l1, line, err = getAddr(line, false, 256)
		}
		if err != "" {
			return []byte{}, errors.New(err + opName)
		}

		next, line = getNext(line)
		if next != ',' {
			return []byte{}, errors.New("invalid format for " + opName)
		}

		if opc.opFlags == twoOp {
			b2, d2, l2, line, err = getAddr(line, false, 16)
		} else {
			b2, d2, _, line, err = getAddr(line, false, 0)
		}
		if err != "" {
			return []byte{}, errors.New(err + opName)
		}

		inst[1] = byte(l1 | l2)
		inst[2] = byte(b1<<4) | byte(d1>>8)
		inst[3] = byte(d1 & 0xff)
		inst[4] = byte(b2<<4) | byte(d2>>8)
		inst[5] = byte(d2 & 0xff)
	case ty370: // Specific 0xB2 370 instruction
		if opc.opFlags != zeroOp {
			b2, d2, _, line, err = getAddr(line, false, 0)
			if err != "" {
				return []byte{}, errors.New(err + opName)
			}
		}
		inst[0] = 0xb2
		inst[1] = op
		inst[2] = byte(b2<<4) | byte(d2>>8)
		inst[3] = byte(d2 & 0xff)
	}
	line = skipSpace(line)
	if line != "" {
		return []byte{}, errors.New("Extra data after instruction " + opName)
	}
	return inst, nil
}

// Skip forward over line until none whitespace character found.
func skipSpace(str string) string {
	for i := range str {
		if !unicode.IsSpace(rune(str[i])) {
			return str[i:]
		}
	}
	return ""
}

// Get next name.
func getName(str string) (string, string) {
	str = skipSpace(str)
	for i := range str {
		if unicode.IsSpace(rune(str[i])) {
			return str[:i], str[i+1:]
		}
	}
	return str, ""
}

// Get decimal number.
// Return -1 if too big not a number.
func getNumber(str string, max int) (int, string) {
	num := 0
	l := 0
	str = skipSpace(str)
	if str == "" {
		return -1, ""
	}
	for _, by := range str {
		if unicode.IsDigit(by) {
			num = (num * 10) + int(by-'0')
			l++
		} else {
			break
		}
	}
	if num >= max {
		num = -1
	}
	return num, str[l:]
}

// Get next non blank character.
func getNext(str string) (byte, string) {
	if str == "" {
		return 0, ""
	}
	for i := range str {
		if !unicode.IsSpace(rune(str[i])) {
			return str[i], str[i+1:]
		}
	}
	return ' ', ""
}

// Get Hex number.
// Return -1 if too big not a number.
func getHex(str string, max int) (int, string) {
	str = skipSpace(str)
	if str == "" {
		return -1, ""
	}
	num := 0
	l := 0
	for _, by := range str {
		if unicode.IsDigit(by) {
			num = (num * 16) + int(by-'0')
			l++
			continue
		}
		if by >= 'a' && by <= 'f' {
			num = (num * 16) + (int(by-'a') + 10)
			l++
			continue
		}
		if by >= 'A' && by <= 'F' {
			num = (num * 16) + (int(by-'A') + 10)
			l++
			continue
		}
		break
	}
	if num >= max {
		num = -1
	}
	return num, str[l:]
}

// Get an Address.
// format: (index)  d(x,b)  d(b) d
// format: (noindex) d(b) d
// format: (immed > 0) d(i,b) d(i)
//
// Return 1: b
//
//	       2: d
//		      3: x or i
//		      4: remainder of line
//	       5: Error
func getAddr(line string, index bool, immed int) (int, int, int, string, string) {
	var next byte
	d := 0
	x := 0
	b := 0
	d, line = getHex(line, 4096)
	if d < 0 {
		return 0, 0, 0, line, "displacment out of range "
	}
	next, line = getNext(line)
	// If next is (, start of index, or length
	if next == '(' {
		if immed != 0 {
			x, line = getNumber(line, immed)
		} else {
			x, line = getNumber(line, 16)
		}

		// Check if next is , or ).
		next, line = getNext(line)
		// Only allowed if index allowed or immediate value.
		if (index || immed > 0) && next == ',' {
			b, line = getNumber(line, 16)
			next, line = getNext(line)
		}

		// If only one index, move it to base.
		if immed == 0 && b == 0 {
			b = x
			x = 0
		}

		// Index must end in ).
		if next != ')' {
			return 0, 0, 0, line, "invalid format for "
		}
	} else if next != 0 {
		// Did not match, put it back.
		line = string(next) + line
	}
	if b < 0 {
		return 0, 0, 0, line, "base register out of range for "
	}
	if x < 0 {
		if immed > 0 {
			return 0, 0, 0, line, "length value out of range "
		}
		return 0, 0, 0, line, "index register out of range for "
	}
	return b, d, x, line, ""
}
