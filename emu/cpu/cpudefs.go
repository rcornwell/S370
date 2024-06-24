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

	//  uint16       irqcode;                   // Interupt code

	pageShift uint32 // Amount to shift for page
	pageMask  uint32 // Mask of bits in page address
	pageIndex uint32 // PTE index mask
	segShift  uint32 // Amount to shift for segment
	segMask   uint32 // Mask bits for segment
	segLen    uint32 // Length of segment table

	segAddr     uint32    // Address of segment table
	pteLenShift uint32    // Shift to Check if out out page table
	pteAvail    uint32    // Mask of available bit in PTE
	pteMBZ      uint32    // Bits that must be zero in PTE
	pteShift    uint32    // Bits to shift a PTE entry
	perEnb      bool      // Enable PER tracing
	perRegMod   uint32    // Module modification mask
	perCode     uint16    // Code for PER
	perAddr     uint32    // Address of last reference
	perBranch   bool      // Trap on sucessful branch
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
	vmEnb     bool      // VM Assist enabled.
	table     [256]func(*stepInfo) uint16
}

const (
	// PSW enable bits in SSM
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

	// low addresses
	iPSW     uint32 = 0x00 // IPSW
	iccCCW1  uint32 = 0x08 // ICCW1
	iccCCW2  uint32 = 0x10 // ICCW2
	oEPSW    uint32 = 0x18 // External old PSW
	oSPSW    uint32 = 0x20 // Supervisior call old PSW
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

	// DAT masks definitions
	pteLength uint32 = 0xff000000 // Page table length
	pteAddr   uint32 = 0x00fffffe // Address of table
	pteValid  uint32 = 0x00000001 // table valid
	tlbSeg    uint32 = 0x0001f000 // Segment address
	tlbValid  uint32 = 0x80000000 // Entry valid
	tlbPhy    uint32 = 0x00000fff // Physical page
	segMask   uint32 = 0xfffff000 // Mask segment

	// Mask constants
	AMASK  uint32 = 0x00ffffff // Mask address bits
	LMASK  uint32 = 0x0000ffff // Lower Half word maske
	SPMASK uint32 = 0x00fff800 // Mask off storage boundry
	WMASK  uint32 = 0x00fffffc // Mask address to word boundry
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

	// Long masks
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
	// Opcode definitions
	OP_SPM   = 0x04 // src1 = R1, src2 = R2
	OP_BALR  = 0x05 // src1 = R1, src2 = R2
	OP_BCTR  = 0x06 // src1 = R1, src2 = R2
	OP_BCR   = 0x07 // src1 = R1, src2 = R2
	OP_SSK   = 0x08 // src1 = R1, src2 = R2
	OP_ISK   = 0x09 // src1 = R1, src2 = R2
	OP_SVC   = 0x0A // src1 = R1, src2 = R2
	OP_BASR  = 0x0D // src1 = R1, src2 = R2
	OP_MVCL  = 0x0E // 370 Move long
	OP_CLCL  = 0x0F // 370 Compare logical long
	OP_LPR   = 0x10 // src1 = R1, src2 = R2
	OP_LNR   = 0x11 // src1 = R1, src2 = R2
	OP_LTR   = 0x12 // src1 = R1, src2 = R2
	OP_LCR   = 0x13 // src1 = R1, src2 = R2
	OP_NR    = 0x14 // src1 = R1, src2 = R2
	OP_CLR   = 0x15 // src1 = R1, src2 = R2
	OP_OR    = 0x16 // src1 = R1, src2 = R2
	OP_XR    = 0x17 // src1 = R1, src2 = R2
	OP_CR    = 0x19 // src1 = R1, src2 = R2
	OP_LR    = 0x18 // src1 = R1, src2 = R2
	OP_AR    = 0x1A // src1 = R1, src2 = R2
	OP_SR    = 0x1B // src1 = R1, src2 = R2
	OP_MR    = 0x1C // src1 = R1, src2 = R2
	OP_DR    = 0x1D // src1 = R1, src2 = R2
	OP_ALR   = 0x1E // src1 = R1, src2 = R2
	OP_SLR   = 0x1F // src1 = R1, src2 = R2
	OP_LPDR  = 0x20
	OP_LNDR  = 0x21
	OP_LTDR  = 0x22
	OP_LCDR  = 0x23
	OP_HDR   = 0x24
	OP_LRDR  = 0x25
	OP_MXR   = 0x26
	OP_MXDR  = 0x27
	OP_LDR   = 0x28
	OP_CDR   = 0x29
	OP_ADR   = 0x2A
	OP_SDR   = 0x2B
	OP_MDR   = 0x2C
	OP_DDR   = 0x2D
	OP_AWR   = 0x2E
	OP_SWR   = 0x2F
	OP_LPER  = 0x30
	OP_LNER  = 0x31
	OP_LTER  = 0x32
	OP_LCER  = 0x33
	OP_HER   = 0x34
	OP_LRER  = 0x35
	OP_AXR   = 0x36
	OP_SXR   = 0x37
	OP_LER   = 0x38
	OP_CER   = 0x39
	OP_AER   = 0x3A
	OP_SER   = 0x3B
	OP_MER   = 0x3C
	OP_DER   = 0x3D
	OP_AUR   = 0x3E
	OP_SUR   = 0x3F
	OP_STH   = 0x40 // src1 = R1, src2= A1
	OP_LA    = 0x41 // src1 = R1, src2= A1
	OP_STC   = 0x42 // src1 = R1, src2= A1
	OP_IC    = 0x43 // src1 = R1, src2= A1
	OP_EX    = 0x44 // src1 = R1, src2= A1
	OP_BAL   = 0x45 // src1 = R1, src2= A1
	OP_BCT   = 0x46 // src1 = R1, src2= A1
	OP_BC    = 0x47 // src1 = R1, src2= A1
	OP_LH    = 0x48 // src1 = R1, src2= MH
	OP_CH    = 0x49 // src1 = R1, src2= MH
	OP_AH    = 0x4A // src1 = R1, src2= MH
	OP_SH    = 0x4B // src1 = R1, src2= MH
	OP_MH    = 0x4C // src1 = R1, src2= MH
	OP_BAS   = 0x4D // src1 = R1, src2= A1
	OP_CVD   = 0x4E // src1 = R1, src2= A1
	OP_CVB   = 0x4F // src1 = R1, src2= A1
	OP_ST    = 0x50 // src1 = R1, src2= A1
	OP_N     = 0x54 // src1 = R1, src2= M
	OP_CL    = 0x55 // src1 = R1, src2= M
	OP_O     = 0x56 // src1 = R1, src2= M
	OP_X     = 0x57 // src1 = R1, src2= M
	OP_L     = 0x58 // src1 = R1, src2= M
	OP_C     = 0x59 // src1 = R1, src2= M
	OP_A     = 0x5A // src1 = R1, src2= M
	OP_S     = 0x5B // src1 = R1, src2= M
	OP_M     = 0x5C // src1 = R1, src2= M
	OP_D     = 0x5D // src1 = R1, src2= M
	OP_AL    = 0x5E // src1 = R1, src2= M
	OP_SL    = 0x5F // src1 = R1, src2= M
	OP_STD   = 0x60
	OP_MXD   = 0x67
	OP_LD    = 0x68
	OP_CD    = 0x69
	OP_AD    = 0x6A
	OP_SD    = 0x6B
	OP_MD    = 0x6C
	OP_DD    = 0x6D
	OP_AW    = 0x6E
	OP_SW    = 0x6F
	OP_STE   = 0x70
	OP_LE    = 0x78
	OP_CE    = 0x79
	OP_AE    = 0x7A
	OP_SE    = 0x7B
	OP_ME    = 0x7C
	OP_DE    = 0x7D
	OP_AU    = 0x7E
	OP_SU    = 0x7F
	OP_SSM   = 0x80
	OP_LPSW  = 0x82
	OP_DIAG  = 0x83
	OP_BXH   = 0x86
	OP_BXLE  = 0x87
	OP_SRL   = 0x88
	OP_SLL   = 0x89
	OP_SRA   = 0x8A
	OP_SLA   = 0x8B
	OP_SRDL  = 0x8C
	OP_SLDL  = 0x8D
	OP_SRDA  = 0x8E
	OP_SLDA  = 0x8F
	OP_STM   = 0x90
	OP_TM    = 0x91
	OP_MVI   = 0x92
	OP_TS    = 0x93
	OP_NI    = 0x94
	OP_CLI   = 0x95
	OP_OI    = 0x96
	OP_XI    = 0x97
	OP_LM    = 0x98
	OP_SIO   = 0x9C
	OP_TIO   = 0x9D
	OP_HIO   = 0x9E
	OP_TCH   = 0x9F
	OP_STNSM = 0xAC // 370 Store then and system mask
	OP_STOSM = 0xAD // 370 Store then or system mask
	OP_SIGP  = 0xAE // 370 Signal processor
	OP_MC    = 0xAF // 370 Monitor call
	OP_STMC  = 0xB0 // 360/67 Store control
	OP_LRA   = 0xB1
	OP_370   = 0xB2 // Misc 370 system instructions
	OP_STCTL = 0xB6 // 370 Store control
	OP_LCTL  = 0xB7 // 370 Load control
	OP_LMC   = 0xB8 // 360/67 Load Control
	OP_CS    = 0xBA // 370 Compare and swap
	OP_CDS   = 0xBB // 370 Compare double and swap
	OP_CLM   = 0xBD // 370 Compare character under mask
	OP_STCM  = 0xBE // 370 Store character under mask
	OP_ICM   = 0xBF // 370 Insert character under mask
	OP_MVN   = 0xD1
	OP_MVC   = 0xD2
	OP_MVZ   = 0xD3
	OP_NC    = 0xD4
	OP_CLC   = 0xD5
	OP_OC    = 0xD6
	OP_XC    = 0xD7
	OP_TR    = 0xDC
	OP_TRT   = 0xDD
	OP_ED    = 0xDE
	OP_EDMK  = 0xDF
	OP_MVCIN = 0xE8 // 370 Move inverse
	OP_SRP   = 0xF0 // 370 Shift and round decimal
	OP_MVO   = 0xF1
	OP_PACK  = 0xF2
	OP_UNPK  = 0xF3
	OP_ZAP   = 0xF8
	OP_CP    = 0xF9
	OP_AP    = 0xFA
	OP_SP    = 0xFB
	OP_MP    = 0xFC
	OP_DP    = 0xFD
)
