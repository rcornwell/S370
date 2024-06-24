package sys_channel

import D "github.com/rcornwell/S370/emu/device"

/*
 * S370 - Channel Channel definitions
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

// Interface for devices to handle commands
type Device interface {
	StartIO() uint8
	StartCmd(cmd uint8) uint8
	HaltIO() uint8
	InitDev() uint8
}

// Holds individual subchannel control information
type chanCtl struct { // Channel control structure
	dev        D.Device // Pointer to device interface
	caw        uint32   // Channel command address word
	ccwAddr    uint32   // Channel address
	ccwIAddr   uint32   // Channel indirect address
	ccwCount   uint16   // Channel count
	ccwCmd     uint8    // Channel command and flags
	ccwKey     uint8    // Channel key
	ccwFlags   uint16   // Channel control flags
	chanBuffer uint32   // Channel data buffer
	chanStatus uint16   // Channel status
	chanDirty  bool     // Buffer has been modified
	devAddr    uint16   // Device on channel
	chanByte   uint8    // Current byte, dirty/full
	chainFlg   bool     // Holding on chain
}

// Holds channel information
type chanDev struct { // Channel information
	devTab     [256]D.Device // Pointer to device interfaces
	devStatus  [256]uint8    // Status from each device
	chanType   int           // Type of channel
	numSubChan int           // Number of subchannels
	irqPending bool          // Channel has pending IRQ
	subChans   [256]chanCtl  // Subchannel control
	enabled    bool          // Channel enabled
}

var IrqPending bool
var Loading uint16 = NoDev

const (
	MAX_CHAN uint16 = 12 // Max number of channels
	TypeDis  int    = 0  // Channel disabled
	TypeSel  int    = 1  // Selector channel
	TypeMux  int    = 2  // Mulitplexer channel
	TypeBMux int    = 3  // Block multiplexer channel
	TypeUNA  int    = 4  // Channel unavailable

	cmdMask    uint32 = 0xff000000 // Mask for command
	keyMask    uint32 = 0xf0000000 // Channel key mask
	addrMask   uint32 = 0x00ffffff // Mask for data address
	countMask  uint32 = 0x0000ffff // Mask for data count
	flagMask   uint32 = 0xfc000000 // Mask for flags
	statusMask uint32 = 0xffff0000 // Mask for status bits

	errorStatus uint16 = (statusAttn | statusPCI | statusExcept | statusCheck |
		statusProt | statusCDChk | statusCCChk | statusCIChk | statusChain)

	chainData uint16 = 0x8000 // Chain data
	chainCmd  uint16 = 0x4000 // Chain command
	flagSLI   uint16 = 0x2000 // Suppress length indicator
	flagSkip  uint16 = 0x1000 // Suppress memory write
	flagPCI   uint16 = 0x0800 // Program controled interrupt
	flagIDA   uint16 = 0x0400 // Channel indirect

	bufEmpty uint8 = 0x04 // Buffer is empty
	bufEnd   uint8 = 0x10 // Device has returned channel end, no more data

	// Common Channel sense bytes
	CStatusAttn   uint8 = 0x80 // Unit attention
	CStatusSMS    uint8 = 0x40 // Status modifier
	CStatusCtlEnd uint8 = 0x20 // Control unit end
	CStatusBusy   uint8 = 0x10 // Unit Busy
	CStatusChnEnd uint8 = 0x08 // Channel end
	CStatusDevEnd uint8 = 0x04 // Device end
	CStatusCheck  uint8 = 0x02 // Unit check
	CStatusExpt   uint8 = 0x01 // Unit exception

	// Command masks
	//CMD_TYPE uint8 = 0x3 // Type mask
	//	CMD_CHAN  uint8 = 0x0 // Channel command
	CmdWrite uint8 = 0x1 // Write command
	CmdRead  uint8 = 0x2 // Read command
	CmdCTL   uint8 = 0x3 // Control command
	CmdSense uint8 = 0x4 // Sense channel command
	CmdTIC   uint8 = 0x8 // Transfer in channel
	CmdRDBWD uint8 = 0xc // Read backward

	// Channel status information
	statusAttn   uint16 = 0x8000 // Device raised attention
	statusSMS    uint16 = 0x4000 // Status modifier
	statusCtlEnd uint16 = 0x2000 // Control end
	statusBusy   uint16 = 0x1000 // Device busy
	statusChnEnd uint16 = 0x0800 // Channel end
	statusDevEnd uint16 = 0x0400 // Device end
	statusCheck  uint16 = 0x0200 // Unit check
	statusExcept uint16 = 0x0100 // Unit excpetion
	statusPCI    uint16 = 0x0080 // Program interupt
	statusLength uint16 = 0x0040 // Incorrect length
	statusPCHK   uint16 = 0x0020 // Program check
	statusProt   uint16 = 0x0010 // Protection check
	statusCDChk  uint16 = 0x0008 // Channel data check
	statusCCChk  uint16 = 0x0004 // Channel control check
	statusCIChk  uint16 = 0x0002 // Channel interface check
	statusChain  uint16 = 0x0001 // Channel chain check

	NoDev uint16 = 0xffff // Code for no device

	// Basic sense information
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
