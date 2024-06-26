/*
 * S370 - Test device controller.
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

package cpu

import (
	Dv "github.com/rcornwell/S370/emu/device"
	Ev "github.com/rcornwell/S370/emu/event"
	Ch "github.com/rcornwell/S370/emu/sys_channel"
)

type TestDev struct {
	Addr  uint16     // Current device address
	Mask  uint16     // Mask for unit
	Data  [256]uint8 // Data to read/write
	count int        // Pointer to input/output
	Max   int        // Maximum size of date
	Sense uint8      // Current sense byte
	halt  bool       // Halt I/O requested
	busy  bool       // Device is busy
	Sms   bool       // Return SMS at end of command
}

//  /*
//   *  Commands.
//   *
//   *            01234567
//   *  Write     00000001
//   *  Read      00000010
//   *  Nop       00000011
//   *  One Byte  00001011    Read one byte of option.
//   *  End       00010011    Immediate channel end, device end after 100 cycles.
//   *  Sense     00000100    Return one byte of sense data.
//   *  Read Bk   00001100
//   */

// Handle start of CCW chain.
func (d *TestDev) StartIO() uint8 {
	return 0
}

// Handle start of new command.
func (d *TestDev) StartCmd(cmd uint8) uint8 {
	var r uint8
	if d.busy {
		return Dv.CStatusBusy
	}
	switch cmd & 7 {
	case 0: // Test I/O
		return 0
	case 1: // Write
		d.Sense = 0
		d.count = 0
		d.busy = true
	case 2: // Read
		d.Sense = 0
		d.count = 0
		d.busy = true
	case 3: // Nop or control
		d.Sense = 0
		d.count = 0
		switch cmd {
		case 0x03: // Nop
			r = Dv.CStatusChnEnd | Dv.CStatusDevEnd
		case 0x0b: // Grab a data byte
			d.busy = true
		case 0x13: // Issue channel end
			r = Dv.CStatusChnEnd
			d.busy = true
			Ev.AddEvent(d, d.callback, 10, int(cmd))
		default:
			d.Sense = Dv.SenseCMDREJ
		}
	case 4: // Sense
		switch cmd {
		case 0x0c: // Read backward
			d.Sense = 0
			d.count = 0
			d.busy = true
		case 0x4: // Sense
			d.busy = true
			Ev.AddEvent(d, d.callback, 10, int(cmd))
			d.busy = true
			return 0
		default:
			d.Sense = Dv.SenseCMDREJ
		}
	default:
		d.Sense = Dv.SenseCMDREJ
	}

	d.halt = false
	if d.Sense != 0 {
		r = Dv.CStatusChnEnd | Dv.CStatusDevEnd | Dv.CStatusCheck
	} else if (r & Dv.CStatusChnEnd) == 0 {
		Ev.AddEvent(d, d.callback, 10, int(cmd))
	}
	return r
}

// Handle HIO instruction.
func (d *TestDev) HaltIO() uint8 {
	d.halt = true
	return 1
}

// Initialize a device.
func (d *TestDev) InitDev() uint8 {
	d.busy = false
	d.count = 0
	d.Max = 0
	d.Sense = 0
	d.Sms = false
	return 0
}

// Handle channel operations.
func (d *TestDev) callback(cmd int) {
	var v uint8
	var e bool

	switch cmd {
	case 0x01: // Write
		r := Dv.CStatusChnEnd | Dv.CStatusDevEnd
		if d.Sms {
			r |= Dv.CStatusSMS
		}
		if d.count > d.Max {
			d.busy = false
			d.Sms = false
			Ch.ChanEnd(d.Addr, r)
			return
		}
		if d.halt {
			Ch.ChanEnd(d.Addr, Dv.CStatusChnEnd|Dv.CStatusDevEnd)
			d.busy = false
			d.halt = false
			return
		}
		v, e = Ch.ChanReadByte(d.Addr)
		if e {
			d.Data[d.count] = v
			d.count++
			d.busy = false
			d.Sms = false
			Ch.ChanEnd(d.Addr, r)
			return
		}
		d.Data[d.count] = v
		d.count++
		Ev.AddEvent(d, d.callback, 10, cmd)
	case 0x02, 0x0c: // Read and Read backwards
		r := Dv.CStatusChnEnd | Dv.CStatusDevEnd
		if d.Sms {
			r |= Dv.CStatusSMS
		}
		if d.count >= d.Max {
			d.busy = false
			d.Sms = false
			Ch.ChanEnd(d.Addr, r)
			return
		}
		if d.halt {
			Ch.ChanEnd(d.Addr, Dv.CStatusChnEnd|Dv.CStatusDevEnd)
			d.busy = false
			d.halt = false
			return
		}
		if Ch.ChanWriteByte(d.Addr, d.Data[d.count]) {
			d.busy = false
			d.Sms = false
			Ch.ChanEnd(d.Addr, r)
		} else {
			d.count++
			Ev.AddEvent(d, d.callback, 10, cmd)
		}
	case 0x0b:
		d.Data[0], _ = Ch.ChanReadByte(d.Addr)
		Ch.ChanEnd(d.Addr, Dv.CStatusChnEnd)
		Ev.AddEvent(d, d.callback, 10, 0x13)
	case 0x04:
		if Ch.ChanWriteByte(d.Addr, d.Sense) {
			Ch.ChanEnd(d.Addr, Dv.CStatusChnEnd|Dv.CStatusDevEnd|Dv.CStatusExpt)
		} else {
			Ch.ChanEnd(d.Addr, Dv.CStatusChnEnd|Dv.CStatusDevEnd)
		}
		d.busy = false
	case 0x13: // Return channel end
		d.busy = false
		Ch.SetDevAttn(d.Addr, Dv.CStatusDevEnd)
	}
}
