/*
ibm370 IBM 370 Channel Interface functions

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
package device

// Interface for devices to handle commands.
type Device interface {
	StartIO() uint8           // Start of command chain.
	StartCmd(cmd uint8) uint8 // Start command.
	HaltIO() uint8            // Halt I/O instruction issued.
	InitDev() uint8           // Initialize device.
	Shutdown()                // Shutdown device, close any open files.
	Debug(debug string) error // Enable debug option.
}

// Channel types.
const (
	TypeDis  int = 0 // Channel disabled
	TypeSel  int = 1 // Selector channel
	TypeMux  int = 2 // Mulitplexer channel
	TypeBMux int = 3 // Block multiplexer channel
	TypeUNA  int = 4 // Channel unavailable
)

// Device responses.
const (
	// Common Channel sense bytes.
	CStatusAttn   uint8 = 0x80 // Unit attention
	CStatusSMS    uint8 = 0x40 // Status modifier
	CStatusCtlEnd uint8 = 0x20 // Control unit end
	CStatusBusy   uint8 = 0x10 // Unit Busy
	CStatusChnEnd uint8 = 0x08 // Channel end
	CStatusDevEnd uint8 = 0x04 // Device end
	CStatusCheck  uint8 = 0x02 // Unit check
	CStatusExpt   uint8 = 0x01 // Unit exception

	// Command masks.
	// CMD_TYPE uint8 = 0x3 // Type mask.
	// CMD_CHAN  uint8 = 0x0 // Channel command.
	CmdWrite uint8 = 0x1 // Write command
	CmdRead  uint8 = 0x2 // Read command
	CmdCTL   uint8 = 0x3 // Control command
	CmdSense uint8 = 0x4 // Sense channel command
	CmdTIC   uint8 = 0x8 // Transfer in channel
	CmdRDBWD uint8 = 0xc // Read backward

	NoDev uint16 = 0xffff // Code for no device

	// Basic sense information.
	SenseCMDREJ  uint8 = 0x80 // Command reject
	SenseINTVENT uint8 = 0x40 // Unit intervention required
	SenseBUSCHK  uint8 = 0x20 // Parity error on bus
	SenseEQUCHK  uint8 = 0x10 // Equipment check
	SenseDATCHK  uint8 = 0x08 // Data Check
	SenseUNITSPC uint8 = 0x04 // Specific to unit
	SenseCTLCHK  uint8 = 0x02 // Timeout on device
	SenseOVRRUN  uint8 = 0x02 // Data Overrun
	SenseOPRCHK  uint8 = 0x01 // Invalid operation to device
)

// Values to retrieve or set CPU registers.
const (
	Register = 1 + iota
	FPRegister
	CtlRegister
	PSWRegister
	PCRegister
	Symbolic
	Memory
)
