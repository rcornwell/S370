/* IBM 2540 Card Reader.

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
   ready to receeve/transmit data when they are activated since
   they will transfer their block during chan_cmd. All data is
   transmitted as BCD characters.

*/

package model2540r

import (
	ev "github.com/rcornwell/S370/emu/event"
	ch "github.com/rcornwell/S370/emu/sys_channel"
	card "github.com/rcornwell/S370/util/card"
)

const (
	maskStack = 0xc0 // Mask for stacker option
	maskCMD   = 0x27 // Mask command part of
)

type Model2540Rctx struct {
	addr  uint16            // Current device address
	col   int               // Current column
	busy  bool              // Reader busy
	eof   bool              // EOF pending
	err   bool              // Error pending
	rdy   bool              // Have card ready to read
	halt  bool              // Signal halt requested
	sense uint8             // Current sense byte
	image card.Card         // Current card image
	ctx   *card.CardContext // Context for card reader.
}

// Handle start of CCW chain.
func (d *Model2540Rctx) StartIO() uint8 {
	return 0
}

// Handle start of new command.
func (d *Model2540Rctx) StartCmd(cmd uint8) uint8 {
	var r uint8

	// If busy return busy status right away
	if d.busy {
		return ch.CStatusBusy
	}

	// Decode command
	switch cmd & maskCMD {
	case 0:
		return 0
	case ch.CmdRead:
		var err int
		if !d.ctx.Attached() {
			d.halt = false
			d.sense = ch.SenseINTVENT
			return ch.CStatusChnEnd | ch.CStatusDevEnd | ch.CStatusCheck
		}
		d.sense = 0
		d.col = 0
		if d.eof {
			d.eof = false
			d.err = false

			// Read next card.
			d.image, err = d.ctx.ReadCard()
			switch err {
			case card.CardOK:
				d.rdy = true
			case card.CardEOF:
				d.eof = true
			case card.CardEmpty:
			case card.CardError:
				d.err = true
				d.rdy = true
			}
			r = ch.CStatusChnEnd | ch.CStatusDevEnd | ch.CStatusExpt
		}
		// Check if no more cards left in deck
		if !d.rdy {
			d.sense = ch.SenseINTVENT
		} else {
			d.busy = true
			ev.AddEvent(d, d.callback, 100, int(cmd))
			return 0
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
		if cmd == ch.CmdCTL {
			r = ch.CStatusChnEnd | ch.CStatusDevEnd
			break
		}
		if (cmd&0x30) != 0x20 || (cmd&maskStack) == maskStack {
			d.sense |= ch.SenseCMDREJ
			r = ch.CStatusChnEnd | ch.CStatusDevEnd | ch.CStatusCheck
		} else {
			d.busy = true
			ev.AddEvent(d, d.callback, 1000, int(cmd))
			r = ch.CStatusChnEnd
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

// Handle HIO instruction.
func (d *Model2540Rctx) HaltIO() uint8 {
	d.halt = true
	return 1
}

// Initialize a device.
func (d *Model2540Rctx) InitDev() uint8 {
	d.col = 0
	d.sense = 0
	d.busy = false
	d.halt = false
	d.eof = false
	d.err = false
	return 0
}

// Handle channel operations.
func (d *Model2540Rctx) callback(cmd int) {
	var r uint8
	var err int
	var xlat uint16

	if cmd == int(ch.CmdSense) {
		d.busy = false
		d.halt = false
		_ = ch.ChanWriteByte(d.addr, d.sense)
		ch.ChanEnd(d.addr, (ch.CStatusChnEnd | ch.CStatusDevEnd))
		return
	}

	// Handle feed end
	if cmd == 0x100 {
		d.busy = false
		ch.SetDevAttn(d.addr, ch.CStatusDevEnd)
		return
	}
	if d.halt {
		goto feed
	}
	// Check if new card requested
	if !d.rdy {
		if d.err {
			r = ch.CStatusCheck
		}
	}
	// Read next card.
	d.image, err = d.ctx.ReadCard()
	switch err {
	case card.CardOK:
		d.rdy = true
	case card.CardEOF:
		d.eof = true
		d.busy = false
		d.halt = false
		ch.SetDevAttn(d.addr, ch.CStatusDevEnd|r)
		return
	case card.CardEmpty:
		d.busy = false
		d.halt = false
		ch.SetDevAttn(d.addr, ch.CStatusDevEnd|r)
		return
	case card.CardError:
		d.err = true
		d.rdy = true
		d.busy = false
		d.halt = false
	}

	// Copy next column of card over
	if (cmd & maskCMD) == int(ch.CmdRead) {
		if d.err {
			d.sense = ch.SenseDATCHK
			goto feed
		}
	}
	xlat = card.HolToEBCDIC(d.image.Image[d.col])

	if xlat == 0x100 {
		d.sense = ch.SenseDATCHK
		xlat = 0
	} else {
		xlat &= 0xff
	}
	if ch.ChanWriteByte(d.addr, uint8(xlat)) {
		goto feed
	}
	d.col++
	if d.col != 80 {
		ev.AddEvent(d, d.callback, 20, cmd)
		return
	}
feed:
	d.halt = false
	// If feed give, request a new card
	if (cmd & maskStack) != maskStack {
		d.rdy = false
		ch.ChanEnd(d.addr, ch.CStatusChnEnd)
		ev.AddEvent(d, d.callback, 1000, 0) // Feed the card
	} else {
		if d.err {
			r = ch.CStatusCheck
		}
		d.busy = false
		ch.ChanEnd(d.addr, (ch.CStatusChnEnd | ch.CStatusDevEnd | r))
	}
}
