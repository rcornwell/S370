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
	D "github.com/rcornwell/S370/internal/device"
	M "github.com/rcornwell/S370/internal/memory"
)

var ch_unit [MAX_CHAN]chan_unit // Hold infomation about channels

var bmux_enable bool

var null_dev D.Device

// Set whether Block multiplexer is enabled or not
func SetBMUXenable(enable bool) {
	bmux_enable = enable
}

// Return type of channel
func GetType(dev_num uint16) int {
	ch := (dev_num >> 8)
	// Check if over max supported channels
	if ch > MAX_CHAN {
		return TYPE_UNA
	}
	cu := &ch_unit[ch&0xf]
	if !cu.enabled {
		return TYPE_UNA
	}
	return cu.ch_type
}

// Process SIO instruction
func StartIO(dev_num uint16) uint8 {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return 3
	}
	ch &= 0xf
	d := dev_num & 0xff

	sc := find_subchannel(dev_num)

	cu := &ch_unit[ch]
	// Check if channel disabled
	if !cu.enabled {
		return 3
	}

	// If no device or channel, return CC = 3
	if cu.dev_tab[d] == null_dev || sc == nil {
		return 3
	}

	// If pending status is for us, return it with status code
	if sc.daddr == dev_num && sc.chan_status != 0 {
		store_csw(sc)
		return 1
	}

	// If channel is active return cc = 2
	if sc.ccw_cmd != 0 || (sc.ccw_flags&(FLAG_CC|FLAG_CD)) != 0 || sc.chan_status != 0 {
		return 2
	}

	ds := cu.dev_status[d]
	if ds == SNS_DEVEND || ds == (SNS_DEVEND|SNS_CHNEND) {
		cu.dev_status[d] = 0
		ds = 0
	}

	// Check for any pending status for this device
	if ds != 0 {
		M.SetMemory(0x44, uint32(ds)<<24)
		M.SetMemory(0x40, 0)
		cu.dev_status[d] = 0
		return 1
	}

	status := uint16(cu.dev_tab[d].Start_IO()) << 8
	if (status & STATUS_BUSY) != 0 {
		return 2
	}
	if status != 0 {
		M.PutWordMask(0x44, uint32(status)<<16, M.UMASK)
		return 1
	}

	// All ok, get caw address
	sc.chan_status = 0
	sc.caw = M.GetMemory(0x48)
	sc.ccw_key = uint8(((sc.caw & M.PMASK) >> 24) & 0xff)
	sc.caw &= M.AMASK
	sc.daddr = dev_num
	sc.dev = cu.dev_tab[d]
	cu.dev_status[d] = 0

	if load_ccw(cu, sc, false) {
		M.SetMemoryMask(0x44, uint32(sc.chan_status)<<16, M.UMASK)
		sc.chan_status = 0
		sc.ccw_cmd = 0
		sc.daddr = NO_DEV
		sc.dev = nil
		cu.dev_status[d] = 0
		return 1
	}

	// If channel returned busy save CSW and return CC = 1
	if (sc.chan_status & STATUS_BUSY) != 0 {
		M.SetMemory(0x40, 0)
		M.SetMemory(0x44, uint32(sc.chan_status)<<16)
		sc.chan_status = 0
		sc.ccw_cmd = 0
		sc.daddr = NO_DEV
		sc.dev = nil
		cu.dev_status[d] = 0
		return 1
	}

	return 0
}

// Handle TIO instruction
func TestIO(dev_num uint16) uint8 {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return 3
	}
	ch &= 0xf
	d := dev_num & 0xff
	sch := find_subchannel(dev_num)

	cu := &ch_unit[ch]
	// Check if channel disabled
	if !cu.enabled {
		return 3
	}

	// If no device or channel, return CC = 3
	if cu.dev_tab[d] == null_dev || sch == nil {
		return 3
	}

	// If any error pending save csw and return cc=1
	if (sch.chan_status & ERROR_STATUS) != 0 {
		store_csw(sch)
		return 1
	}

	// If channel active, return cc=2
	if sch.ccw_cmd != 0 || (sch.ccw_flags&(FLAG_CC|FLAG_CD)) != 0 {
		return 2
	}

	// Device finished and channel status pending return it and cc=1
	if sch.ccw_cmd == 0 && sch.chan_status != 0 {
		store_csw(sch)
		sch.daddr = NO_DEV
		return 1
	}

	// Device has returned a status, store the csw and return cc=1
	if cu.dev_status[d] != 0 {
		M.SetMemory(0x40, 0)
		M.SetMemory(0x44, (uint32(cu.dev_status[d]) << 24))
		cu.dev_status[d] = 0
		return 1
	}

	// If error pending for another device on subchannel, return cc = 2
	if cu.irq_pend {
		// Check if might be false
		for d := range uint16(256) {
			if cu.dev_status[d] != 0 {
				// Check if same subchannel
				if find_subchannel(d) == sch {
					cu.irq_pend = true
					Irq_pending = true
					return 2
				}
			}
		}
	}

	// Nothing pending, send a 0 command to device to get status
	status := uint16(cu.dev_tab[d].Start_cmd(0)) << 8

	// If we get a error, save csw and return cc = 1
	if (status & ERROR_STATUS) != 0 {
		M.SetMemoryMask(0x44, uint32(status)<<16, M.UMASK)
		return 1
	}

	// Check if device BUSY
	if (status & STATUS_BUSY) != 0 {
		return 2
	}

	// Everything ok, return cc = 0
	return 0
}

// Handle HIO instruction
func HaltioIO(dev_num uint16) uint8 {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return 3
	}
	ch &= 0xf
	d := dev_num & 0xff
	sch := find_subchannel(dev_num)

	cu := &ch_unit[ch]
	// Check if channel disabled
	if !cu.enabled {
		return 3
	}

	// If no device or channel, return CC = 3
	if cu.dev_tab[d] == null_dev || sch == nil {
		return 3
	}

	// Generic halt I/O, tell device to stop end
	// If any error pending save csw and return cc = 1
	if (sch.chan_status & ERROR_STATUS) != 0 {
		return 1
	}

	// If channel active, tell it to terminate
	if sch.ccw_cmd != 0 {
		sch.chan_byte = BUFF_CHNEND
		sch.ccw_flags &= ^(FLAG_CC | FLAG_CD)
	}

	// Executing a command, issue halt if available
	// Let device try to halt
	cc := cu.dev_tab[d].Halt_IO()
	if cc == 1 {
		M.SetMemoryMask(0x44, (uint32(sch.chan_status) << 16), M.UMASK)

	}
	return cc
}

// Handle TCH instruction
func TestChan(dev_num uint16) uint8 {
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
	c := (dev_num >> 8) & 0xf
	if c > MAX_CHAN {
		return 3
	}

	cu := ch_unit[c]
	// Check if channel disabled
	if !cu.enabled {
		return 3
	}

	// Multiplexer channel always returns available
	if cu.ch_type == TYPE_MUX {
		return 0
	}

	// If Block Multiplexer channel operating in select mode
	if cu.ch_type == TYPE_BMUX && bmux_enable {
		return 0
	}

	ch := &cu.subchan[0]
	// If channel is executing a command, return cc = 2
	if ch.ccw_cmd != 0 || (ch.ccw_flags&(FLAG_CC|FLAG_CD)) != 0 {
		return 2
	}

	// If pending status, return 1
	if ch.chan_status != 0 {
		return 1
	}

	return 0
}

// Read a byte from memory
func Chan_read_byte(dev_num uint16) (uint8, bool) {
	var sc *chan_ctl

	// Return abort if no channel
	if sc = find_subchannel(dev_num); sc == nil {
		return 0, true
	}
	// Channel has pending system status
	if (sc.chan_status & 0x7f) != 0 {
		return 0, true
	}
	// Not read command
	if (sc.ccw_cmd & 1) == 0 {
		return 0, true
	}
	// Check if transfer is finished
	if sc.chan_byte == BUFF_CHNEND {
		return 0, true
	}

	cu := &ch_unit[(dev_num>>8)&0xf]
	// Check if count zero
	if sc.ccw_count == 0 {
		// If not data chaining, let device know there will be no
		// more data to come
		if (sc.ccw_flags & FLAG_CD) == 0 {
			sc.chan_status |= STATUS_CEND
			sc.chan_byte = BUFF_CHNEND
			return 0, true
		} else {
			// If chaining try and start next CCW
			if load_ccw(cu, sc, true) {
				return 0, true
			}
		}
	}

	// Read in next word if buffer is in empty status
	if sc.chan_byte == BUFF_EMPTY {
		if readbuff(cu, sc) {
			return 0, true
		}
		if next_byte_address(cu, sc) {
			return 0, true
		}
	}

	// Return current byte
	sc.ccw_count--
	byte := uint8(sc.chan_buf >> (8 * (3 - (sc.chan_byte & 3))) & 0xff)
	sc.chan_byte++
	// If count is zero and chaining load in new CCW
	if sc.ccw_count == 0 && (sc.ccw_flags&FLAG_CD) != 0 {
		// If chaining try and start next CCW
		if load_ccw(cu, sc, true) {
			// Return that this is last byte device will get
			return byte, true
		}
	}
	return byte, false
}

// Write a byte to memory
func Chan_write_byte(dev_num uint16, data uint8) bool {
	var sc *chan_ctl

	// Return abort if no channel
	if sc = find_subchannel(dev_num); sc == nil {
		return true
	}
	// Channel has pending system status
	if (sc.chan_status & 0x7f) != 0 {
		return true
	}
	// Not read command
	if (sc.ccw_cmd & 1) != 0 {
		return true
	}
	// Check if transfer is finished
	if sc.chan_byte == BUFF_CHNEND {
		if (sc.ccw_flags & FLAG_SLI) == 0 {
			sc.chan_status |= STATUS_LENGTH
		}
		return true
	}
	cu := &ch_unit[(dev_num>>8)&0xf]
	// Check if count zero
	if sc.ccw_count == 0 {
		if sc.chan_dirty {
			if writebuff(cu, sc) {
				return true
			}
		}
		// If not data chaining, let device know there will be no
		// more data to come
		if (sc.ccw_flags & FLAG_CD) == 0 {
			sc.chan_byte = BUFF_CHNEND
			if (sc.ccw_flags & FLAG_SLI) == 0 {
				sc.chan_status |= STATUS_LENGTH
			}
			return true
		}
		// Otherwise try and grab next CCW
		if load_ccw(cu, sc, true) {
			return true
		}
	}

	// If we are skipping, just adjust count
	if (sc.ccw_flags & FLAG_SKIP) != 0 {
		sc.ccw_count--
		sc.chan_byte = BUFF_EMPTY
		return next_byte_address(cu, sc)
	}

	// Check if we need to save what we have
	if sc.chan_byte == BUFF_EMPTY && sc.chan_dirty {
		if writebuff(cu, sc) {
			return true
		}
		if next_byte_address(cu, sc) {
			return true
		}
		sc.chan_byte = BUFF_EMPTY
	}
	if sc.chan_byte == BUFF_EMPTY {
		if readbuff(cu, sc) {
			return true
		}

	}

	// Store it in buffer and adjust pointer
	sc.ccw_count--
	offset := 8 * (sc.chan_byte & 3)
	mask := uint32(0xff000000 >> offset)
	sc.chan_buf &= ^mask
	sc.chan_buf |= uint32(data) << (24 - offset)
	if (sc.ccw_cmd & 0xf) == CMD_RDBWD {
		if (sc.chan_byte & 3) != 0 {
			sc.chan_byte--
		} else {
			sc.chan_byte = BUFF_EMPTY
		}
	} else {
		sc.chan_byte++
	}
	sc.chan_dirty = true
	// If count is zero and chaining load in new CCW
	if sc.ccw_count == 0 && (sc.ccw_flags&FLAG_CD) != 0 {
		// Flush buffer
		if sc.chan_dirty && writebuff(cu, sc) {
			return true
		}
		// If chaining try and start next CCW
		if load_ccw(cu, sc, true) {
			// Return that this is last byte device will get
			return true
		}
	}
	return false
}

// Compute address of next byte to read/write
func next_byte_address(cu *chan_unit, sc *chan_ctl) bool {
	if (sc.ccw_flags & FLAG_IDA) != 0 {
		var err bool
		var t uint32
		if (sc.ccw_cmd & 0xf) == CMD_RDBWD {
			sc.ccw_iaddr--
			if (sc.ccw_iaddr & 0x7ff) == 0x7ff {
				sc.ccw_addr += 4
				if t, err = readfull(cu, sc, sc.ccw_addr); err {
					return true
				}
				sc.ccw_iaddr = t & M.AMASK
			}
		} else {
			sc.ccw_iaddr++
			if (sc.ccw_iaddr & 0x7ff) == 0x000 {

				sc.ccw_addr += 4
				if t, err = readfull(cu, sc, sc.ccw_addr); err {
					return true
				}
				sc.ccw_iaddr = t & M.AMASK
			}
		}
		sc.chan_byte = uint8(sc.ccw_iaddr & 3)
	} else {
		if (sc.ccw_cmd & 0xf) == CMD_RDBWD {
			sc.ccw_addr -= 1 + (sc.ccw_addr & 0x3)
		} else {
			sc.ccw_addr += 4 - (sc.ccw_addr & 0x3)
		}
		sc.chan_byte = uint8(sc.ccw_addr & 3)
	}
	return false
}

// Signal end of transfer by device
func Chan_end(dev_num uint16, flags uint8) {
	var sc *chan_ctl

	// Return abort if no channel
	if sc = find_subchannel(dev_num); sc == nil {
		return
	}

	cu := &ch_unit[(dev_num>>8)&0xf]
	if sc.chan_dirty {
		_ = writebuff(cu, sc)
	}
	sc.chan_status |= STATUS_CEND
	sc.chan_status |= uint16(flags) << 8
	sc.ccw_cmd = 0

	// If count not zero and not suppressing length, report error
	if sc.ccw_count != 0 && (sc.ccw_flags&FLAG_SLI) == 0 {
		sc.chan_status |= STATUS_LENGTH
		sc.ccw_flags = 0
	}

	// If count not zero and not suppressing length, report error
	if sc.ccw_count != 0 && (sc.ccw_flags&(FLAG_CD|FLAG_SLI)) == (FLAG_CD|FLAG_SLI) {
		sc.chan_status |= STATUS_LENGTH
	}

	if (flags & (SNS_ATTN | SNS_UNITCHK | SNS_UNITEXP)) != 0 {
		sc.ccw_flags = 0
	}

	if (flags & SNS_DEVEND) != 0 {
		sc.ccw_flags &= ^(FLAG_CD | FLAG_SLI)
	}

	cu.irq_pend = true
	Irq_pending = true

}

// A device wishes to inform the CPU it needs some service.
func Set_devattn(dev_num uint16, flags uint8) {
	var ch *chan_ctl

	if ch = find_subchannel(dev_num); ch == nil {
		return
	}

	c := (dev_num >> 8) & 0xf
	cu := &ch_unit[c]
	// Check if chain being held
	if ch.daddr == dev_num && ch.chain_flg && (flags&SNS_DEVEND) != 0 {
		ch.chan_status |= uint16(flags) << 8
	} else {
		// Check if Device is currently on channel
		if ch.daddr == dev_num && (ch.chan_status&STATUS_CEND) != 0 && (flags&SNS_DEVEND) != 0 {
			ch.chan_status |= uint16(flags) << 8
		} else { // Device reporting status change
			cu.dev_status[dev_num&0xff] = flags
		}
	}

	cu.irq_pend = true
	Irq_pending = true
}

// Scan all channels and see if one is ready to start or has interrupt pending
func Chan_scan(mask uint16, irq_en bool) uint16 {
	var sc *chan_ctl
	var c *chan_unit
	var imask uint16

	sc = nil
	// Quick exit if no pending IRQ's
	if !Irq_pending {
		return NO_DEV
	}

	// Clear pending flag
	Irq_pending = false
	pend := NO_DEV
	// Start with channel 0 and work through all channels
	for i := range MAX_CHAN {
		c = &ch_unit[i]

		if !c.enabled || !c.irq_pend {
			continue
		}
		// Mask for this channel
		imask = 0x8000 >> i
		nchan := 1
		if c.ch_type == TYPE_BMUX && bmux_enable {
			nchan = 32
		}
		if c.ch_type == TYPE_MUX {
			nchan = c.nsubchan
		}
		// Scan all subchannels on this channel
		for j := range nchan {
			sc = &c.subchan[j]
			if sc.daddr == NO_DEV {
				continue
			}

			// Check if PCI pending
			if irq_en && (sc.chan_status&STATUS_PCI) != 0 {
				if (imask & mask) != 0 {
					pend = sc.daddr
					break
				}
			}

			// If chaining and device end continue
			if sc.chain_flg && (sc.chan_status&STATUS_DEND) != 0 {
				// Restart command that was flagged as an issue
				_ = load_ccw(c, sc, true)
			}

			// If channel end, check if we should continue
			if (sc.chan_status & STATUS_DEND) != 0 {
				// Grab another command if command chaining in effect
				if (sc.ccw_flags & FLAG_CC) != 0 {
					_ = load_ccw(c, sc, true)
					if (sc.ccw_flags&FLAG_CC) != 0 || (sc.chan_status&STATUS_DEND) == 0 {
						continue
					}
				}
				if irq_en || Loading != NO_DEV {
					// Disconnect from device
					if (imask&mask) != 0 || Loading != NO_DEV {
						pend = sc.daddr
						break
					}
				}
			}
		}
		if pend != NO_DEV {
			break
		}
	}

	// Only return loading unit on loading
	if Loading != NO_DEV && Loading != pend {
		return NO_DEV
	}

	// See if we can post an IRQ
	if pend != NO_DEV {
		// Set to scan next time
		Irq_pending = true
		//		sc = find_subchannel(pend)
		if Loading == pend {
			sc.chan_status = 0
			c.dev_status[pend&0xff] = 0
			return pend
		}
		if Loading == NO_DEV {
			store_csw(sc)
			c.dev_status[pend&0xff] = 0
			return pend
		}
	} else {
		if irq_en {
			// If interrupts are wanted, check for pending device status
			for i := range MAX_CHAN {
				c = &ch_unit[i]
				// Mask for this channel
				imask := uint16(0x8000 >> i)
				if !c.irq_pend || (imask&mask) == 0 {
					continue
				}
				for j := range 256 {
					// Look for device with pending status
					if c.dev_status[j] != 0 {
						c.irq_pend = true
						Irq_pending = true
						M.SetMemory(0x44, uint32(c.dev_status[j])<<16)
						M.SetMemory(0x40, 0)
						c.dev_status[j] = 0
						return uint16((i << 8)) | uint16(j)
					}
				}
				c.irq_pend = false
			}
		}
	}
	// No pending device
	return NO_DEV
}

// IPL a device.
func Boot_device(dev_num uint16) bool {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return true
	}
	ch &= 0xf
	d := dev_num & 0xff

	sc := find_subchannel(dev_num)

	cu := &ch_unit[ch]
	// Check if channel disabled
	if !cu.enabled {
		return true
	}

	// If no device or channel, return CC = 3
	if cu.dev_tab[d] == nil || sc == nil {
		return true
	}
	status := uint16(cu.dev_tab[d].Start_IO()) << 8
	if status != 0 {
		return true
	}

	sc.chan_status = 0
	sc.dev = cu.dev_tab[d]
	sc.caw = 0x8
	sc.daddr = dev_num
	sc.ccw_count = 24
	sc.ccw_flags = FLAG_CC | FLAG_SLI
	sc.ccw_addr = 0
	sc.ccw_key = 0
	sc.chan_byte = BUFF_EMPTY
	sc.chan_dirty = false

	sc.chan_status |= uint16(sc.dev.Start_cmd(sc.ccw_cmd)) << 8

	// Check if any errors from initial command
	if (sc.chan_status & (STATUS_ATTN | STATUS_CHECK | STATUS_EXPT)) != 0 {
		sc.ccw_cmd = 0
		sc.ccw_flags = 0
		return true
	}
	Loading = dev_num
	return false
}

// Add a device at given address
func Add_device(dev D.Device, dev_num uint16) bool {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return true
	}
	ch &= 0xf
	d := dev_num & 0xff

	cu := &ch_unit[ch]
	// Check if channel disabled
	if !cu.enabled || cu.dev_tab[d] != nil {
		return true
	}

	cu.dev_tab[d] = dev
	return false
}

// Delete a device at a given address
func Del_device(dev_num uint16) {
	ch := (dev_num >> 8)
	if ch > MAX_CHAN {
		return
	}
	ch &= 0xf
	d := dev_num & 0xff
	cu := &ch_unit[ch]
	cu.dev_tab[d] = nil
}

// Enable a channel of a given type
func Add_channel(chan_num uint16, ty int, subchan int) {
	if chan_num <= MAX_CHAN {
		cu := &ch_unit[chan_num]
		cu.enabled = true
		cu.ch_type = ty
		switch ty {
		case TYPE_DIS:
			cu.enabled = false
		case TYPE_SEL:
			cu.nsubchan = 1
		case TYPE_MUX:
			cu.nsubchan = subchan
		case TYPE_BMUX:
			cu.nsubchan = 32
		}
	}
}

// Initialize all channels and clear any device assignments
func InitializeChannels() {
	//var d *D.Device = nil
	for i := range MAX_CHAN {
		cu := &ch_unit[i]
		cu.enabled = false
		cu.irq_pend = false
		cu.ch_type = 0
		cu.nsubchan = 0
		for j := range 256 {
			cu.dev_tab[j] = null_dev
			cu.dev_status[j] = 0
			cu.subchan[j].daddr = NO_DEV
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
func find_subchannel(dev_num uint16) *chan_ctl {
	ch := (dev_num >> 8) & 0xf
	if ch > MAX_CHAN {
		return nil
	}
	device := int(dev_num & 0xff)
	switch ch_unit[ch].ch_type {
	case TYPE_SEL:
		return &ch_unit[ch].subchan[0]
	case TYPE_BMUX:
		if bmux_enable {
			d := (dev_num >> 3) & 0x1f
			return &ch_unit[ch].subchan[d]
		}
		return &ch_unit[ch].subchan[0]
	case TYPE_MUX:
		if device >= ch_unit[ch].nsubchan {
			if device < 128 { // All shared devices over subchannels
				return nil
			}
			device = (device >> 4) & 0x7
		}
		return &ch_unit[ch].subchan[device]
	}
	return nil
}

// Save full csw.
func store_csw(ch *chan_ctl) {
	M.SetMemory(0x40, (uint32(ch.ccw_key)<<24)|ch.caw)
	M.SetMemory(0x44, uint32(ch.ccw_count)|(uint32(ch.chan_status)<<16))
	if (ch.chan_status & STATUS_PCI) != 0 {
		ch.chan_status &= ^STATUS_PCI
	} else {
		ch.chan_status = 0
	}
	ch.ccw_flags &= ^FLAG_PCI
}

// Load in the next CCW, return true if failure, false if success.
func load_ccw(cu *chan_unit, sc *chan_ctl, tic_ok bool) bool {
	var word uint32
	var error bool
	var cmd bool = false
	var chain bool = false
	var c uint8

loop:
	// If last chain, start command
	if sc.chain_flg && (sc.ccw_flags&FLAG_CD) == 0 {
		chain = true
		sc.chain_flg = false
		cmd = true
		goto start_cmd
	}

	// Abort if ccw not on double word boundry
	if (sc.caw & 0x7) != 0 {
		sc.chan_status |= STATUS_PCHK
		return true
	}

	// Abort if we have pending errors
	if (sc.chan_status & 0x7F) != 0 {
		return true
	}

	// Remember if we were chainging
	if (sc.ccw_flags & FLAG_CC) != 0 {
		chain = true
	}

	// Check if we have status modifier set
	if (sc.chan_status & STATUS_MOD) != 0 {
		sc.caw += 8
		sc.caw &= M.AMASK
		sc.chan_status &= ^STATUS_MOD
	}

	// Read in next CCW
	if word, error = readfull(cu, sc, sc.caw); error {
		return true
	}

	// TIC can't follow TIC nor bt first in chain
	c = uint8((word >> 24) & 0xf)
	if c == CMD_TIC {
		if tic_ok {
			sc.caw = word & M.AMASK
			tic_ok = false
			goto loop
		}
		sc.chan_status |= STATUS_PCHK
		cu.irq_pend = true
		Irq_pending = true
		return true
	}
	sc.caw += 4
	sc.caw &= M.AMASK

	// Check if not chaining data
	if (sc.ccw_flags & FLAG_CD) == 0 {
		sc.ccw_cmd = uint8((word >> 24) & 0xff)
		cmd = true
	}

	// Set up for this command
	sc.ccw_addr = word & M.AMASK
	if word, error = readfull(cu, sc, sc.caw); error {
		return true
	}
	sc.caw += 4
	sc.caw &= M.AMASK
	sc.ccw_count = uint16(word & M.HMASK)

	// Copy SLI indicator in CD command
	if (sc.ccw_flags & (FLAG_CD | FLAG_SLI)) == (FLAG_CD | FLAG_SLI) {
		word |= uint32(FLAG_SLI) << 16
	}
	sc.ccw_flags = uint16(word>>16) & 0xff00
	sc.chan_byte = BUFF_EMPTY

	// Check if invalid count
	if sc.ccw_count == 0 {
		sc.chan_status |= STATUS_PCHK
		sc.ccw_cmd = 0
		cu.irq_pend = true
		Irq_pending = true
		return true
	}

	// Handle IDA
	if (sc.ccw_flags & FLAG_IDA) != 0 {
		if word, error = readfull(cu, sc, sc.ccw_addr); error {
			return true
		}
		sc.ccw_iaddr = word & M.AMASK
	}

start_cmd:
	if cmd {
		// Check if invalid command
		if (sc.ccw_cmd & 0xf) == 0 {
			sc.chan_status |= STATUS_PCHK
			sc.ccw_cmd = 0
			cu.irq_pend = true
			Irq_pending = true
			return true
		}

		if sc.dev == nil {
			return true
		}

		sc.chan_byte = BUFF_EMPTY
		sc.chan_status &= 0xff
		sc.chan_status |= uint16(sc.dev.Start_cmd(sc.ccw_cmd)) << 8

		// If device is busy, check if last was CC, then mark pending
		if (sc.chan_status & STATUS_BUSY) != 0 {
			if chain {
				sc.chain_flg = true
			}
			return false
		}

		// Check if any errors from initial command
		if (sc.chan_status & (STATUS_ATTN | STATUS_CHECK | STATUS_EXPT)) != 0 {
			sc.ccw_cmd = 0
			sc.ccw_flags = 0
			cu.dev_status[sc.daddr&0xff] = uint8((sc.chan_status >> 8) & 0xff)
			cu.irq_pend = true
			Irq_pending = true
			return true
		}

		// Check if meediate channel end
		if (sc.chan_status & STATUS_CEND) != 0 {
			sc.ccw_flags |= FLAG_SLI // Force SLI for immediate command
			if (sc.chan_status & STATUS_DEND) != 0 {
				// If we are not chaining, save status.
				//			if (sc.ccw_flags & FLAG_CC) != 0 {
				//				cu.dev_status[sc.daddr&0xff] = uint8((sc.chan_status >> 8) & 0xff)
				//			}
				sc.ccw_cmd = 0
				cu.irq_pend = true
				Irq_pending = true
			}
		}
	}

	if (sc.ccw_flags & FLAG_PCI) != 0 {
		sc.chan_status |= STATUS_PCI
		cu.irq_pend = true
		Irq_pending = true
	}
	return false
}

// Read a fill word from memory
// Return true if fail and false if success
func readfull(cu *chan_unit, sc *chan_ctl, addr uint32) (uint32, bool) {
	if !M.CheckAddr(addr) {
		sc.chan_status |= STATUS_PCHK
		cu.irq_pend = true
		Irq_pending = true
		return 0, true
	}
	if sc.ccw_key != 0 {
		k := M.GetKey(addr)
		if (k&0x8) != 0 && (k&0xf0) != sc.ccw_key {
			sc.chan_status |= STATUS_PCHK
			cu.irq_pend = true
			Irq_pending = true
			return 0, true
		}
	}
	w := M.GetMemory(addr)
	return w, false
}

// Read a word into channel buffer
// Return true if fail, false if success
func readbuff(cu *chan_unit, sc *chan_ctl) bool {
	var addr uint32
	var error bool

	if (sc.ccw_flags & FLAG_IDA) != 0 {
		addr = sc.ccw_iaddr
	} else {
		addr = sc.ccw_addr
	}
	if sc.chan_buf, error = readfull(cu, sc, addr); error {
		sc.chan_byte = BUFF_CHNEND
		return error
	}
	sc.chan_byte = uint8(addr & 3)
	return false
}

// Write channel buffer to memory
// Return true if fail, false if success
func writebuff(cu *chan_unit, sc *chan_ctl) bool {
	var addr uint32
	var error bool

	if (sc.ccw_flags & FLAG_IDA) != 0 {
		addr = sc.ccw_iaddr
	} else {
		addr = sc.ccw_addr
	}

	// Check if address valid
	addr &= M.AMASK
	if !M.CheckAddr(addr) {
		sc.chan_status |= STATUS_PCHK
		sc.chan_byte = BUFF_CHNEND
		cu.irq_pend = true
		Irq_pending = true
		return true
	}

	// Check protection key
	if sc.ccw_key != 0 {
		k := M.GetKey(addr)
		if (k & 0xf0) != sc.ccw_key {
			sc.chan_status |= STATUS_PCHK
			sc.chan_byte = BUFF_CHNEND
			cu.irq_pend = true
			Irq_pending = true
			return true
		}
	}

	// Write memory
	error = M.PutWord(addr, sc.chan_buf)
	sc.chan_byte = BUFF_EMPTY
	sc.chan_dirty = false
	return error
}
