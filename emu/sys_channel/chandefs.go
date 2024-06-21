package sys_channel

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

import (
	D "github.com/rcornwell/S370/internal/device"
)

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

	chanStatus uint16 // Channel status
	chanDirty  bool   // Buffer has been modified
	devAddr    uint16 // Device on channel
	chanByte   uint8  // Current byte, dirty/full
	chainFlg   bool   // Holding on chain
}

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
var Loading uint16 = NO_DEV

const (
	MAX_CHAN  uint16 = 12 // Max number of channels
	TYPE_DIS  int    = 0  // Channel disabled
	TYPE_SEL  int    = 1  // Selector channel
	TYPE_MUX  int    = 2  // Mulitplexer channel
	TYPE_BMUX int    = 3  // Block multiplexer channel
	TYPE_UNA  int    = 4  // Channel unavailable

	CCMDMSK  uint32 = 0xff000000 // Mask for command
	CDADRMSK uint32 = 0x00ffffff // Mask for data address
	CCNTMSK  uint32 = 0x0000ffff // Mask for data count
	CD       uint32 = 0x80000000 // Chain data
	CC       uint32 = 0x40000000 // Chain command
	SLI      uint32 = 0x20000000 // Suppress length indication
	SKIP     uint32 = 0x10000000 // Skip flag
	PCI      uint32 = 0x08000000 // Program controlled interuption
	IDA      uint32 = 0x04000000 // Indirect Channel addressing

	ERROR_STATUS uint16 = (STATUS_ATTN | STATUS_PCI | STATUS_EXPT | STATUS_CHECK |
		STATUS_PROT | STATUS_CDATA | STATUS_CCNTL | STATUS_INTER |
		STATUS_CHAIN)

	FLAG_CD   uint16 = 0x8000 // Chain data
	FLAG_CC   uint16 = 0x4000 // Chain command
	FLAG_SLI  uint16 = 0x2000 // Suppress length indicator
	FLAG_SKIP uint16 = 0x1000 // Suppress memory write
	FLAG_PCI  uint16 = 0x0800 // Program controled interrupt
	FLAG_IDA  uint16 = 0x0400 // Channel indirect

	BUFF_EMPTY  uint8 = 0x04 // Buffer is empty
	BUFF_CHNEND uint8 = 0x10 // Channel end

	// Channel sense bytes
	SNS_ATTN    uint8 = 0x80 // Unit attention
	SNS_SMS     uint8 = 0x40 // Status modifier
	SNS_CTLEND  uint8 = 0x20 // Control unit end
	SNS_BSY     uint8 = 0x10 // Unit Busy
	SNS_CHNEND  uint8 = 0x08 // Channel end
	SNS_DEVEND  uint8 = 0x04 // Device end
	SNS_UNITCHK uint8 = 0x02 // Unit check
	SNS_UNITEXP uint8 = 0x01 // Unit exception

	// Command masks
	CMD_TYPE  uint8 = 0x3 // Type mask
	CMD_CHAN  uint8 = 0x0 // Channel command
	CMD_WRITE uint8 = 0x1 // Write command
	CMD_READ  uint8 = 0x2 // Read command
	CMD_CTL   uint8 = 0x3 // Control command
	CMD_SENSE uint8 = 0x4 // Sense channel command
	CMD_TIC   uint8 = 0x8 // Transfer in channel
	CMD_RDBWD uint8 = 0xc // Read backward

	STATUS_ATTN   uint16 = 0x8000 // Device raised attention
	STATUS_SMS    uint16 = 0x4000 // Status modifier
	STATUS_CTLEND uint16 = 0x2000 // Control end
	STATUS_BUSY   uint16 = 0x1000 // Device busy
	STATUS_CEND   uint16 = 0x0800 // Channel end
	STATUS_DEND   uint16 = 0x0400 // Device end
	STATUS_CHECK  uint16 = 0x0200 // Unit check
	STATUS_EXPT   uint16 = 0x0100 // Unit excpetion
	STATUS_PCI    uint16 = 0x0080 // Program interupt
	STATUS_LENGTH uint16 = 0x0040 // Incorrect lenght
	STATUS_PCHK   uint16 = 0x0020 // Program check
	STATUS_PROT   uint16 = 0x0010 // Protection check
	STATUS_CDATA  uint16 = 0x0008 // Channel data check
	STATUS_CCNTL  uint16 = 0x0004 // Channel control check
	STATUS_INTER  uint16 = 0x0002 // Channel interface check
	STATUS_CHAIN  uint16 = 0x0001 // Channel chain check

	NO_DEV uint16 = 0xffff // Code for no device

	// Basic sense information
	SNS_CMDREJ  uint8 = 0x80 // Command reject
	SNS_INTVENT uint8 = 0x40 // Unit intervention required
	SNS_BUSCHK  uint8 = 0x20 // Parity error on bus
	SNS_EQUCHK  uint8 = 0x10 // Equipment check
	SNS_DATCHK  uint8 = 0x08 // Data Check
	SNS_UNITSPC uint8 = 0x04 // Specific to unit
	SNS_CTLCHK  uint8 = 0x02 // Timeout on device
	SNS_OVRRUN  uint8 = 0x02 // Data Overrun
	SNS_OPRCHK  uint8 = 0x01 // Invalid operation to device
)
