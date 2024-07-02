/* ibm370 IBM 370 Channel functions.

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

package syschannel

import (
	D "github.com/rcornwell/S370/emu/device"
	M "github.com/rcornwell/S370/emu/memory"
)

const (
	MaxChan uint16 = 12 // Max number of channels

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
	flagPCI   uint16 = 0x0800 // Program controlled interrupt
	flagIDA   uint16 = 0x0400 // Channel indirect

	bufEmpty uint8 = 0x04 // Buffer is empty
	bufEnd   uint8 = 0x10 // Device has returned channel end, no more data

	// Channel status information.
	statusAttn   uint16 = 0x8000 // Device raised attention
	statusSMS    uint16 = 0x4000 // Status modifier
	statusCtlEnd uint16 = 0x2000 // Control end
	statusBusy   uint16 = 0x1000 // Device busy
	statusChnEnd uint16 = 0x0800 // Channel end
	statusDevEnd uint16 = 0x0400 // Device end
	statusCheck  uint16 = 0x0200 // Unit check
	statusExcept uint16 = 0x0100 // Unit excpretion
	statusPCI    uint16 = 0x0080 // Program interrupt
	statusLength uint16 = 0x0040 // Incorrect length
	statusPCHK   uint16 = 0x0020 // Program check
	statusProt   uint16 = 0x0010 // Protection check
	statusCDChk  uint16 = 0x0008 // Channel data check
	statusCCChk  uint16 = 0x0004 // Channel control check
	statusCIChk  uint16 = 0x0002 // Channel interface check
	statusChain  uint16 = 0x0001 // Channel chain check
)

// Holds individual subchannel control information.
type chanCtl struct {
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

// Holds channel information.
type chanDev struct {
	devTab     [256]D.Device // Pointer to device interfaces
	devStatus  [256]uint8    // Status from each device
	chanType   int           // Type of channel
	numSubChan int           // Number of subchannels
	irqPending bool          // Channel has pending IRQ
	subChans   []chanCtl     // Subchannel control
}

var (
	IrqPending bool
	Loading    = D.NoDev

	// Hold information about channels.
	chanUnit [16]*chanDev

	// Are block multiplexer channels enabled.
	bmuxEnable bool

	// Empty device for initialization.
	nullDev D.Device
)

// Set whether Block multiplexer is enabled or not.
func SetBMUXenable(enable bool) {
	bmuxEnable = enable
}

// Return type of channel.
func GetType(devNum uint16) int {
	cUnit := chanUnit[(devNum>>8)&0xf]
	if cUnit == nil {
		return D.TypeUNA
	}
	return cUnit.chanType
}

// Process SIO instruction.
func StartIO(devNum uint16) uint8 {
	chNum := (devNum >> 8) & 0xf
	dNum := devNum & 0xff
	subChan := findSubChannel(devNum)
	cUnit := chanUnit[chNum]
	// Check if channel disabled
	if cUnit == nil {
		return 3
	}

	// If no device or channel, return CC = 3
	if cUnit.devTab[dNum] == nullDev || subChan == nil {
		return 3
	}

	// If pending status is for us, return it with status code
	if subChan.devAddr == devNum && subChan.chanStatus != 0 {
		storeCSW(subChan)
		return 1
	}

	// If channel is active return cc = 2
	if subChan.ccwCmd != 0 || (subChan.ccwFlags&(chainCmd|chainData)) != 0 || subChan.chanStatus != 0 {
		return 2
	}

	dStatus := cUnit.devStatus[dNum]
	if dStatus == D.CStatusDevEnd || dStatus == (D.CStatusDevEnd|D.CStatusChnEnd) {
		cUnit.devStatus[dNum] = 0
		dStatus = 0
	}

	// Check for any pending status for this device
	if dStatus != 0 {
		M.SetMemory(0x44, uint32(dStatus)<<24)
		M.SetMemory(0x40, 0)
		cUnit.devStatus[dNum] = 0
		return 1
	}

	status := uint16(cUnit.devTab[dNum].StartIO()) << 8
	if (status & statusBusy) != 0 {
		return 2
	}
	if status != 0 {
		M.PutWordMask(0x44, uint32(status)<<16, statusMask)
		return 1
	}

	// All ok, get caw address
	subChan.chanStatus = 0
	subChan.caw = M.GetMemory(0x48)
	subChan.ccwKey = uint8(((subChan.caw & keyMask) >> 24) & 0xff)
	subChan.caw &= addrMask
	subChan.devAddr = devNum
	subChan.dev = cUnit.devTab[dNum]
	cUnit.devStatus[dNum] = 0

	if loadCCW(cUnit, subChan, false) {
		M.SetMemoryMask(0x44, uint32(subChan.chanStatus)<<16, statusMask)
		subChan.chanStatus = 0
		subChan.ccwCmd = 0
		subChan.devAddr = D.NoDev
		subChan.dev = nil
		cUnit.devStatus[dNum] = 0
		return 1
	}

	// If channel returned busy save CSW and return CC = 1
	if (subChan.chanStatus & statusBusy) != 0 {
		M.SetMemoryMask(0x44, uint32(subChan.chanStatus)<<16, statusMask)
		subChan.chanStatus = 0
		subChan.ccwCmd = 0
		subChan.devAddr = D.NoDev
		subChan.dev = nil
		cUnit.devStatus[dNum] = 0
		return 1
	}

	// If immediate command and not command chainting
	if (subChan.chanStatus&statusChnEnd) != 0 && (subChan.ccwFlags&chainCmd) == 0 {
		// If we also have data end write out fill CSW and mark subchannel free

		if (subChan.chanStatus & statusDevEnd) != 0 {
			storeCSW(subChan)
		} else {
			M.SetMemoryMask(0x44, uint32(subChan.chanStatus)<<16, statusMask)
		}
		subChan.ccwCmd = 0
		subChan.devAddr = D.NoDev
		subChan.dev = nil
		cUnit.devStatus[dNum] = 0
		subChan.chanStatus = 0

		return 1
	}

	// If immediate command and chaining report status, but don't clear things
	if (subChan.chanStatus&(statusChnEnd|statusDevEnd)) == statusChnEnd && (subChan.ccwFlags&chainCmd) != 0 {
		M.SetMemoryMask(0x44, uint32(subChan.chanStatus)<<16, statusMask)
		return 1
	}

	return 0
}

// Handle TIO instruction.
func TestIO(devNum uint16) uint8 {
	ch := (devNum >> 8) & 0xf
	dNum := devNum & 0xff
	subChan := findSubChannel(devNum)

	cUnit := chanUnit[ch]
	// Check if channel disabled
	if cUnit == nil {
		return 3
	}

	// If no device or channel, return CC = 3
	if cUnit.devTab[dNum] == nullDev || subChan == nil {
		return 3
	}

	// If any error pending save csw and return cc=1
	if (subChan.chanStatus & errorStatus) != 0 {
		storeCSW(subChan)
		return 1
	}

	// If channel active, return cc=2
	if subChan.ccwCmd != 0 || (subChan.ccwFlags&(chainCmd|chainData)) != 0 {
		return 2
	}

	// Device finished and channel status pending return it and cc=1
	if subChan.ccwCmd == 0 && subChan.chanStatus != 0 {
		storeCSW(subChan)
		subChan.devAddr = D.NoDev
		return 1
	}

	// Device has returned a status, store the csw and return cc=1
	if cUnit.devStatus[dNum] != 0 {
		M.SetMemory(0x40, 0)
		M.SetMemory(0x44, (uint32(cUnit.devStatus[dNum]) << 24))
		cUnit.devStatus[dNum] = 0
		return 1
	}

	// If error pending for another device on subchannel, return cc = 2
	if cUnit.irqPending {
		// Check if might be false
		for d := range uint16(256) {
			if cUnit.devStatus[d] != 0 {
				// Check if same subchannel
				if findSubChannel(d) == subChan {
					cUnit.irqPending = true
					IrqPending = true
					return 2
				}
			}
		}
	}

	// Nothing pending, send a 0 command to device to get status
	status := uint16(cUnit.devTab[dNum].StartCmd(0)) << 8

	// If we get a error, save csw and return cc = 1
	if (status & errorStatus) != 0 {
		M.SetMemoryMask(0x44, uint32(status)<<16, statusMask)
		return 1
	}

	// Check if device BUSY
	if (status & statusBusy) != 0 {
		return 2
	}

	// Everything ok, return cc = 0
	return 0
}

// Handle HIO instruction.
func HaltIO(devNum uint16) uint8 {
	ch := (devNum >> 8) & 0xf
	dNum := devNum & 0xff
	subChan := findSubChannel(devNum)

	cUnit := chanUnit[ch]
	// Check if channel disabled
	if cUnit == nil {
		return 3
	}

	// If no device or channel, return CC = 3
	if cUnit.devTab[dNum] == nullDev || subChan == nil {
		return 3
	}

	// Generic halt I/O, tell device to stop end
	// If any error pending save csw and return cc = 1
	if (subChan.chanStatus & errorStatus) != 0 {
		return 1
	}

	// If channel active, tell it to terminate
	if subChan.ccwCmd != 0 {
		subChan.chanByte = bufEmpty
		subChan.ccwFlags &= ^(chainCmd | chainData)
	}

	// Executing a command, issue halt if available
	// Let device try to halt
	cc := cUnit.devTab[dNum].HaltIO()
	if cc == 1 {
		M.SetMemoryMask(0x44, (uint32(subChan.chanStatus) << 16), statusMask)
	}
	return cc
}

// Handle TCH instruction.
func TestChan(devNum uint16) uint8 {
	/* 360 Principles of Operation says, "Bit positions 21-23 of the
	   sum formed by the addition of the content of register B1 and the
	   content of the D1 field identify the channel to which the
	   instruction applies. Bit positions 24-31 of the address are ignored.
	   /67 Functional Characteristics do not mention any changes in basic or
	   extended control mode of the TCH instruction behavior.
	   However, historic /67 code for MTS suggests that bits 19-20 of the
	   address indicate the channel controller which should be used to query
	   the channel.

	   Original testchan code did not recognize the channel controller (CC) part
	   of the address and treats the query as referring to a channel # like so:
	   CC = 0 channel# 0  1  2  3  4  5  6
	   CC = 1    "     8  9 10 11 12 13 14
	   CC = 2    "    16 17 18 19 20 21 22
	   CC = 3    "    24 25 26 27 28 29 30
	   which may interfere with subchannel mapping.

	   For the nonce, TCH only indicates that channels connected to CC 0 & 1 are
	   attached.  Channels 0, 4, 8 (0 on CC 1) & 12 (4 on CC 1) are multiplexer
	   channels. */
	ch := (devNum >> 8) & 0xf
	cUnit := chanUnit[ch]
	// Check if channel disabled
	if cUnit == nil {
		return 3
	}

	// Multiplexer channel always returns available
	if cUnit.chanType == D.TypeMux {
		return 0
	}

	// If Block Multiplexer channel operating in select mode
	if cUnit.chanType == D.TypeBMux && bmuxEnable {
		return 0
	}

	subChan := cUnit.subChans[0]
	// If channel is executing a command, return cc = 2
	if subChan.ccwCmd != 0 || (subChan.ccwFlags&(chainCmd|chainData)) != 0 {
		return 2
	}

	// If pending status, return 1
	if subChan.chanStatus != 0 {
		return 1
	}

	return 0
}

// Read a byte from memory.
func ChanReadByte(devNum uint16) (uint8, bool) {
	// Return abort if no channel
	subChan := findSubChannel(devNum)
	if subChan == nil {
		return 0, true
	}
	// Channel has pending system status
	if (subChan.chanStatus & 0x7f) != 0 {
		return 0, true
	}
	// Not read command
	if (subChan.ccwCmd & 1) == 0 {
		return 0, true
	}
	// Check if transfer is finished
	if subChan.chanByte == bufEnd {
		return 0, true
	}

	cUnit := chanUnit[(devNum>>8)&0xf]
	// Check if count zero
	if subChan.ccwCount == 0 {
		// If not data chaining, let device know there will be no
		// more data to come
		if (subChan.ccwFlags & chainData) == 0 {
			subChan.chanStatus |= statusChnEnd
			subChan.chanByte = bufEnd
			return 0, true
		}
		// If chaining try and start next CCW
		if loadCCW(cUnit, subChan, true) {
			return 0, true
		}
	}

	// Read in next word if buffer is in empty status
	if subChan.chanByte == bufEmpty {
		if readBuffer(cUnit, subChan) {
			return 0, true
		}
		if nextAddress(cUnit, subChan) {
			return 0, true
		}
	}

	// Return current byte
	subChan.ccwCount--
	data := uint8(subChan.chanBuffer >> (8 * (3 - (subChan.chanByte & 3))) & 0xff)
	subChan.chanByte++
	// If count is zero and chaining load in new CCW
	if subChan.ccwCount == 0 && (subChan.ccwFlags&chainData) != 0 {
		// If chaining try and start next CCW
		if loadCCW(cUnit, subChan, true) {
			// Return that this is last byte device will get
			return data, true
		}
	}
	return data, false
}

// Write a byte to memory.
func ChanWriteByte(devNum uint16, data uint8) bool {
	// Return abort if no channel
	subChan := findSubChannel(devNum)
	if subChan == nil {
		return true
	}
	// Channel has pending system status
	if (subChan.chanStatus & 0x7f) != 0 {
		return true
	}
	// Not read command
	if (subChan.ccwCmd & 1) != 0 {
		return true
	}
	// Check if transfer is finished
	if subChan.chanByte == bufEnd {
		if (subChan.ccwFlags & flagSLI) == 0 {
			subChan.chanStatus |= statusLength
		}
		return true
	}
	cUnit := chanUnit[(devNum>>8)&0xf]
	// Check if count zero
	if subChan.ccwCount == 0 {
		if subChan.chanDirty {
			if writeBuffer(cUnit, subChan) {
				return true
			}
		}
		// If not data chaining, let device know there will be no
		// more data to come
		if (subChan.ccwFlags & chainData) == 0 {
			subChan.chanByte = bufEnd
			if (subChan.ccwFlags & flagSLI) == 0 {
				subChan.chanStatus |= statusLength
			}
			return true
		}
		// Otherwise try and grab next CCW
		if loadCCW(cUnit, subChan, true) {
			return true
		}
	}

	// If we are skipping, just adjust count
	if (subChan.ccwFlags & flagSkip) != 0 {
		subChan.ccwCount--
		subChan.chanByte = bufEmpty
		return nextAddress(cUnit, subChan)
	}

	// Check if we need to save what we have
	if subChan.chanByte == bufEmpty && subChan.chanDirty {
		if writeBuffer(cUnit, subChan) {
			return true
		}
		if nextAddress(cUnit, subChan) {
			return true
		}
		subChan.chanByte = bufEmpty
	}
	if subChan.chanByte == bufEmpty {
		if readBuffer(cUnit, subChan) {
			return true
		}
	}

	// Store it in buffer and adjust pointer
	subChan.ccwCount--
	offset := 8 * (subChan.chanByte & 3)
	mask := uint32(0xff000000 >> offset)
	subChan.chanBuffer &= ^mask
	subChan.chanBuffer |= uint32(data) << (24 - offset)
	if (subChan.ccwCmd & 0xf) == D.CmdRDBWD {
		if (subChan.chanByte & 3) != 0 {
			subChan.chanByte--
		} else {
			subChan.chanByte = bufEmpty
		}
	} else {
		subChan.chanByte++
	}
	subChan.chanDirty = true
	// If count is zero and chaining load in new CCW
	if subChan.ccwCount == 0 && (subChan.ccwFlags&chainData) != 0 {
		// Flush buffer
		if subChan.chanDirty && writeBuffer(cUnit, subChan) {
			return true
		}
		// If chaining try and start next CCW
		if loadCCW(cUnit, subChan, true) {
			// Return that this is last byte device will get
			return true
		}
	}
	return false
}

// Compute address of next byte to read/write.
func nextAddress(cUnit *chanDev, subChan *chanCtl) bool {
	if (subChan.ccwFlags & flagIDA) != 0 {
		if (subChan.ccwCmd & 0xf) == D.CmdRDBWD {
			subChan.ccwIAddr--
			if (subChan.ccwIAddr & 0x7ff) == 0x7ff {
				subChan.ccwAddr += 4
				word, err := readFullWord(cUnit, subChan, subChan.ccwAddr)
				if err {
					return true
				}
				subChan.ccwIAddr = word & M.AMASK
			}
		} else {
			subChan.ccwIAddr++
			if (subChan.ccwIAddr & 0x7ff) == 0x000 {
				subChan.ccwAddr += 4
				word, err := readFullWord(cUnit, subChan, subChan.ccwAddr)
				if err {
					return true
				}
				subChan.ccwIAddr = word & M.AMASK
			}
		}
		subChan.chanByte = uint8(subChan.ccwIAddr & 3)
		return false
	}
	if (subChan.ccwCmd & 0xf) == D.CmdRDBWD {
		subChan.ccwAddr -= 1 + (subChan.ccwAddr & 0x3)
	} else {
		subChan.ccwAddr += 4 - (subChan.ccwAddr & 0x3)
	}
	subChan.chanByte = uint8(subChan.ccwAddr & 3)
	return false
}

// Signal end of transfer by device.
func ChanEnd(devNum uint16, flags uint8) {

	// Return abort if no channel
	subChan := findSubChannel(devNum)
	if subChan == nil {
		return
	}

	ch := (devNum >> 8) & 0xf
	cUnit := chanUnit[ch]
	if subChan.chanDirty {
		_ = writeBuffer(cUnit, subChan)
	}
	subChan.chanStatus |= statusChnEnd
	subChan.chanStatus |= uint16(flags) << 8
	subChan.ccwCmd = 0

	// If count not zero and not suppressing length, report error
	if subChan.ccwCount != 0 && (subChan.ccwFlags&flagSLI) == 0 {
		subChan.chanStatus |= statusLength
		subChan.ccwFlags = 0
	}

	// If count not zero and not suppressing length, report error
	if subChan.ccwCount != 0 && (subChan.ccwFlags&(chainData|flagSLI)) == (chainData|flagSLI) {
		subChan.chanStatus |= statusLength
	}

	if (flags & (D.CStatusAttn | D.CStatusCheck | D.CStatusExpt)) != 0 {
		subChan.ccwFlags = 0
	}

	if (flags & D.CStatusDevEnd) != 0 {
		subChan.ccwFlags &= ^(chainData | flagSLI)
	}

	cUnit.irqPending = true
	IrqPending = true
}

// A device wishes to inform the CPU it needs some service.
func SetDevAttn(devNum uint16, flags uint8) {
	subChan := findSubChannel(devNum)
	if subChan == nil {
		return
	}

	ch := (devNum >> 8) & 0xf
	cUnit := chanUnit[ch]
	// Check if chain being held
	if subChan.devAddr == devNum && subChan.chainFlg && (flags&D.CStatusDevEnd) != 0 {
		subChan.chanStatus |= uint16(flags) << 8
	} else {
		// Check if Device is currently on channel
		if subChan.devAddr == devNum && (flags&D.CStatusDevEnd) != 0 &&
			((subChan.chanStatus&statusChnEnd) != 0 || subChan.ccwCmd != 0) {
			subChan.chanStatus |= uint16(flags) << 8
			subChan.ccwCmd = 0
		} else { // Device reporting status change
			cUnit.devStatus[devNum&0xff] = flags
		}
	}
	cUnit.irqPending = true
	IrqPending = true
}

// Scan all channels and see if one is ready to start or has interrupt pending.
func ChanScan(mask uint16, irqEnb bool) uint16 {
	// Quick exit if no pending IRQ's
	if !IrqPending {
		return D.NoDev
	}

	// Clear pending flag
	IrqPending = false
	pendDev := D.NoDev // Device with Pending interrupt
	// Start with channel 0 and work through all channels
	for i := range len(chanUnit) {
		cUnit := chanUnit[i]

		if cUnit == nil {
			continue
		}
		// Mask for this channel
		imask := uint16(0x8000) >> i
		numSubChan := cUnit.numSubChan
		if cUnit.chanType == D.TypeBMux {
			if !bmuxEnable {
				numSubChan = 1
			}
		}
		// Scan all subchannels on this channel
		for j := range numSubChan {
			subChan := &cUnit.subChans[j]
			if subChan.devAddr == D.NoDev {
				continue
			}

			// Check if PCI pending
			if irqEnb && (imask&mask) != 0 && (subChan.chanStatus&statusPCI) != 0 {
				pendDev = subChan.devAddr
				break
			}

			// If device has hard error, store CSW and end.
			if irqEnb && (imask&mask) != 0 && (subChan.chanStatus&0xff) != 0 {
				pendDev = subChan.devAddr
				break
			}

			// If chaining and device end continue
			if subChan.chainFlg && (subChan.chanStatus&statusDevEnd) != 0 {
				// Restart command that was flagged as an issue
				_ = loadCCW(cUnit, subChan, true)
				continue
			}

			if (subChan.chanStatus & statusChnEnd) != 0 {
				// Grab another command if command chaining in effect
				if (subChan.ccwFlags & chainCmd) != 0 {
					// If channel end, check if we should continue
					_ = loadCCW(cUnit, subChan, true)
				} else if irqEnb || Loading != D.NoDev {
					// Disconnect from device
					if (imask&mask) != 0 || Loading != D.NoDev {
						pendDev = subChan.devAddr
						break
					}
				}
			}
		}
	}

	// Only return loading unit on loading
	if Loading != D.NoDev && Loading != pendDev {
		return D.NoDev
	}

	// See if we can post an IRQ
	if pendDev != D.NoDev {
		// Set to scan next time
		IrqPending = true
		subChan := findSubChannel(pendDev)
		cUnit := chanUnit[(pendDev>>8)&0xf]
		if Loading == pendDev {
			subChan.chanStatus = 0
			cUnit.devStatus[pendDev&0xff] = 0
			return pendDev
		}
		if Loading == D.NoDev {
			storeCSW(subChan)
			cUnit.devStatus[pendDev&0xff] = 0
			return pendDev
		}
	} else if irqEnb {
		// If interrupts are wanted, check for pending device status
		for i := range len(chanUnit) {
			cUnit := chanUnit[i]
			if cUnit == nil {
				continue
			}
			// Mask for this channel
			imask := uint16(0x8000 >> i)
			if !cUnit.irqPending || (imask&mask) == 0 {
				continue
			}
			cUnit.irqPending = false
			for j := range 256 {
				// Look for device with pending status
				if cUnit.devStatus[j] != 0 {
					cUnit.irqPending = true
					IrqPending = true
					M.SetMemory(0x44, uint32(cUnit.devStatus[j])<<24)
					M.SetMemory(0x40, 0)
					cUnit.devStatus[j] = 0
					return (uint16(i) << 8) | uint16(j)
				}
			}
		}
	}
	// No pending device
	return D.NoDev
}

// IPL a device.
func BootDevice(devNum uint16) bool {
	ch := (devNum >> 8) & 0xf
	dNum := devNum & 0xff
	subChan := findSubChannel(devNum)
	cUnit := chanUnit[ch]

	// Check if channel disabled
	if subChan == nil || cUnit == nil {
		return true
	}

	// If no device or channel, return CC = 3
	if cUnit.devTab[dNum] == nil || subChan == nil {
		return true
	}
	status := uint16(cUnit.devTab[dNum].StartIO()) << 8
	if status != 0 {
		return true
	}

	subChan.chanStatus = 0
	subChan.dev = cUnit.devTab[dNum]
	subChan.caw = 0x8
	subChan.devAddr = devNum
	subChan.ccwCount = 24
	subChan.ccwFlags = chainCmd | flagSLI
	subChan.ccwAddr = 0
	subChan.ccwKey = 0
	subChan.chanByte = bufEmpty
	subChan.chanDirty = false

	subChan.chanStatus |= uint16(subChan.dev.StartCmd(subChan.ccwCmd)) << 8

	// Check if any errors from initial command
	if (subChan.chanStatus & (statusAttn | statusCheck | statusExcept)) != 0 {
		subChan.ccwCmd = 0
		subChan.ccwFlags = 0
		return true
	}
	Loading = devNum
	return false
}

// Add a device at given address.
func AddDevice(dev D.Device, devNum uint16) bool {
	ch := (devNum >> 8) & 0xf
	dNum := devNum & 0xff
	cUnit := chanUnit[ch]
	// Check if channel disabled
	if cUnit == nil {
		return false
	}

	if cUnit.devTab[dNum] != nil {
		return false
	}
	cUnit.devTab[dNum] = dev
	return true
}

// Get a device pointer
func GetDevice(devNum uint16) D.Device {
	ch := (devNum >> 8) & 0xf
	dNum := devNum & 0xff
	cUnit := chanUnit[ch]
	return cUnit.devTab[dNum]
}

// Delete a device at a given address.
func DelDevice(devNum uint16) {
	ch := (devNum >> 8) & 0xf
	dNum := devNum & 0xff
	cUnit := chanUnit[ch]
	cUnit.devTab[dNum] = nil
	cUnit.devStatus[dNum] = 0
}

// Enable a channel of a given type.
func AddChannel(cNum int, ty int, subchan int) {
	if cNum > len(chanUnit) {
		return
	}

	if chanUnit[cNum] != nil {
		return
	}

	numSubChan := subchan
	switch ty {
	case D.TypeSel:
		numSubChan = 1
	case D.TypeMux:
		numSubChan = subchan
	case D.TypeBMux:
		numSubChan = 32
	}

	cUnit := chanDev{}
	chanUnit[cNum] = &cUnit
	cUnit.numSubChan = numSubChan
	cUnit.chanType = ty
	sc := [256]chanCtl{}
	cUnit.subChans = sc[:numSubChan]
}

// Initialize all channels and clear any device assignments.
func InitializeChannels() {
	for i := range chanUnit {
		chanUnit[i] = nil
	}
}

/* channel:
    subchannels = 128
    0 - 7       0x80-0xff
   8 - 127     0x00-0x7f
   128 - +6    0x1xx - 0x6xx
*/

// Look up device to find subchannel device is on.
func findSubChannel(devNum uint16) *chanCtl {
	ch := (devNum >> 8) & 0xf
	dNum := int(devNum & 0xff)
	cUnit := chanUnit[ch]
	if cUnit == nil {
		return nil
	}
	switch cUnit.chanType {
	case D.TypeSel:
		return &cUnit.subChans[0]
	case D.TypeBMux:
		if bmuxEnable {
			subChanNum := (devNum >> 3) & 0x1f
			return &cUnit.subChans[subChanNum]
		}
		return &cUnit.subChans[0]
	case D.TypeMux:
		if dNum >= cUnit.numSubChan {
			if dNum < 128 { // All shared devices over subchannels
				return nil
			}
			dNum = (dNum >> 4) & 0x7
		}
		return &cUnit.subChans[dNum]
	}
	return nil
}

// Save full csw.
func storeCSW(cUnit *chanCtl) {
	M.SetMemory(0x40, (uint32(cUnit.ccwKey)<<24)|cUnit.caw)
	M.SetMemory(0x44, uint32(cUnit.ccwCount)|(uint32(cUnit.chanStatus)<<16))
	if (cUnit.chanStatus & statusPCI) != 0 {
		cUnit.chanStatus &= ^statusPCI
	} else {
		cUnit.chanStatus = 0
	}
	cUnit.ccwFlags &= ^flagPCI
}

// Load in the next CCW, return true if failure, false if success.
func loadCCW(cUnit *chanDev, subChan *chanCtl, ticOk bool) bool {
	var word uint32
	var err bool
	var cmdFlag bool
	var chain bool

loop:
	// If last chain, start command
	if subChan.chainFlg && (subChan.ccwFlags&chainData) == 0 {
		chain = true
		subChan.chainFlg = false
		cmdFlag = true
	} else {
		// Abort if ccw not on double word boundary
		if (subChan.caw & 0x7) != 0 {
			subChan.chanStatus = statusPCHK
			return true
		}

		// Abort if we have pending errors
		if (subChan.chanStatus & 0x7F) != 0 {
			return true
		}

		// Remember if we were chainging
		chain = (subChan.ccwFlags & chainCmd) != 0

		// Check if we have status modifier set
		if (subChan.chanStatus & statusSMS) != 0 {
			subChan.caw += 8
			subChan.caw &= addrMask
			subChan.chanStatus &= ^statusSMS
		}

		// Read in next CCW
		word, err = readFullWord(cUnit, subChan, subChan.caw)
		if err {
			return true
		}

		// Next word
		subChan.caw += 4
		subChan.caw &= addrMask

		// TIC can't follow TIC nor bt first in chain
		cmd := uint8((word & cmdMask) >> 24)
		if cmd == D.CmdTIC {
			// Pretend to fetch next word.
			subChan.caw += 4
			subChan.caw &= addrMask
			subChan.ccwCmd = 0
			subChan.ccwFlags = 0
			if ticOk {
				subChan.caw = word & addrMask
				ticOk = false
				goto loop
			}
			subChan.chanStatus = statusPCHK
			cUnit.irqPending = true
			IrqPending = true
			return true
		}

		// Check if not chaining data
		if (subChan.ccwFlags & chainData) == 0 {
			subChan.ccwCmd = cmd
			cmdFlag = true
		}

		// Set up for this command
		subChan.ccwAddr = word & addrMask
		word, err = readFullWord(cUnit, subChan, subChan.caw)
		if err {
			return true
		}
		subChan.caw += 4
		subChan.caw &= addrMask
		subChan.ccwCount = uint16(word & countMask)

		// Copy SLI indicator in CD command
		if (subChan.ccwFlags & (chainData | flagSLI)) == (chainData | flagSLI) {
			word |= uint32(flagSLI) << 16
		}
		subChan.ccwFlags = uint16(word>>16) & 0xff00
		subChan.chanByte = bufEmpty

		// Check if invalid count
		if subChan.ccwCount == 0 {
			subChan.chanStatus = statusPCHK
			subChan.ccwCmd = 0
			cUnit.irqPending = true
			IrqPending = true
			return true
		}

		// Handle IDA
		if (subChan.ccwFlags & flagIDA) != 0 {
			word, err = readFullWord(cUnit, subChan, subChan.ccwAddr)
			if err {
				return true
			}
			subChan.ccwIAddr = word & addrMask
		}
	}

	// If command pending start it.
	if cmdFlag {
		// Check if invalid command
		if (subChan.ccwCmd & 0xf) == 0 {
			subChan.chanStatus |= statusPCHK
			subChan.ccwCmd = 0
			cUnit.irqPending = true
			IrqPending = true
			return true
		}

		if subChan.dev == nil {
			return true
		}

		subChan.chanByte = bufEmpty
		status := uint16(subChan.dev.StartCmd(subChan.ccwCmd)) << 8

		// If device is busy, check if last was CC, then mark pending
		if (status & statusBusy) != 0 {
			if chain {
				subChan.chainFlg = true
			}
			return false
		}
		subChan.chanStatus &= 0xff
		subChan.chanStatus |= status
		// Check if any errors from initial command
		if (subChan.chanStatus & (statusAttn | statusCheck | statusExcept)) != 0 {
			subChan.ccwCmd = 0
			subChan.ccwFlags = 0
			cUnit.devStatus[subChan.devAddr&0xff] = uint8((subChan.chanStatus >> 8) & 0xff)
			cUnit.irqPending = true
			IrqPending = true
			return true
		}

		// Check if immediate channel end
		if (subChan.chanStatus & statusChnEnd) != 0 {
			subChan.ccwFlags |= flagSLI // Force SLI for immediate command
			if (subChan.chanStatus & statusChnEnd) != 0 {
				subChan.ccwCmd = 0
				cUnit.irqPending = true
				IrqPending = true
			}
		}
	}

	if (subChan.ccwFlags & flagPCI) != 0 {
		subChan.chanStatus |= statusPCI
		cUnit.irqPending = true
		IrqPending = true
	}
	return false
}

// Read a fill word from memory.
// Return true if fail and false if success.
func readFullWord(cUnit *chanDev, subChan *chanCtl, addr uint32) (uint32, bool) {
	if !M.CheckAddr(addr) {
		subChan.chanStatus |= statusPCHK
		cUnit.irqPending = true
		IrqPending = true
		return 0, true
	}
	if subChan.ccwKey != 0 {
		key := M.GetKey(addr)
		if (key&0x8) != 0 && (key&0xf0) != subChan.ccwKey {
			subChan.chanStatus |= statusProt
			cUnit.irqPending = true
			IrqPending = true
			return 0, true
		}
	}
	w := M.GetMemory(addr)
	return w, false
}

// Read a word into channel buffer.
// Return true if fail, false if success.
func readBuffer(cUnit *chanDev, subChan *chanCtl) bool {
	var addr uint32

	if (subChan.ccwFlags & flagIDA) != 0 {
		addr = subChan.ccwIAddr
	} else {
		addr = subChan.ccwAddr
	}
	word, err := readFullWord(cUnit, subChan, addr)
	if err {
		subChan.chanByte = bufEnd
		return err
	}
	subChan.chanBuffer = word
	subChan.chanByte = uint8(addr & 3)
	return false
}

// Write channel buffer to memory.
// Return true if fail, false if success.
func writeBuffer(cUnit *chanDev, subChan *chanCtl) bool {
	var addr uint32

	if (subChan.ccwFlags & flagIDA) != 0 {
		addr = subChan.ccwIAddr
	} else {
		addr = subChan.ccwAddr
	}

	// Check if address valid
	addr &= M.AMASK
	if !M.CheckAddr(addr) {
		subChan.chanStatus |= statusPCHK
		subChan.chanByte = bufEnd
		subChan.chanDirty = false
		cUnit.irqPending = true
		IrqPending = true
		return true
	}

	// Check protection key
	if subChan.ccwKey != 0 {
		k := M.GetKey(addr)
		if (k & 0xf0) != subChan.ccwKey {
			subChan.chanStatus |= statusProt
			subChan.chanByte = bufEnd
			subChan.chanDirty = false
			cUnit.irqPending = true
			IrqPending = true
			return true
		}
	}

	// Write memory
	err := M.PutWord(addr, subChan.chanBuffer)
	subChan.chanByte = bufEmpty
	subChan.chanDirty = false
	return err
}
