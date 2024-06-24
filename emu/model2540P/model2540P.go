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
   ready to recieve/transmit data when they are activated since
   they will transfer their block during chan_cmd. All data is
   transmitted as BCD characters.

*/

package model2540P

import (
	ev "github.com/rcornwell/S370/emu/event"
	ch "github.com/rcornwell/S370/emu/sys_channel"
	card "github.com/rcornwell/S370/util/card"
)

type Model2540P_ctx struct {
	addr  uint16            // Current device address
	col   int               // Current column
	busy  bool              // Reader busy
	eof   bool              // EOF pending
	err   bool              // Error pending
	rdy   bool              // Have card ready to punch
	halt  bool              // Signal halt requested
	count int               // Pointer to input/output
	sense uint8             // Current sense byte
	image card.Card         // Current card image
	ctx   *card.CardContext // Context for card reader.
}

// Handle start of CCW chain
func (d *Model2540P_ctx) StartIO() uint8 {
	return 0
}

// Start the card punch to punch one card.
func (d *Model2540P_ctx) StartCmd(cmd uint8) uint8 {
	var r uint8 = 0

	// If busy return busy status right away
	if d.busy {
		return ch.CStatusBusy
	}

	// Decode command
	switch cmd & 07 {
	case 0:
		return 0
	// Punch a card.
	case ch.CmdWrite:
		d.halt = false
		d.col = 0
		d.sense = 0
		d.rdy = false
		if !d.ctx.Attached() {
			d.sense = ch.SenseINTVENT
			r = ch.CStatusChnEnd | ch.CStatusDevEnd
		} else {
			d.busy = true
			ev.AddEvent(d, d.callback, 100, int(cmd))
		}

	// Queue up sense command
	case ch.CmdSense:
		if cmd != ch.CmdSense {
			d.sense |= ch.SenseCMDREJ
		} else {
			d.busy = true
			ev.AddEvent(d, d.callback, 10, int(cmd))
			r = 0
		}
	case ch.CmdCTL:
		d.sense = 0
		r = ch.CStatusChnEnd | ch.CStatusDevEnd
		if cmd != ch.CmdCTL {
			d.sense |= ch.SenseCMDREJ

		}
		if !d.ctx.Attached() {
			d.sense = ch.SenseINTVENT
		}

	default:
		d.sense = ch.SenseCMDREJ
	}

	if d.sense != 0 {
		r = ch.CStatusChnEnd | ch.CStatusDevEnd | ch.CStatusCheck
	}
	d.halt = false
	return r
}

// Handle HIO instruction
func (d *Model2540P_ctx) HaltIO() uint8 {
	d.halt = true
	return 1
}

// Initialize a device.
func (d *Model2540P_ctx) InitDev() uint8 {

	d.count = 0
	d.sense = 0
	d.busy = false
	d.halt = false
	d.eof = false
	d.err = false
	return 0
}

func (d *Model2540P_ctx) callback(cmd int) {
	if cmd == int(ch.CmdSense) {
		d.busy = false
		d.halt = false
		_ = ch.ChanWriteByte(d.addr, d.sense)
		ch.ChanEnd(d.addr, (ch.CStatusChnEnd | ch.CStatusDevEnd))
		return
	}

	// If ready, punch out current card.
	if d.rdy {
		switch d.ctx.PunchCard(d.image) {
		case card.CARD_OK:
			ch.SetDevAttn(d.addr, ch.CStatusDevEnd)
		default:
			ch.SetDevAttn(d.addr, ch.CStatusDevEnd|ch.CStatusCheck)
		}
		d.col = 0
		d.rdy = false
		d.busy = false
		return
	}

	// Add next byte to image.
	if d.col < 80 {
		var c uint8
		var err bool

		c, err = ch.ChanReadByte(d.addr)
		if err {
			d.rdy = true
		} else {
			d.image.Image[d.col] = card.EbcdicToHol(c)
			d.col++
			if d.col == 80 {
				d.rdy = true
			}
		}
	}
	if d.rdy {
		ch.ChanEnd(d.addr, ch.CStatusChnEnd)
		ev.AddEvent(d, d.callback, 1000, int(cmd))
		d.rdy = true
	} else {
		ev.AddEvent(d, d.callback, 100, int(cmd))
	}
}
