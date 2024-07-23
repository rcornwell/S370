/*
   CPU definitions for IBM 370 simulator definitions

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

type cpuState struct {
	PC       uint32     // Program counter
	iPC      uint32     // Initial PC for instruction
	regs     [16]uint32 // Internal registers
	fpregs   [8]uint64  // Floating point registers
	cregs    [16]uint32 // Control registers /67 or 370 only
	sysMask  uint16     // Channel interrupt enable
	stKey    uint8      // Current storage key
	ecMode   bool       // Current PSW is EC Mode
	cc       uint8      // Current CC code
	ilc      uint8      // Current instruction length
	progMask uint8      // Program mask
	flags    uint8      // System flags
	pageEnb  bool       // Paging enabled

	tlb         [256]uint32 // Translation Lookaside Buffer
	pageShift   uint32      // Amount to shift for page
	pageMask    uint32      // Mask of bits in page address
	pageIndex   uint32      // PTE index mask
	segShift    uint32      // Amount to shift for segment
	segMask     uint32      // Mask bits for segment
	segLen      uint32      // Length of segment table
	segAddr     uint32      // Address of segment table
	pteLenShift uint32      // Shift to Check if out of page table
	pteAvail    uint32      // Mask of available bit in PTE
	pteMBZ      uint32      // Bits that must be zero in PTE
	pteShift    uint32      // Bits to shift a PTE entry

	perEnb    bool   // Enable PER tracing
	perRegMod uint32 // Module modification mask
	perCode   uint16 // Code for PER
	perAddr   uint32 // Address of last reference
	perBranch bool   // Trap on successful branch
	perFetch  bool   // Trap Fetch of instructions
	perStore  bool   // Trap on storage modify
	perReg    bool   // Trap on register modify

	irqEnb   bool      // Interrupts enabled
	extEnb   bool      // External interrupts enabled
	extIrq   bool      // External interrupt pending
	intIrq   bool      // Interval timer interrupt
	intEnb   bool      // Interval timer enable
	todClock [2]uint32 // Current Time of Day Clock
	todSet   bool      // TOD set to correct time

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
	CSW      uint32 = 0x40 // Channel Status Word
	CAW      uint32 = 0x48 // Channel Address Word
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
	// Debug options.
	debugCmd = 1 << iota
	debugInst
	debugData
	debugDetail
	debugIO
	debugIRQ
)

var debugOption = map[string]int{
	"CMD":    debugCmd,  // Debug I/O commands.
	"INST":   debugInst, // Debug instruction execution.
	"DATA":   debugData,
	"DETAIL": debugDetail,
	"IO":     debugIO,
	"IRQ":    debugIRQ,
}

var debugMsk int
