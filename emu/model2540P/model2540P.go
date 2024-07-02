/* IBM 2540 Card Punch.

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

   This is the standard card reader.

   These units each buffer one record in local memory and signal
   ready when the buffer is full or empty. The channel must be
   ready to receive/transmit data when they are activated since
   they will transfer their block during chan_cmd. All data is
   transmitted as BCD characters.

*/

package model2540p

import (
	dev "github.com/rcornwell/S370/emu/device"
	"github.com/rcornwell/S370/emu/event"
	sys_channel "github.com/rcornwell/S370/emu/sys_channel"
	card "github.com/rcornwell/S370/util/card"
)

type Model2540Pctx struct {
	addr       uint16            // Current device address
	currentCol int               // Current column
	busy       bool              // Reader busy
	eof        bool              // EOF pending
	err        bool              // Error pending
	ready      bool              // Have card ready to punch
	halt       bool              // Signal halt requested
	sense      uint8             // Current sense byte
	image      card.Card         // Current card image
	context    *card.CardContext // Context for card reader.
}

// Handle start of CCW chain.
func (device *Model2540Pctx) StartIO() uint8 {
	return 0
}

// Start the card punch to punch one card.
func (device *Model2540Pctx) StartCmd(cmd uint8) uint8 {
	var status uint8

	// If busy return busy status right away
	if device.busy {
		return dev.CStatusBusy
	}

	// Decode command
	switch cmd & 0o7 {
	case 0:
		return 0
	// Punch a card.
	case dev.CmdWrite:
		device.halt = false
		device.currentCol = 0
		device.sense = 0
		device.ready = false
		if !device.context.Attached() {
			device.sense = dev.SenseINTVENT
			status = dev.CStatusChnEnd | dev.CStatusDevEnd
		} else {
			device.busy = true
			event.AddEvent(device, device.callback, 100, int(cmd))
		}

	// Queue up sense command
	case dev.CmdSense:
		if cmd != dev.CmdSense {
			device.sense |= dev.SenseCMDREJ
		} else {
			device.busy = true
			event.AddEvent(device, device.callback, 10, int(cmd))
			status = 0
		}
	case dev.CmdCTL:
		device.sense = 0
		status = dev.CStatusChnEnd | dev.CStatusDevEnd
		if cmd != dev.CmdCTL {
			device.sense |= dev.SenseCMDREJ
		}
		if !device.context.Attached() {
			device.sense = dev.SenseINTVENT
		}

	default:
		device.sense = dev.SenseCMDREJ
	}

	if device.sense != 0 {
		status = dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
	}
	device.halt = false
	return status
}

// Handle HIO instruction.
func (device *Model2540Pctx) HaltIO() uint8 {
	device.halt = true
	return 1
}

// Initialize a device.
func (device *Model2540Pctx) InitDev() uint8 {
	device.sense = 0
	device.busy = false
	device.halt = false
	device.eof = false
	device.err = false
	return 0
}

// Process card punch operations.
func (device *Model2540Pctx) callback(cmd int) {
	if cmd == int(dev.CmdSense) {
		device.busy = false
		device.halt = false
		_ = sys_channel.ChanWriteByte(device.addr, device.sense)
		sys_channel.ChanEnd(device.addr, (dev.CStatusChnEnd | dev.CStatusDevEnd))
		return
	}

	// If ready, punch out current card.
	if device.ready {
		switch device.context.PunchCard(device.image) {
		case card.CardOK:
			sys_channel.SetDevAttn(device.addr, dev.CStatusDevEnd)
		default:
			sys_channel.SetDevAttn(device.addr, dev.CStatusDevEnd|dev.CStatusCheck)
		}
		device.currentCol = 0
		device.ready = false
		device.busy = false
		return
	}

	// Add next byte to image.
	if device.currentCol < 80 {
		var char uint8
		var err bool

		char, err = sys_channel.ChanReadByte(device.addr)
		if err {
			device.ready = false
		} else {
			device.image.Image[device.currentCol] = card.EBCDICToHol(char)
			device.currentCol++
			if device.currentCol == 80 {
				device.ready = true
			}
		}
	}
	if device.ready {
		sys_channel.ChanEnd(device.addr, dev.CStatusChnEnd)
		event.AddEvent(device, device.callback, 1000, cmd)
		device.ready = true
	} else {
		event.AddEvent(device, device.callback, 100, cmd)
	}
}
