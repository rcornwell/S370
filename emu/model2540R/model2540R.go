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
	"errors"
	"fmt"
	"strings"

	config "github.com/rcornwell/S370/config/configparser"
	dev "github.com/rcornwell/S370/emu/device"
	event "github.com/rcornwell/S370/emu/event"
	ch "github.com/rcornwell/S370/emu/sys_channel"
	card "github.com/rcornwell/S370/util/card"
)

const (
	maskStack = 0xc0 // Mask for stacker option
	maskCMD   = 0x27 // Mask command part of
)

type Model2540Rctx struct {
	addr       uint16            // Current device address
	currentCol int               // Current column
	busy       bool              // Reader busy
	eof        bool              // EOF pending
	err        bool              // Error pending
	ready      bool              // Have card ready to read
	halt       bool              // Signal halt requested
	sense      uint8             // Current sense byte
	image      card.Card         // Current card image
	context    *card.CardContext // Context for card reader.
}

// Handle start of CCW chain.
func (device *Model2540Rctx) StartIO() uint8 {
	return 0
}

// Handle start of new command.
func (device *Model2540Rctx) StartCmd(cmd uint8) uint8 {
	var r uint8

	// If busy return busy status right away
	if device.busy {
		return dev.CStatusBusy
	}

	// Decode command
	switch cmd & maskCMD {
	case 0:
		return 0
	case dev.CmdRead:
		var err int
		if !device.context.Attached() {
			device.halt = false
			device.sense = dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		device.sense = 0
		device.currentCol = 0
		if device.eof {
			device.eof = false
			device.err = false

			// Read next card.
			device.image, err = device.context.ReadCard()
			switch err {
			case card.CardOK:
				device.ready = true
			case card.CardEOF:
				device.eof = true
			case card.CardEmpty:
			case card.CardError:
				device.err = true
				device.ready = true
			}
			r = dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusExpt
		}
		// Check if no more cards left in deck
		if device.context.HopperSize() == 0 {
			device.sense = dev.SenseINTVENT
		} else {
			device.busy = true
			event.AddEvent(device, device.callback, 100, int(cmd))
			return 0
		}
		// Queue up sense command
	case dev.CmdSense:
		if cmd != dev.CmdSense {
			device.sense |= dev.SenseCMDREJ
		} else {
			device.busy = true
			event.AddEvent(device, device.callback, 10, int(cmd))
			r = 0
		}
	case dev.CmdCTL:
		device.sense = 0
		if cmd == dev.CmdCTL {
			r = dev.CStatusChnEnd | dev.CStatusDevEnd
			break
		}
		if (cmd&0x30) != 0x20 || (cmd&maskStack) == maskStack {
			device.sense |= dev.SenseCMDREJ
			r = dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		} else {
			device.busy = true
			event.AddEvent(device, device.callback, 1000, int(cmd))
			r = dev.CStatusChnEnd
		}

	default:
		device.sense = dev.SenseCMDREJ
	}

	if device.sense != 0 {
		r = dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
	}
	device.halt = false
	return r
}

// Handle HIO instruction.
func (device *Model2540Rctx) HaltIO() uint8 {
	device.halt = true
	return 1
}

// Initialize a device.
func (device *Model2540Rctx) InitDev() uint8 {
	device.currentCol = 0
	device.sense = 0
	device.busy = false
	device.halt = false
	device.eof = false
	device.err = false
	return 0
}

// Attach file to device.
func (device *Model2540Rctx) Attach(fileName string) error {
	return device.context.Attach(fileName, false, false)
}

// Detach device.
func (device *Model2540Rctx) Detach() error {
	return device.context.Detach()
}

// Handle channel operations.
func (device *Model2540Rctx) callback(cmd int) {
	var status uint8
	var err int
	var xlat uint16

	if cmd == int(dev.CmdSense) {
		device.busy = false
		device.halt = false
		_ = ch.ChanWriteByte(device.addr, device.sense)
		ch.ChanEnd(device.addr, (dev.CStatusChnEnd | dev.CStatusDevEnd))
		return
	}

	// Handle feed end
	if cmd == 0x100 {
		device.busy = false
		ch.SetDevAttn(device.addr, dev.CStatusDevEnd)
		return
	}
	if device.halt {
		goto feed
	}

	// Check if new card requested
	if !device.ready {
		// Read next card.
		device.image, err = device.context.ReadCard()
		switch err {
		case card.CardOK:
			device.ready = true
		case card.CardEOF:
			device.eof = true
			device.busy = false
			device.halt = false
			ch.SetDevAttn(device.addr, dev.CStatusDevEnd|status)
			return
		case card.CardEmpty:
			device.busy = false
			device.halt = false
			ch.SetDevAttn(device.addr, dev.CStatusDevEnd|status)
			return
		case card.CardError:
			device.err = true
			device.ready = true
			device.busy = false
			device.halt = false
		}

		if device.err {
			status = dev.CStatusCheck
		}
	}

	// Copy next column of card over
	if (cmd & maskCMD) == int(dev.CmdRead) {
		if device.err {
			device.sense = dev.SenseDATCHK
			goto feed
		}
	}
	xlat = card.HolToEBCDIC(device.image.Image[device.currentCol])

	if xlat == 0x100 {
		device.sense = dev.SenseDATCHK
		xlat = 0
	} else {
		xlat &= 0xff
	}
	if ch.ChanWriteByte(device.addr, uint8(xlat)) {
		goto feed
	}
	device.currentCol++
	if device.currentCol != 80 {
		event.AddEvent(device, device.callback, 20, cmd)
		return
	}

feed:
	device.halt = false
	// If feed give, request a new card
	if (cmd & maskStack) != maskStack {
		device.ready = false
		ch.ChanEnd(device.addr, dev.CStatusChnEnd)
		event.AddEvent(device, device.callback, 1000, 0x100) // Feed the card
	} else {
		if device.err {
			status = dev.CStatusCheck
		}
		device.busy = false
		ch.ChanEnd(device.addr, (dev.CStatusChnEnd | dev.CStatusDevEnd | status))
	}
}

// register a device on initialize.
func init() {
	config.RegisterModel("2540R", config.TypeModel, create)
}

// Create a card reader device.
func create(devNum uint16, _ string, options []config.Option) error {
	dev := Model2540Rctx{addr: devNum}
	err := ch.AddDevice(&dev, devNum)
	if err != nil {
		err := fmt.Sprintf("Unable to create 2540R at %03x\n", devNum)
		return errors.New(err)
	}
	dev.context = card.NewCardContext(card.ModeAuto)
	eof := false
	for _, option := range options {
		switch strings.ToUpper(option.Name) {
		case "FORMAT", "FMT":
			if !dev.context.SetFormat(option.EqualOpt) {
				return errors.New("Invalid Card formt type: " + option.EqualOpt)
			}
		case "EOF":
			eof = true
		case "NOEOF":
			eof = false
		case "FILE":
			if option.EqualOpt == "" {
				return errors.New("File option missing filename")
			}
			err := dev.context.Attach(option.EqualOpt, false, eof)
			if err != nil {
				return err
			}
		default:
			return errors.New("Reader invalid option: " + option.Name)
		}
		if option.Value != nil {
			return errors.New("Extra options not supported on: " + option.Name)
		}
	}
	return nil
}
