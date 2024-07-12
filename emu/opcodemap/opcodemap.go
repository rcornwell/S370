/*
   CPU opcodes for assembly and disassembly

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
   ROBERT M SUPNIK BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
   IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

*/

package opcodemap

const (
	// Opcode definitions.
	OpSPM   = 0x04 // src1 = R1, src2 = R2
	OpBALR  = 0x05 // src1 = R1, src2 = R2
	OpBCTR  = 0x06 // src1 = R1, src2 = R2
	OpBCR   = 0x07 // src1 = R1, src2 = R2
	OpSSK   = 0x08 // src1 = R1, src2 = R2
	OpISK   = 0x09 // src1 = R1, src2 = R2
	OpSVC   = 0x0A // src1 = R1, src2 = R2
	OpBASR  = 0x0D // src1 = R1, src2 = R2
	OpMVCL  = 0x0E // 370 Move long
	OpCLCL  = 0x0F // 370 Compare logical long
	OpLPR   = 0x10 // src1 = R1, src2 = R2
	OpLNR   = 0x11 // src1 = R1, src2 = R2
	OpLTR   = 0x12 // src1 = R1, src2 = R2
	OpLCR   = 0x13 // src1 = R1, src2 = R2
	OpNR    = 0x14 // src1 = R1, src2 = R2
	OpCLR   = 0x15 // src1 = R1, src2 = R2
	OpOR    = 0x16 // src1 = R1, src2 = R2
	OpXR    = 0x17 // src1 = R1, src2 = R2
	OpCR    = 0x19 // src1 = R1, src2 = R2
	OpLR    = 0x18 // src1 = R1, src2 = R2
	OpAR    = 0x1A // src1 = R1, src2 = R2
	OpSR    = 0x1B // src1 = R1, src2 = R2
	OpMR    = 0x1C // src1 = R1, src2 = R2
	OpDR    = 0x1D // src1 = R1, src2 = R2
	OpALR   = 0x1E // src1 = R1, src2 = R2
	OpSLR   = 0x1F // src1 = R1, src2 = R2
	OpLPDR  = 0x20
	OpLNDR  = 0x21
	OpLTDR  = 0x22
	OpLCDR  = 0x23
	OpHDR   = 0x24
	OpLRDR  = 0x25
	OpMXR   = 0x26
	OpMXDR  = 0x27
	OpLDR   = 0x28
	OpCDR   = 0x29
	OpADR   = 0x2A
	OpSDR   = 0x2B
	OpMDR   = 0x2C
	OpDDR   = 0x2D
	OpAWR   = 0x2E
	OpSWR   = 0x2F
	OpLPER  = 0x30
	OpLNER  = 0x31
	OpLTER  = 0x32
	OpLCER  = 0x33
	OpHER   = 0x34
	OpLRER  = 0x35
	OpAXR   = 0x36
	OpSXR   = 0x37
	OpLER   = 0x38
	OpCER   = 0x39
	OpAER   = 0x3A
	OpSER   = 0x3B
	OpMER   = 0x3C
	OpDER   = 0x3D
	OpAUR   = 0x3E
	OpSUR   = 0x3F
	OpSTH   = 0x40 // src1 = R1, src2= A1
	OpLA    = 0x41 // src1 = R1, src2= A1
	OpSTC   = 0x42 // src1 = R1, src2= A1
	OpIC    = 0x43 // src1 = R1, src2= A1
	OpEX    = 0x44 // src1 = R1, src2= A1
	OpBAL   = 0x45 // src1 = R1, src2= A1
	OpBCT   = 0x46 // src1 = R1, src2= A1
	OpBC    = 0x47 // src1 = R1, src2= A1
	OpLH    = 0x48 // src1 = R1, src2= MH
	OpCH    = 0x49 // src1 = R1, src2= MH
	OpAH    = 0x4A // src1 = R1, src2= MH
	OpSH    = 0x4B // src1 = R1, src2= MH
	OpMH    = 0x4C // src1 = R1, src2= MH
	OpBAS   = 0x4D // src1 = R1, src2= A1
	OpCVD   = 0x4E // src1 = R1, src2= A1
	OpCVB   = 0x4F // src1 = R1, src2= A1
	OpST    = 0x50 // src1 = R1, src2= A1
	OpN     = 0x54 // src1 = R1, src2= M
	OpCL    = 0x55 // src1 = R1, src2= M
	OpO     = 0x56 // src1 = R1, src2= M
	OpX     = 0x57 // src1 = R1, src2= M
	OpL     = 0x58 // src1 = R1, src2= M
	OpC     = 0x59 // src1 = R1, src2= M
	OpA     = 0x5A // src1 = R1, src2= M
	OpS     = 0x5B // src1 = R1, src2= M
	OpM     = 0x5C // src1 = R1, src2= M
	OpD     = 0x5D // src1 = R1, src2= M
	OpAL    = 0x5E // src1 = R1, src2= M
	OpSL    = 0x5F // src1 = R1, src2= M
	OpSTD   = 0x60
	OpMXD   = 0x67
	OpLD    = 0x68
	OpCD    = 0x69
	OpAD    = 0x6A
	OpSD    = 0x6B
	OpMD    = 0x6C
	OpDD    = 0x6D
	OpAW    = 0x6E
	OpSW    = 0x6F
	OpSTE   = 0x70
	OpLE    = 0x78
	OpCE    = 0x79
	OpAE    = 0x7A
	OpSE    = 0x7B
	OpME    = 0x7C
	OpDE    = 0x7D
	OpAU    = 0x7E
	OpSU    = 0x7F
	OpSSM   = 0x80
	OpLPSW  = 0x82
	OpDIAG  = 0x83
	OpBXH   = 0x86
	OpBXLE  = 0x87
	OpSRL   = 0x88
	OpSLL   = 0x89
	OpSRA   = 0x8A
	OpSLA   = 0x8B
	OpSRDL  = 0x8C
	OpSLDL  = 0x8D
	OpSRDA  = 0x8E
	OpSLDA  = 0x8F
	OpSTM   = 0x90
	OpTM    = 0x91
	OpMVI   = 0x92
	OpTS    = 0x93
	OpNI    = 0x94
	OpCLI   = 0x95
	OpOI    = 0x96
	OpXI    = 0x97
	OpLM    = 0x98
	OpSIO   = 0x9C
	OpTIO   = 0x9D
	OpHIO   = 0x9E
	OpTCH   = 0x9F
	OpSTNSM = 0xAC // 370 Store then and system mask
	OpSTOSM = 0xAD // 370 Store then or system mask
	OpSIGP  = 0xAE // 370 Signal processor
	OpMC    = 0xAF // 370 Monitor call
	OpLRA   = 0xB1
	Op370   = 0xB2 // Misc 370 system instructions
	OpSTCTL = 0xB6 // 370 Store control
	OpLCTL  = 0xB7 // 370 Load control
	OpCS    = 0xBA // 370 Compare and swap
	OpCDS   = 0xBB // 370 Compare double and swap
	OpCLM   = 0xBD // 370 Compare character under mask
	OpSTCM  = 0xBE // 370 Store character under mask
	OpICM   = 0xBF // 370 Insert character under mask
	OpMVN   = 0xD1
	OpMVC   = 0xD2
	OpMVZ   = 0xD3
	OpNC    = 0xD4
	OpCLC   = 0xD5
	OpOC    = 0xD6
	OpXC    = 0xD7
	OpTR    = 0xDC
	OpTRT   = 0xDD
	OpED    = 0xDE
	OpEDMK  = 0xDF
	OpMVCIN = 0xE8 // 370 Move inverse
	OpSRP   = 0xF0 // 370 Shift and round decimal
	OpMVO   = 0xF1
	OpPACK  = 0xF2
	OpUNPK  = 0xF3
	OpZAP   = 0xF8
	OpCP    = 0xF9
	OpAP    = 0xFA
	OpSP    = 0xFB
	OpMP    = 0xFC
	OpDP    = 0xFD
)
