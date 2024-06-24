package sys_channel

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

import (
	D "github.com/rcornwell/S370/emu/device"
	M "github.com/rcornwell/S370/emu/memory"
)

var chanUnit [MAX_CHAN]chanDev // Hold infomation about channels

var bmuxEnable bool

var nullDev D.Device

// Set whether Block multiplexer is enabled or not
func SetBMUXenable(enable bool) {
	bmuxEnable = enable
}

// Return type of channel
func GetType(devNum uint16) int {
	ch := (devNum >> 8)
	// Check if over max supported channels
	if ch > MAX_CHAN {
		return TypeUNA
	}
	cu := &chanUnit[ch&0xf]
	if !cu.enabled {
		return TypeUNA
	}
	return cu.chanType
}

// Process SIO instruction
func StartIO(devNum uint16) uint8 {
	ch := (devNum >> 8)
	if ch > MAX_CHAN {
		return 3
	}
	ch &= 0xf
	d := devNum & 0xff

	sc := findSubChannel(devNum)

	cu := &chanUnit[ch]
	// Check if channel disabled
	if !cu.enabled {
		return 3
	}

	// If no device or channel, return CC = 3
	if cu.devTab[d] == nullDev || sc == nil {
		return 3
	}

	// If pending status is for us, return it with status code
	if sc.devAddr == devNum && sc.chanStatus != 0 {
		storeCSW(sc)
		return 1
	}

	// If channel is active return cc = 2
	if sc.ccwCmd != 0 || (sc.ccwFlags&(chainCmd|chainData)) != 0 || sc.chanStatus != 0 {
		return 2
	}

	ds := cu.devStatus[d]
	if ds == SNS_DEVEND || ds == (SNS_DEVEND|SNS_CHNEND) {
		cu.devStatus[d] = 0
		ds = 0
	}

	// Check for any pending status for this device
	if ds != 0 {
		M.SetMemory(0x44, uint32(ds)<<24)
		M.SetMemory(0x40, 0)
		cu.devStatus[d] = 0
		return 1
	}

	status := uint16(cu.devTab[d].StartIO()) << 8
	if (status & statusBusy) != 0 {
		return 2
	}
	if status != 0 {
		M.PutWordMask(0x44, uint32(status)<<16, statusMask)
		return 1
	}

	// All ok, get caw address
	sc.chanStatus = 0
	sc.caw = M.GetMemory(0x48)
	sc.ccwKey = uint8(((sc.caw & keyMask) >> 24) & 0xff)
	sc.caw &= addrMask
	sc.devAddr = devNum
	sc.dev = cu.devTab[d]
	cu.devStatus[d] = 0

	if loadCCW(cu, sc, false) {
		M.SetMemoryMask(0x44, uint32(sc.chanStatus)<<16, statusMask)
		sc.chanStatus = 0
		sc.ccwCmd = 0
		sc.devAddr = NoDev
		sc.dev = nil
		cu.devStatus[d] = 0
		return 1
	}

	// If channel returned busy save CSW and return CC = 1
	if (sc.chanStatus & statusBusy) != 0 {
		M.SetMemoryMask(0x44, uint32(sc.chanStatus)<<16, statusMask)
		sc.chanStatus = 0
		sc.ccwCmd = 0
		sc.devAddr = NoDev
		sc.dev = nil
		cu.devStatus[d] = 0
		return 1
	}

	// If immediate command and not command chainting
	if (sc.chanStatus&statusChnEnd) != 0 && (sc.ccwFlags&chainCmd) == 0 {
		// If we also have data end write out fill CSW and mark subchannel free

		if (sc.chanStatus & statusDevEnd) != 0 {
			storeCSW(sc)
		} else {
			M.SetMemoryMask(0x44, uint32(sc.chanStatus)<<16, statusMask)
		}
		sc.ccwCmd = 0
		sc.devAddr = NoDev
		sc.dev = nil
		cu.devStatus[d] = 0
		sc.chanStatus = 0

		return 1
	}

	// If immediate command and chaining report status, but don't clear things
	if (sc.chanStatus&(statusChnEnd|statusDevEnd)) == statusChnEnd && (sc.ccwFlags&chainCmd) != 0 {
		M.SetMemoryMask(0x44, uint32(sc.chanStatus)<<16, statusMask)
		return 1
	}

	return 0
}

// Handle TIO instruction
func TestIO(devNum uint16) uint8 {
	ch := (devNum >> 8)
	if ch > MAX_CHAN {
		return 3
	}
	ch &= 0xf
	d := devNum & 0xff
	sch := findSubChannel(devNum)

	cu := &chanUnit[ch]
	// Check if channel disabled
	if !cu.enabled {
		return 3
	}

	// If no device or channel, return CC = 3
	if cu.devTab[d] == nullDev || sch == nil {
		return 3
	}

	// If any error pending save csw and return cc=1
	if (sch.chanStatus & errorStatus) != 0 {
		storeCSW(sch)
		return 1
	}

	// If channel active, return cc=2
	if sch.ccwCmd != 0 || (sch.ccwFlags&(chainCmd|chainData)) != 0 {
		return 2
	}

	// Device finished and channel status pending return it and cc=1
	if sch.ccwCmd == 0 && sch.chanStatus != 0 {
		storeCSW(sch)
		sch.devAddr = NoDev
		return 1
	}

	// Device has returned a status, store the csw and return cc=1
	if cu.devStatus[d] != 0 {
		M.SetMemory(0x40, 0)
		M.SetMemory(0x44, (uint32(cu.devStatus[d]) << 24))
		cu.devStatus[d] = 0
		return 1
	}

	// If error pending for another device on subchannel, return cc = 2
	if cu.irqPending {
		// Check if might be false
		for d := range uint16(256) {
			if cu.devStatus[d] != 0 {
				// Check if same subchannel
				if findSubChannel(d) == sch {
					cu.irqPending = true
					IrqPending = true
					return 2
				}
			}
		}
	}

	// Nothing pending, send a 0 command to device to get status
	status := uint16(cu.devTab[d].StartCmd(0)) << 8

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

// Handle HIO instruction
func HaltIO(devNum uint16) uint8 {
	ch := (devNum >> 8)
	if ch > MAX_CHAN {
		return 3
	}
	ch &= 0xf
	d := devNum & 0xff
	sch := findSubChannel(devNum)

	cu := &chanUnit[ch]
	// Check if channel disabled
	if !cu.enabled {
		return 3
	}

	// If no device or channel, return CC = 3
	if cu.devTab[d] == nullDev || sch == nil {
		return 3
	}

	// Generic halt I/O, tell device to stop end
	// If any error pending save csw and return cc = 1
	if (sch.chanStatus & errorStatus) != 0 {
		return 1
	}

	// If channel active, tell it to terminate
	if sch.ccwCmd != 0 {
		sch.chanByte = bufEmpty
		sch.ccwFlags &= ^(chainCmd | chainData)
	}

	// Executing a command, issue halt if available
	// Let device try to halt
	cc := cu.devTab[d].HaltIO()
	if cc == 1 {
		M.SetMemoryMask(0x44, (uint32(sch.chanStatus) << 16), statusMask)

	}
	return cc
}

// Handle TCH instruction
func TestChan(devNum uint16) uint8 {
	/* 360 Principles of Operation says, "Bit positions 21-23 of the
	   sum formed by the addition of the content of register B1 and the
	   content of the D1 field identify the channel to which the
	   instruction applies. Bit positions 24-31 of the address are ignored.
	   /67 Functional Characteristics do not mention any changes in basic or
	   extended control mode of the TCH instruction behaviour.
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
	c := (devNum >> 8) & 0xf
	if c > MAX_CHAN {
		return 3
	}

	cu := chanUnit[c]
	// Check if channel disabled
	if !cu.enabled {
		return 3
	}

	// Multiplexer channel always returns available
	if cu.chanType == TypeMux {
		return 0
	}

	// If Block Multiplexer channel operating in select mode
	if cu.chanType == TypeBMux && bmuxEnable {
		return 0
	}

	ch := &cu.subChans[0]
	// If channel is executing a command, return cc = 2
	if ch.ccwCmd != 0 || (ch.ccwFlags&(chainCmd|chainData)) != 0 {
		return 2
	}

	// If pending status, return 1
	if ch.chanStatus != 0 {
		return 1
	}

	return 0
}

// Read a byte from memory
func ChanReadByte(devNum uint16) (uint8, bool) {
	var sc *chanCtl

	// Return abort if no channel
	if sc = findSubChannel(devNum); sc == nil {
		return 0, true
	}
	// Channel has pending system status
	if (sc.chanStatus & 0x7f) != 0 {
		return 0, true
	}
	// Not read command
	if (sc.ccwCmd & 1) == 0 {
		return 0, true
	}
	// Check if transfer is finished
	if sc.chanByte == bufEnd {
		return 0, true
	}

	cu := &chanUnit[(devNum>>8)&0xf]
	// Check if count zero
	if sc.ccwCount == 0 {
		// If not data chaining, let device know there will be no
		// more data to come
		if (sc.ccwFlags & chainData) == 0 {
			sc.chanStatus |= statusChnEnd
			sc.chanByte = bufEnd
			return 0, true
		} else {
			// If chaining try and start next CCW
			if loadCCW(cu, sc, true) {
				return 0, true
			}
		}
	}

	// Read in next word if buffer is in empty status
	if sc.chanByte == bufEmpty {
		if readBuffer(cu, sc) {
			return 0, true
		}
		if nextAddress(cu, sc) {
			return 0, true
		}
	}

	// Return current byte
	sc.ccwCount--
	byte := uint8(sc.chanBuffer >> (8 * (3 - (sc.chanByte & 3))) & 0xff)
	sc.chanByte++
	// If count is zero and chaining load in new CCW
	if sc.ccwCount == 0 && (sc.ccwFlags&chainData) != 0 {
		// If chaining try and start next CCW
		if loadCCW(cu, sc, true) {
			// Return that this is last byte device will get
			return byte, true
		}
	}
	return byte, false
}

// Write a byte to memory
func ChanWriteByte(devNum uint16, data uint8) bool {
	var sc *chanCtl

	// Return abort if no channel
	if sc = findSubChannel(devNum); sc == nil {
		return true
	}
	// Channel has pending system status
	if (sc.chanStatus & 0x7f) != 0 {
		return true
	}
	// Not read command
	if (sc.ccwCmd & 1) != 0 {
		return true
	}
	// Check if transfer is finished
	if sc.chanByte == bufEnd {
		if (sc.ccwFlags & flagSLI) == 0 {
			sc.chanStatus |= statusLength
		}
		return true
	}
	cu := &chanUnit[(devNum>>8)&0xf]
	// Check if count zero
	if sc.ccwCount == 0 {
		if sc.chanDirty {
			if writeBuffer(cu, sc) {
				return true
			}
		}
		// If not data chaining, let device know there will be no
		// more data to come
		if (sc.ccwFlags & chainData) == 0 {
			sc.chanByte = bufEnd
			if (sc.ccwFlags & flagSLI) == 0 {
				sc.chanStatus |= statusLength
			}
			return true
		}
		// Otherwise try and grab next CCW
		if loadCCW(cu, sc, true) {
			return true
		}
	}

	// If we are skipping, just adjust count
	if (sc.ccwFlags & flagSkip) != 0 {
		sc.ccwCount--
		sc.chanByte = bufEmpty
		return nextAddress(cu, sc)
	}

	// Check if we need to save what we have
	if sc.chanByte == bufEmpty && sc.chanDirty {
		if writeBuffer(cu, sc) {
			return true
		}
		if nextAddress(cu, sc) {
			return true
		}
		sc.chanByte = bufEmpty
	}
	if sc.chanByte == bufEmpty {
		if readBuffer(cu, sc) {
			return true
		}

	}

	// Store it in buffer and adjust pointer
	sc.ccwCount--
	offset := 8 * (sc.chanByte & 3)
	mask := uint32(0xff000000 >> offset)
	sc.chanBuffer &= ^mask
	sc.chanBuffer |= uint32(data) << (24 - offset)
	if (sc.ccwCmd & 0xf) == CMD_RDBWD {
		if (sc.chanByte & 3) != 0 {
			sc.chanByte--
		} else {
			sc.chanByte = bufEmpty
		}
	} else {
		sc.chanByte++
	}
	sc.chanDirty = true
	// If count is zero and chaining load in new CCW
	if sc.ccwCount == 0 && (sc.ccwFlags&chainData) != 0 {
		// Flush buffer
		if sc.chanDirty && writeBuffer(cu, sc) {
			return true
		}
		// If chaining try and start next CCW
		if loadCCW(cu, sc, true) {
			// Return that this is last byte device will get
			return true
		}
	}
	return false
}

// Compute address of next byte to read/write
func nextAddress(cu *chanDev, sc *chanCtl) bool {
	if (sc.ccwFlags & flagIDA) != 0 {
		if (sc.ccwCmd & 0xf) == CMD_RDBWD {
			sc.ccwIAddr--
			if (sc.ccwIAddr & 0x7ff) == 0x7ff {
				sc.ccwAddr += 4
				if t, err := readFullWord(cu, sc, sc.ccwAddr); err {
					return true
				} else {
					sc.ccwIAddr = t & M.AMASK
				}
			}
		} else {
			sc.ccwIAddr++
			if (sc.ccwIAddr & 0x7ff) == 0x000 {
				sc.ccwAddr += 4
				if t, err := readFullWord(cu, sc, sc.ccwAddr); err {
					return true
				} else {
					sc.ccwIAddr = t & M.AMASK
				}
			}
		}
		sc.chanByte = uint8(sc.ccwIAddr & 3)
	} else {
		if (sc.ccwCmd & 0xf) == CMD_RDBWD {
			sc.ccwAddr -= 1 + (sc.ccwAddr & 0x3)
		} else {
			sc.ccwAddr += 4 - (sc.ccwAddr & 0x3)
		}
		sc.chanByte = uint8(sc.ccwAddr & 3)
	}
	return false
}

// Signal end of transfer by device
func ChanEnd(devNum uint16, flags uint8) {
	var sc *chanCtl

	// Return abort if no channel
	if sc = findSubChannel(devNum); sc == nil {
		return
	}

	cu := &chanUnit[(devNum>>8)&0xf]
	if sc.chanDirty {
		_ = writeBuffer(cu, sc)
	}
	sc.chanStatus |= statusChnEnd
	sc.chanStatus |= uint16(flags) << 8
	sc.ccwCmd = 0

	// If count not zero and not suppressing length, report error
	if sc.ccwCount != 0 && (sc.ccwFlags&flagSLI) == 0 {
		sc.chanStatus |= statusLength
		sc.ccwFlags = 0
	}

	// If count not zero and not suppressing length, report error
	if sc.ccwCount != 0 && (sc.ccwFlags&(chainData|flagSLI)) == (chainData|flagSLI) {
		sc.chanStatus |= statusLength
	}

	if (flags & (SNS_ATTN | SNS_UNITCHK | SNS_UNITEXP)) != 0 {
		sc.ccwFlags = 0
	}

	if (flags & SNS_DEVEND) != 0 {
		sc.ccwFlags &= ^(chainData | flagSLI)
	}

	cu.irqPending = true
	IrqPending = true

}

// A device wishes to inform the CPU it needs some service.
func SetDevAttn(devNum uint16, flags uint8) {
	var ch *chanCtl

	if ch = findSubChannel(devNum); ch == nil {
		return
	}
	cu := &chanUnit[(devNum>>8)&0xf]
	// Check if chain being held
	if ch.devAddr == devNum && ch.chainFlg && (flags&SNS_DEVEND) != 0 {
		ch.chanStatus |= uint16(flags) << 8
	} else {
		// Check if Device is currently on channel
		if ch.devAddr == devNum && (flags&SNS_DEVEND) != 0 &&
			((ch.chanStatus&statusChnEnd) != 0 || ch.ccwCmd != 0) {
			ch.chanStatus |= uint16(flags) << 8
			ch.ccwCmd = 0
		} else { // Device reporting status change
			cu.devStatus[devNum&0xff] = flags
		}
	}
	cu.irqPending = true
	IrqPending = true
}

// Scan all channels and see if one is ready to start or has interrupt pending
func ChanScan(mask uint16, irqEnb bool) uint16 {
	var sc *chanCtl
	var cu *chanDev
	var imask uint16

	sc = nil
	// Quick exit if no pending IRQ's
	if !IrqPending {
		return NoDev
	}

	// Clear pending flag
	IrqPending = false
	pendDev := NoDev // Device with Pending interrupt
	// Start with channel 0 and work through all channels
	for i := range MAX_CHAN {
		cu = &chanUnit[i]

		if !cu.enabled { //&& !cu.irqPending {
			continue
		}
		// Mask for this channel
		imask = 0x8000 >> i
		nchan := 1
		if cu.chanType == TypeBMux && bmuxEnable {
			nchan = 32
		}
		if cu.chanType == TypeMux {
			nchan = cu.numSubChan
		}
		// Scan all subchannels on this channel
		for j := range nchan {
			sc = &cu.subChans[j]
			if sc.devAddr == NoDev {
				continue
			}

			// Check if PCI pending
			if irqEnb && (imask&mask) != 0 && (sc.chanStatus&statusPCI) != 0 {
				pendDev = sc.devAddr
				break
			}

			// If device has hard error, store CSW and end.
			if irqEnb && (imask&mask) != 0 && (sc.chanStatus&0xff) != 0 {
				pendDev = sc.devAddr
				break
			}

			// If chaining and device end continue
			if sc.chainFlg && (sc.chanStatus&statusDevEnd) != 0 {
				// Restart command that was flagged as an issue
				_ = loadCCW(cu, sc, true)
			}

			if (sc.chanStatus & statusChnEnd) != 0 {
				// Grab another command if command chaining in effect
				if (sc.ccwFlags & chainCmd) != 0 {
					// If channel end, check if we should continue
					_ = loadCCW(cu, sc, true)
				} else if irqEnb || Loading != NoDev {
					// Disconnect from device
					if (imask&mask) != 0 || Loading != NoDev {
						pendDev = sc.devAddr
						break
					}
				}
			}
		}
	}

	// Only return loading unit on loading
	if Loading != NoDev && Loading != pendDev {
		return NoDev
	}

	// See if we can post an IRQ
	if pendDev != NoDev {
		// Set to scan next time
		IrqPending = true
		sc = findSubChannel(pendDev)
		if Loading == pendDev {
			sc.chanStatus = 0
			cu.devStatus[pendDev&0xff] = 0
			return pendDev
		}
		if Loading == NoDev {
			storeCSW(sc)
			cu.devStatus[pendDev&0xff] = 0
			return pendDev
		}
	} else {
		if irqEnb {
			// If interrupts are wanted, check for pending device status
			for i := range MAX_CHAN {
				cu = &chanUnit[i]
				// Mask for this channel
				imask := uint16(0x8000 >> i)
				if !cu.irqPending || (imask&mask) == 0 {
					continue
				}
				cu.irqPending = false
				for j := range 256 {
					// Look for device with pending status
					if cu.devStatus[j] != 0 {
						cu.irqPending = true
						IrqPending = true
						M.SetMemory(0x44, uint32(cu.devStatus[j])<<24)
						M.SetMemory(0x40, 0)
						cu.devStatus[j] = 0
						return uint16((i << 8)) | uint16(j)
					}
				}

			}
		}
	}
	// No pending device
	return NoDev
}

// IPL a device.
func BootDevice(dev_num uint16) bool {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return true
	}
	ch &= 0xf
	d := dev_num & 0xff

	sc := findSubChannel(dev_num)

	cu := &chanUnit[ch]
	// Check if channel disabled
	if !cu.enabled {
		return true
	}

	// If no device or channel, return CC = 3
	if cu.devTab[d] == nil || sc == nil {
		return true
	}
	status := uint16(cu.devTab[d].StartIO()) << 8
	if status != 0 {
		return true
	}

	sc.chanStatus = 0
	sc.dev = cu.devTab[d]
	sc.caw = 0x8
	sc.devAddr = dev_num
	sc.ccwCount = 24
	sc.ccwFlags = chainCmd | flagSLI
	sc.ccwAddr = 0
	sc.ccwKey = 0
	sc.chanByte = bufEmpty
	sc.chanDirty = false

	sc.chanStatus |= uint16(sc.dev.StartCmd(sc.ccwCmd)) << 8

	// Check if any errors from initial command
	if (sc.chanStatus & (statusAttn | statusCheck | statusExcept)) != 0 {
		sc.ccwCmd = 0
		sc.ccwFlags = 0
		return true
	}
	Loading = dev_num
	return false
}

// Add a device at given address
func AddDevice(dev D.Device, dev_num uint16) bool {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return true
	}
	ch &= 0xf
	d := dev_num & 0xff

	cu := &chanUnit[ch]
	// Check if channel disabled
	if !cu.enabled || cu.devTab[d] != nil {
		return true
	}

	cu.devTab[d] = dev
	return false
}

// Delete a device at a given address
func DelDevice(dev_num uint16) {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return
	}
	ch &= 0xf
	d := dev_num & 0xff
	cu := &chanUnit[ch]
	cu.devTab[d] = nil
}

// Enable a channel of a given type
func AddChannel(chan_num uint16, ty int, subchan int) {
	if chan_num <= MAX_CHAN {
		cu := &chanUnit[chan_num]
		cu.enabled = true
		cu.chanType = ty
		switch ty {
		case TypeDis:
			cu.enabled = false
		case TypeSel:
			cu.numSubChan = 1
		case TypeMux:
			cu.numSubChan = subchan
		case TypeBMux:
			cu.numSubChan = 32
		}
	}
}

// Initialize all channels and clear any device assignments
func InitializeChannels() {
	//var d *D.Device = nil
	for i := range MAX_CHAN {
		cu := &chanUnit[i]
		cu.enabled = false
		cu.irqPending = false
		cu.chanType = 0
		cu.numSubChan = 0
		for j := range 256 {
			cu.devTab[j] = nullDev
			cu.devStatus[j] = 0
			cu.subChans[j].devAddr = NoDev
		}
	}
}

/* channel:
    subchannels = 128
    0 - 7       0x80-0xff
   8 - 127     0x00-0x7f
   128 - +6    0x1xx - 0x6xx
*/

// Look up device to find subchannel device is on
func findSubChannel(devNum uint16) *chanCtl {
	ch := (devNum >> 8) & 0xf
	if ch > MAX_CHAN {
		return nil
	}
	device := int(devNum & 0xff)
	switch chanUnit[ch].chanType {
	case TypeSel:
		return &chanUnit[ch].subChans[0]
	case TypeBMux:
		if bmuxEnable {
			d := (devNum >> 3) & 0x1f
			return &chanUnit[ch].subChans[d]
		}
		return &chanUnit[ch].subChans[0]
	case TypeMux:
		if device >= chanUnit[ch].numSubChan {
			if device < 128 { // All shared devices over subchannels
				return nil
			}
			device = (device >> 4) & 0x7
		}
		return &chanUnit[ch].subChans[device]
	}
	return nil
}

// Save full csw.
func storeCSW(ch *chanCtl) {
	M.SetMemory(0x40, (uint32(ch.ccwKey)<<24)|ch.caw)
	M.SetMemory(0x44, uint32(ch.ccwCount)|(uint32(ch.chanStatus)<<16))
	if (ch.chanStatus & statusPCI) != 0 {
		ch.chanStatus &= ^statusPCI
	} else {
		ch.chanStatus = 0
	}
	ch.ccwFlags &= ^flagPCI
}

// Load in the next CCW, return true if failure, false if success.
func loadCCW(cu *chanDev, sc *chanCtl, ticOk bool) bool {
	var word uint32
	var err bool
	var cmdFlag bool = false
	var chain bool

loop:
	// If last chain, start command
	if sc.chainFlg && (sc.ccwFlags&chainData) == 0 {
		chain = true
		sc.chainFlg = false
		cmdFlag = true
	} else {

		// Abort if ccw not on double word boundry
		if (sc.caw & 0x7) != 0 {
			sc.chanStatus = statusPCHK
			return true
		}

		// Abort if we have pending errors
		if (sc.chanStatus & 0x7F) != 0 {
			return true
		}

		// Remember if we were chainging
		chain = (sc.ccwFlags & chainCmd) != 0

		// Check if we have status modifier set
		if (sc.chanStatus & statusSMS) != 0 {
			sc.caw += 8
			sc.caw &= addrMask
			sc.chanStatus &= ^statusSMS
		}

		// Read in next CCW
		word, err = readFullWord(cu, sc, sc.caw)
		if err {
			return true
		}

		// Next word
		sc.caw += 4
		sc.caw &= addrMask

		// TIC can't follow TIC nor bt first in chain
		cmd := uint8((word & cmdMask) >> 24)
		if cmd == CMD_TIC {
			// Pretend to fetch next word.
			sc.caw += 4
			sc.caw &= addrMask
			sc.ccwCmd = 0
			sc.ccwFlags = 0
			if ticOk {
				sc.caw = word & addrMask
				ticOk = false
				goto loop
			}
			sc.chanStatus = statusPCHK
			cu.irqPending = true
			IrqPending = true
			return true
		}

		// Check if not chaining data
		if (sc.ccwFlags & chainData) == 0 {
			sc.ccwCmd = cmd
			cmdFlag = true
		}

		// Set up for this command
		sc.ccwAddr = word & addrMask
		word, err = readFullWord(cu, sc, sc.caw)
		if err {
			return true
		}
		sc.caw += 4
		sc.caw &= addrMask
		sc.ccwCount = uint16(word & countMask)

		// Copy SLI indicator in CD command
		if (sc.ccwFlags & (chainData | flagSLI)) == (chainData | flagSLI) {
			word |= uint32(flagSLI) << 16
		}
		sc.ccwFlags = uint16(word>>16) & 0xff00
		sc.chanByte = bufEmpty

		// Check if invalid count
		if sc.ccwCount == 0 {
			sc.chanStatus = statusPCHK
			sc.ccwCmd = 0
			cu.irqPending = true
			IrqPending = true
			return true
		}

		// Handle IDA
		if (sc.ccwFlags & flagIDA) != 0 {
			word, err = readFullWord(cu, sc, sc.ccwAddr)
			if err {
				return true
			}
			sc.ccwIAddr = word & addrMask
		}
	}

	// If command pending start it.
	if cmdFlag {
		// Check if invalid command
		if (sc.ccwCmd & 0xf) == 0 {
			sc.chanStatus |= statusPCHK
			sc.ccwCmd = 0
			cu.irqPending = true
			IrqPending = true
			return true
		}

		if sc.dev == nil {
			return true
		}

		sc.chanByte = bufEmpty
		status := uint16(sc.dev.StartCmd(sc.ccwCmd)) << 8

		// If device is busy, check if last was CC, then mark pending
		if (status & statusBusy) != 0 {
			if chain {
				sc.chainFlg = true
			}
			return false
		}
		sc.chanStatus &= 0xff
		sc.chanStatus |= status
		// Check if any errors from initial command
		if (sc.chanStatus & (statusAttn | statusCheck | statusExcept)) != 0 {
			sc.ccwCmd = 0
			sc.ccwFlags = 0
			cu.devStatus[sc.devAddr&0xff] = uint8((sc.chanStatus >> 8) & 0xff)
			cu.irqPending = true
			IrqPending = true
			return true
		}

		// Check if immediate channel end
		if (sc.chanStatus & statusChnEnd) != 0 {
			sc.ccwFlags |= flagSLI // Force SLI for immediate command
			if (sc.chanStatus & statusChnEnd) != 0 {
				sc.ccwCmd = 0
				cu.irqPending = true
				IrqPending = true
			}
		}
	}

	if (sc.ccwFlags & flagPCI) != 0 {
		sc.chanStatus |= statusPCI
		cu.irqPending = true
		IrqPending = true
	}
	return false
}

// Read a fill word from memory
// Return true if fail and false if success
func readFullWord(cu *chanDev, sc *chanCtl, addr uint32) (uint32, bool) {
	if !M.CheckAddr(addr) {
		sc.chanStatus |= statusPCHK
		cu.irqPending = true
		IrqPending = true
		return 0, true
	}
	if sc.ccwKey != 0 {
		k := M.GetKey(addr)
		if (k&0x8) != 0 && (k&0xf0) != sc.ccwKey {
			sc.chanStatus |= statusProt
			cu.irqPending = true
			IrqPending = true
			return 0, true
		}
	}
	w := M.GetMemory(addr)
	return w, false
}

// Read a word into channel buffer
// Return true if fail, false if success
func readBuffer(cu *chanDev, sc *chanCtl) bool {
	var addr uint32

	if (sc.ccwFlags & flagIDA) != 0 {
		addr = sc.ccwIAddr
	} else {
		addr = sc.ccwAddr
	}
	if word, err := readFullWord(cu, sc, addr); err {
		sc.chanByte = bufEnd
		return err
	} else {
		sc.chanBuffer = word
		sc.chanByte = uint8(addr & 3)
	}
	return false
}

// Write channel buffer to memory
// Return true if fail, false if success
func writeBuffer(cu *chanDev, sc *chanCtl) bool {
	var addr uint32

	if (sc.ccwFlags & flagIDA) != 0 {
		addr = sc.ccwIAddr
	} else {
		addr = sc.ccwAddr
	}

	// Check if address valid
	addr &= M.AMASK
	if !M.CheckAddr(addr) {
		sc.chanStatus |= statusPCHK
		sc.chanByte = bufEnd
		sc.chanDirty = false
		cu.irqPending = true
		IrqPending = true
		return true
	}

	// Check protection key
	if sc.ccwKey != 0 {
		k := M.GetKey(addr)
		if (k & 0xf0) != sc.ccwKey {
			sc.chanStatus |= statusProt
			sc.chanByte = bufEnd
			sc.chanDirty = false
			cu.irqPending = true
			IrqPending = true
			return true
		}
	}

	// Write memory
	err := M.PutWord(addr, sc.chanBuffer)
	sc.chanByte = bufEmpty
	sc.chanDirty = false
	return err
}
