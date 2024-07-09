/* CPU definitions for IBM 370 simulator definitions

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

package cpu

type stepInfo struct {
	opcode   uint8  // Current opcode
	reg      uint8  // R1, R2 Registers
	R1       uint8  // R1
	R2       uint8  // R2
	address1 uint32 // Current instruction first address
	address2 uint32 // Current instruction second address
	src1     uint32 // Source value for first operand
	src2     uint32 // Source value for second operand
	fsrc1    uint64 // Floating point first operand
	fsrc2    uint64 // Floating point second operand
}

type cpu struct {
	PC       uint32      // Program counter
	iPC      uint32      // Initial PC for instruction
	regs     [16]uint32  // Internal registers
	fpregs   [8]uint64   // Floating point registers
	cregs    [16]uint32  // Control registers /67 or 370 only
	sysMask  uint16      // Channel interrupt enable
	stKey    uint8       // Current storage key
	ecMode   bool        // Current PSW is EC Mode
	cc       uint8       // Current CC code
	ilc      uint8       // Current instruction length
	progMask uint8       // Program mask
	flags    uint8       // System flags
	pageEnb  bool        // Paging enabled
	tlb      [256]uint32 // Translation Lookaside Buffer

	//  uint8        ext_en;                    // Enable external and timer IRQ's
	//  uint8        irq_en;                    // Enable channel IRQ's
	//  uint8        tod_en;                    // Enable TOD compare irq's
	//  uint8        intval_en;                 // Enable interval irq's

	//  uint16       irqcode;                   // Interrupt code

	pageShift uint32 // Amount to shift for page
	pageMask  uint32 // Mask of bits in page address
	pageIndex uint32 // PTE index mask
	segShift  uint32 // Amount to shift for segment
	segMask   uint32 // Mask bits for segment
	segLen    uint32 // Length of segment table

	segAddr     uint32    // Address of segment table
	pteLenShift uint32    // Shift to Check if out of page table
	pteAvail    uint32    // Mask of available bit in PTE
	pteMBZ      uint32    // Bits that must be zero in PTE
	pteShift    uint32    // Bits to shift a PTE entry
	perEnb      bool      // Enable PER tracing
	perRegMod   uint32    // Module modification mask
	perCode     uint16    // Code for PER
	perAddr     uint32    // Address of last reference
	perBranch   bool      // Trap on successful branch
	perFetch    bool      // Trap Fetch of instructions
	perStore    bool      // Trap on storage modify
	perReg      bool      // Trap on register modify
	irqEnb      bool      // Interrupts enabled
	extEnb      bool      // External interrupts enabled
	extIrq      bool      // External interrupt pending
	intIrq      bool      // Interval timer interrupt
	intEnb      bool      // Interval timer enable
	todClock    [2]uint32 // Current Time of Day Clock
	todSet      bool      // TOD set to correct time

	//	clk_en        bool      // Clock interrupt enable
	todEnb    bool      // TOD enable
	todIrq    bool      // TOD compare IRQ
	clkCmp    [2]uint32 // Clock compare value
	clkIrq    bool      // Clock compare IRQ
	cpuTimer  [2]uint32 // CPU timer value
	timerTics int       // Interval Timer is ever 3 tics
	vmAssist  bool      // VM Assist functions enabled.
	vmaEnb    bool      // VM Assist enabled.
	table     [256]func(*stepInfo) uint16
}

const (
	// PSW enable bits in SSM.
	extEnable uint8 = 0x01
	irqEnable uint8 = 0x02
	datEnable uint8 = 0x04
	perEnable uint8 = 0x40

	// Program mask bits.
	ecMode  uint8 = 0x08 // PSW is in EC mode
	mCheck  uint8 = 0x04 // Machine check flag
	wait    uint8 = 0x02 // Wait state
	problem uint8 = 0x01 // Problem state

	// exception flags.
	FIXOVER  uint8 = 0x08 // Fixed point overflow
	DECOVER  uint8 = 0x04 // Decimal overflow
	EXPUNDER uint8 = 0x02 // Exponent overflow.
	SIGMASK  uint8 = 0x01 // Significance

	// low addresses.
	iPSW     uint32 = 0x00 // IPSW
	iccCCW1  uint32 = 0x08 // ICCW1
	iccCCW2  uint32 = 0x10 // ICCW2
	oEPSW    uint32 = 0x18 // External old PSW
	oSPSW    uint32 = 0x20 // Supervisor call old PSW
	oPPSW    uint32 = 0x28 // Program old PSW
	oMPSW    uint32 = 0x30 // Machine check PSW
	oIOPSW   uint32 = 0x38 // IO old PSW
	CSW      uint32 = 0x40 // CSW
	CAW      uint32 = 0x48 // CAW
	timer    uint32 = 0x50 // timer
	nEPSW    uint32 = 0x58 // External new PSW
	nSPSW    uint32 = 0x60 // SVC new PSW
	nPPSW    uint32 = 0x68 // Program new PSW
	nMPSW    uint32 = 0x70 // Machine Check PSW
	nIOPSW   uint32 = 0x78 // IOPSW
	diagArea uint32 = 0x80 // Diag scan area.

	// Operator trap values.
	ircOper     uint16 = 0x0001 // Operations exception
	ircPriv     uint16 = 0x0002 // Privlege violation
	ircExec     uint16 = 0x0003 // Execution
	ircProt     uint16 = 0x0004 // Protection violation
	ircAddr     uint16 = 0x0005 // Address error
	ircSpec     uint16 = 0x0006 // Specification error
	ircData     uint16 = 0x0007 // Data exception
	ircFixOver  uint16 = 0x0008 // Fixed point overflow
	ircFixDiv   uint16 = 0x0009 // Fixed point divide
	ircDecOver  uint16 = 0x000a // Decimal overflow
	ircDecDiv   uint16 = 0x000b // Decimal divide
	ircExpOver  uint16 = 0x000c // Exponent overflow
	ircExpUnder uint16 = 0x000d // Exponent underflow
	ircSignif   uint16 = 0x000e // Significance error
	ircFPDiv    uint16 = 0x000f // Floating pointer divide
	ircSeg      uint16 = 0x0010 // Segment translation
	ircPage     uint16 = 0x0011 // Page translation
	ircTrans    uint16 = 0x0012 // Translation special
	ircSpecOp   uint16 = 0x0013 // Special operation
	ircMCE      uint16 = 0x0040 // Monitor event
	ircPer      uint16 = 0x0080 // Per event

	// DAT masks definitions.
	pteLength uint32 = 0xff000000 // Page table length
	pteAddr   uint32 = 0x00fffffe // Address of table
	pteValid  uint32 = 0x00000001 // table valid
	tlbSeg    uint32 = 0x0001f000 // Segment address
	tlbValid  uint32 = 0x80000000 // Entry valid
	tlbPhy    uint32 = 0x00000fff // Physical page
	segMask   uint32 = 0xfffff000 // Mask segment

	// Mask constants.
	AMASK  uint32 = 0x00ffffff // Mask address bits
	LMASK  uint32 = 0x0000ffff // Lower Half word maske
	SPMASK uint32 = 0x00fff800 // Mask off storage boundary
	WMASK  uint32 = 0x00fffffc // Mask address to word boundary
	MSIGN  uint32 = 0x80000000 // Minus sign
	MMASK  uint32 = 0x00ffffff // Mantissa mask
	EMASK  uint32 = 0x7f000000 // Exponent mask
	XMASK  uint32 = 0x0fffffff // Working FP mask
	FMASK  uint32 = 0xffffffff // Full Word mask
	CMASK  uint32 = 0x10000000 // Carry mask
	NMASK  uint32 = 0x00f00000 // Normalize mask
	SNMASK uint32 = 0x0f000000 // Short normal mask
	PMASK  uint32 = 0xf0000000 // Storage protection mask
	HMASK  uint32 = 0xffff0000 // Mask upper half word

	// Long masks.
	HMASKL  uint64 = 0xffffffff00000000 // Upper word
	LMASKL  uint64 = 0x00000000ffffffff // Lower word
	MSIGNL  uint64 = 0x8000000000000000 // Sign up long floating point
	MMASKL  uint64 = 0x00ffffffffffffff // Mantissa of long floating point
	CMASKL  uint64 = 0x1000000000000000 // Carry from long floating point with guard
	EMASKL  uint64 = 0x7f00000000000000 // Exponent mask of long floating point
	XMASKL  uint64 = 0x0fffffffffffffff // Mask after adding two long floating point numbers
	NMASKL  uint64 = 0x00f0000000000000 // Mask to check if number needs to be normalized
	UMASKL  uint64 = 0x0ffffffffffffff0 // Mask to make sure guard digit is zero
	SNMASKL uint64 = 0x0f00000000000000 // Normalize mask for numbers with guard digit
	OMASKL  uint64 = 0xffffffff80000000 // Mask of upper half plus sign used in CVB
	RMASKL  uint64 = 0x0000000080000000 // Long rounding bit
)

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
	OpSTMC  = 0xB0 // 360/67 Store control
	OpLRA   = 0xB1
	Op370   = 0xB2 // Misc 370 system instructions
	OpSTCTL = 0xB6 // 370 Store control
	OpLCTL  = 0xB7 // 370 Load control
	OpLMC   = 0xB8 // 360/67 Load Control
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
