/*
	   IBM 370 Disassembler

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
package disassembler

import (
	"strings"

	op "github.com/rcornwell/S370/emu/opcodemap"
	"github.com/rcornwell/S370/util/hex"
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
	opName  string // Opcode string.
	opType  int    // Opcode type.
	opFlags int    // Opcode flags
}

var opMap = map[int]opcode{
	op.OpSPM:   {"SPM", tyRR, oneOp},
	op.OpBALR:  {"BALR", tyRR, 0},
	op.OpBCTR:  {"BCTR", tyRR, 0},
	op.OpBCR:   {"BCR", tyRR, 0},
	op.OpSSK:   {"SSK", tyRR, 0},
	op.OpISK:   {"ISK", tyRR, 0},
	op.OpSVC:   {"SVC", tyRR, imdOp},
	op.OpBASR:  {"BASR", tyRR, 0},
	op.OpLPR:   {"LPR", tyRR, 0},
	op.OpLNR:   {"LNR", tyRR, 0},
	op.OpLTR:   {"LTR", tyRR, 0},
	op.OpLCR:   {"LCR", tyRR, 0},
	op.OpNR:    {"NR", tyRR, 0},
	op.OpOR:    {"OR", tyRR, 0},
	op.OpXR:    {"XR", tyRR, 0},
	op.OpCLR:   {"CLR", tyRR, 0},
	op.OpCR:    {"CR", tyRR, 0},
	op.OpLR:    {"LR", tyRR, 0},
	op.OpAR:    {"AR", tyRR, 0},
	op.OpSR:    {"SR", tyRR, 0},
	op.OpMR:    {"MR", tyRR, 0},
	op.OpDR:    {"DR", tyRR, 0},
	op.OpALR:   {"ALR", tyRR, 0},
	op.OpSLR:   {"SLR", tyRR, 0},
	op.OpLPDR:  {"LPDR", tyRR, 0},
	op.OpLNDR:  {"LNDR", tyRR, 0},
	op.OpLTDR:  {"LTDR", tyRR, 0},
	op.OpLCDR:  {"LCDR", tyRR, 0},
	op.OpHDR:   {"HDR", tyRR, 0},
	op.OpLRDR:  {"LRDR", tyRR, 0},
	op.OpMXR:   {"MXR", tyRR, 0},
	op.OpMXDR:  {"MXDR", tyRR, 0},
	op.OpLDR:   {"LDR", tyRR, 0},
	op.OpCDR:   {"CDR", tyRR, 0},
	op.OpADR:   {"ADR", tyRR, 0},
	op.OpSDR:   {"SDR", tyRR, 0},
	op.OpMDR:   {"MDR", tyRR, 0},
	op.OpDDR:   {"DDR", tyRR, 0},
	op.OpAWR:   {"AWR", tyRR, 0},
	op.OpSWR:   {"SWR", tyRR, 0},
	op.OpLPER:  {"LPER", tyRR, 0},
	op.OpLNER:  {"LNER", tyRR, 0},
	op.OpLTER:  {"LTER", tyRR, 0},
	op.OpLCER:  {"LCER", tyRR, 0},
	op.OpHER:   {"HER", tyRR, 0},
	op.OpLRER:  {"LRER", tyRR, 0},
	op.OpAXR:   {"AXR", tyRR, 0},
	op.OpSXR:   {"SXR", tyRR, 0},
	op.OpLER:   {"LER", tyRR, 0},
	op.OpCER:   {"CER", tyRR, 0},
	op.OpAER:   {"AER", tyRR, 0},
	op.OpSER:   {"SER", tyRR, 0},
	op.OpMER:   {"MER", tyRR, 0},
	op.OpDER:   {"DER", tyRR, 0},
	op.OpAUR:   {"AUR", tyRR, 0},
	op.OpSUR:   {"SUR", tyRR, 0},
	op.OpSTH:   {"STH", tyRX, 0},
	op.OpLA:    {"LA", tyRX, 0},
	op.OpSTC:   {"STC", tyRX, 0},
	op.OpIC:    {"IC", tyRX, 0},
	op.OpEX:    {"EX", tyRX, 0},
	op.OpBAL:   {"BAL", tyRX, 0},
	op.OpBCT:   {"BCT", tyRX, 0},
	op.OpBC:    {"BC", tyRX, 0},
	op.OpLH:    {"LH", tyRX, 0},
	op.OpCH:    {"CH", tyRX, 0},
	op.OpAH:    {"AH", tyRX, 0},
	op.OpSH:    {"SH", tyRX, 0},
	op.OpMH:    {"MH", tyRX, 0},
	op.OpBAS:   {"BAS", tyRX, 0},
	op.OpCVD:   {"CVD", tyRX, 0},
	op.OpCVB:   {"CVB", tyRX, 0},
	op.OpST:    {"ST", tyRX, 0},
	op.OpN:     {"N", tyRX, 0},
	op.OpCL:    {"CL", tyRX, 0},
	op.OpO:     {"O", tyRX, 0},
	op.OpX:     {"X", tyRX, 0},
	op.OpL:     {"L", tyRX, 0},
	op.OpC:     {"C", tyRX, 0},
	op.OpA:     {"A", tyRX, 0},
	op.OpS:     {"S", tyRX, 0},
	op.OpM:     {"M", tyRX, 0},
	op.OpD:     {"D", tyRX, 0},
	op.OpAL:    {"AL", tyRX, 0},
	op.OpSL:    {"SL", tyRX, 0},
	op.OpSTD:   {"STD", tyRX, 0},
	op.OpMXD:   {"MXD", tyRX, 0},
	op.OpLD:    {"LD", tyRX, 0},
	op.OpCD:    {"CD", tyRX, 0},
	op.OpAD:    {"AD", tyRX, 0},
	op.OpSD:    {"SD", tyRX, 0},
	op.OpMD:    {"MD", tyRX, 0},
	op.OpDD:    {"DD", tyRX, 0},
	op.OpAW:    {"AW", tyRX, 0},
	op.OpSW:    {"SW", tyRX, 0},
	op.OpSTE:   {"STE", tyRX, 0},
	op.OpLE:    {"LE", tyRX, 0},
	op.OpCE:    {"CE", tyRX, 0},
	op.OpAE:    {"AE", tyRX, 0},
	op.OpSE:    {"SE", tyRX, 0},
	op.OpME:    {"ME", tyRX, 0},
	op.OpDE:    {"DE", tyRX, 0},
	op.OpAU:    {"AU", tyRX, 0},
	op.OpSU:    {"SU", tyRX, 0},
	op.OpSSM:   {"SSM", tyS, 0},
	op.OpLPSW:  {"LPSW", tyS, 0},
	op.OpDIAG:  {"DIAG", tySI, 0},
	op.OpBXH:   {"BXH", tyRS, 0},
	op.OpBXLE:  {"BXLE", tyRS, 0},
	op.OpSRL:   {"SRL", tyRS, oneOp},
	op.OpSLL:   {"SLL", tyRS, oneOp},
	op.OpSRA:   {"SRA", tyRS, oneOp},
	op.OpSLA:   {"SLA", tyRS, oneOp},
	op.OpSRDL:  {"SRDL", tyRS, oneOp},
	op.OpSLDL:  {"SLDL", tyRS, oneOp},
	op.OpSRDA:  {"SRDA", tyRS, oneOp},
	op.OpSLDA:  {"SLDA", tyRS, oneOp},
	op.OpSTM:   {"STM", tyRS, 0},
	op.OpTM:    {"TM", tySI, 0},
	op.OpMVI:   {"MVI", tySI, 0},
	op.OpTS:    {"TS", tyS, 0},
	op.OpNI:    {"NI", tySI, 0},
	op.OpCLI:   {"CLI", tySI, 0},
	op.OpOI:    {"OI", tySI, 0},
	op.OpXI:    {"XI", tySI, 0},
	op.OpLM:    {"LM", tyRS, 0},
	op.OpSIO:   {"SIO", tyS, 0},
	op.OpTIO:   {"TIO", tyS, 0},
	op.OpHIO:   {"HIO", tyS, 0},
	op.OpTCH:   {"TCH", tyS, 0},
	op.OpLRA:   {"LRA", tyRX, 0},
	op.OpMVN:   {"MVN", tySS, 0},
	op.OpMVC:   {"MVC", tySS, 0},
	op.OpMVZ:   {"MVZ", tySS, 0},
	op.OpNC:    {"NC", tySS, 0},
	op.OpCLC:   {"CLC", tySS, 0},
	op.OpOC:    {"OC", tySS, 0},
	op.OpXC:    {"XC", tySS, 0},
	op.OpTR:    {"TR", tySS, 0},
	op.OpTRT:   {"TRT", tySS, 0},
	op.OpED:    {"ED", tySS, 0},
	op.OpEDMK:  {"EDMK", tySS, 0},
	op.OpMVCIN: {"MVCIN", tySS, 0},
	op.OpMVO:   {"MVO", tySS, twoOp},
	op.OpPACK:  {"PACK", tySS, twoOp},
	op.OpUNPK:  {"UNPK", tySS, twoOp},
	op.OpZAP:   {"ZAP", tySS, twoOp},
	op.OpCP:    {"CP", tySS, twoOp},
	op.OpAP:    {"AP", tySS, twoOp},
	op.OpSP:    {"SP", tySS, twoOp},
	op.OpMP:    {"MP", tySS, twoOp},
	op.OpDP:    {"DP", tySS, twoOp},
	op.OpMVCL:  {"MVCL", tyRR, twoOp},
	op.OpCLCL:  {"CLCL", tyRR, twoOp},
	op.OpSTNSM: {"STNSM", tySI, 0},
	op.OpSTOSM: {"STOSM", tySI, 0},
	op.OpSIGP:  {"SIGP", tyRS, 0},
	op.OpMC:    {"MC", tySI, 0},
	op.Op370:   {"", ty370, 0},
	op.OpSTCTL: {"STCTL", tyRS, 0},
	op.OpLCTL:  {"LCTL", tyRS, 0},
	op.OpCS:    {"CS", tyRS, 0},
	op.OpCDS:   {"CDS", tyRS, 0},
	op.OpCLM:   {"CLM", tyRS, 0},
	op.OpSTCM:  {"STCM", tyRS, 0},
	op.OpICM:   {"ICM", tyRS, 0},
	op.OpSRP:   {"SRP", tySS, twoOp},
}

var op370 = map[int]opcode{
	0x00: {"CONCS", tyS, 0},
	0x01: {"DISCS", tyS, 0},
	0x02: {"STIDP", tyS, 0},
	0x03: {"STIDC", tyS, 0},
	0x04: {"SCK", tyS, 0},
	0x05: {"STCK", tyS, 0},
	0x06: {"SCKC", tyS, 0},
	0x07: {"STCKC", tyS, 0},
	0x08: {"SPT", tyS, 0},
	0x09: {"STPT", tyS, 0},
	0x0A: {"SPKA", tyS, 0},
	0x0B: {"IPK", tyS, zeroOp},
	0x0D: {"PTLB", tyS, zeroOp},
	0x10: {"SPX", tyS, 0},
	0x11: {"STPX", tyS, 0},
	0x12: {"STAP", tyS, 0},
	0x13: {"RRB", tyS, 0},
}

func PrintInst(data []byte) (string, int) {
	var str strings.Builder
	str.Grow(80)
	length := hexDump(&str, data)
	l := str.Len()
	b := "                        "
	str.WriteString(b[0 : 22-l])

	_ = doInst(&str, data, true)
	return str.String(), length
}

func Disassemble(data []byte) (string, int) {
	var str strings.Builder
	str.Grow(80)
	length := doInst(&str, data, false)
	return str.String(), length
}

func doInst(str *strings.Builder, data []byte, nodump bool) int {
	// Find opcode
	opc := int(data[0])
	op, ok := opMap[opc]
	if !ok {
		if nodump {
			return 0
		}
		return hexDump(str, data)
	}
	if op.opType == ty370 {
		opsub := int(data[1])
		op = op370[opsub]
	}
	// Make opcode align
	inst := op.opName + "       "
	str.WriteString(inst[:6])
	length := 2
	switch op.opType {
	case tyRR:
		switch op.opFlags {
		case imdOp:
			hex.FormatByte(str, data[1])

		case oneOp:
			hex.FormatDigit(str, (data[1]>>4)&0xf)

		default:
			hex.FormatDecimal(str, (data[1]>>4)&0xf)
			str.WriteByte(',')
			hex.FormatDecimal(str, data[1]&0xf)
		}
	case tyRX:
		length += 2
		hex.FormatDecimal(str, (data[1]>>4)&0xf)
		str.WriteByte(',')
		address(str, data[1]&0xf, data[2:4])

	case tyRS:
		length += 2
		hex.FormatDecimal(str, (data[1]>>4)&0xf)
		str.WriteByte(',')
		if op.opFlags != oneOp {
			hex.FormatDecimal(str, data[1]&0xf)
			str.WriteByte(',')
		}
		address(str, 0, data[2:4])

	case tySI:
		length += 2
		address(str, 0, data[2:4])
		str.WriteByte(',')
		hex.FormatByte(str, data[1])

	case tyS:
		length += 2
		if op.opFlags != zeroOp {
			address(str, 0, data[2:4])
		}
	case tySS:
		length += 4
		hex.FormatDisp(str, data[2:4])
		b2 := (data[2] >> 4) & 0xf
		str.WriteByte('(')
		if op.opFlags == twoOp {
			hex.FormatDecimal(str, (data[1]>>4)&0xf)
		} else {
			hex.FormatDecimal(str, data[1])
		}

		if b2 != 0 {
			str.WriteByte(',')
			hex.FormatDecimal(str, b2)
		}
		str.WriteByte(')')
		str.WriteByte(',')
		hex.FormatDisp(str, data[4:6])
		b2 = (data[4] >> 4) & 0xf
		if op.opFlags == twoOp {
			str.WriteByte('(')
			hex.FormatDecimal(str, data[1]&0xf)
			if b2 != 0 {
				str.WriteByte(',')
				hex.FormatDecimal(str, b2)
			}
			str.WriteByte(')')
		} else if b2 != 0 {
			str.WriteByte('(')
			hex.FormatDecimal(str, b2)
			str.WriteByte(')')
		}
	}
	return length
}

func address(str *strings.Builder, x2 byte, data []byte) {
	hex.FormatDisp(str, data)
	b2 := (data[0] >> 4) & 0xf
	if x2 != 0 || b2 != 0 {
		str.WriteByte('(')
		if x2 != 0 {
			hex.FormatDecimal(str, x2)
			str.WriteByte(',')
		}
		hex.FormatDecimal(str, b2)
		str.WriteByte(')')
	}
}

func hexDump(str *strings.Builder, data []byte) int {
	switch data[0] & 0xc0 {
	default:
		fallthrough
	case 0: // RR
		hex.FormatBytes(str, true, data[0:2])
		return 2

	case 0x40, 0x80: // RX
		hex.FormatByte(str, data[0])
		str.WriteByte(' ')
		hex.FormatDigit(str, (data[1]>>4)&0xf)
		str.WriteByte(' ')
		hex.FormatAddr(str, data[2:4])
		return 4

	case 0xC0: // SS
		hex.FormatBytes(str, true, data[0:2])
		str.WriteByte(' ')
		hex.FormatAddr(str, data[2:4])
		str.WriteByte(' ')
		hex.FormatAddr(str, data[4:6])
		return 6
	}
}
