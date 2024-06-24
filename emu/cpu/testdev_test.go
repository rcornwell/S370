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
	ev "github.com/rcornwell/S370/emu/event"
	ch "github.com/rcornwell/S370/emu/sys_channel"
)

type Test_dev struct {
	Addr  uint16     // Current device address
	Mask  uint16     // Mask for device address
	cmd   uint8      // Current command
	Data  [256]uint8 // Data to read/write
	count int        // Pointer to input/output
	Max   int        // Maximum size of date
	sense uint8      // Current sense byte
	halt  bool       // Halt I/O requested
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

// Handle start of CCW chain
func (d *Test_dev) StartIO() uint8 {
	return 0
}

// Handle start of new command
func (d *Test_dev) StartCmd(cmd uint8) uint8 {
	var r uint8 = 0
	if d.cmd != 0 {
		return ch.CStatusBusy
	}
	switch cmd & 7 {
	case 0: // Test I/O
		return 0
	case 1: // Write
		d.sense = 0
		d.count = 0
		d.cmd = cmd
	case 2: // Read
		d.sense = 0
		d.count = 0
		d.cmd = cmd
	case 3: // Nop or control
		d.sense = 0
		d.cmd = cmd
		d.count = 0
		switch cmd {
		case 0x03: // Nop
			r = ch.CStatusChnEnd | ch.CStatusDevEnd
			d.cmd = 0
		case 0x0b: // Grab a data byte
		case 0x13: // Issue channel end
			r = ch.CStatusChnEnd
			ev.AddEvent(d, d.callback, 10, 1)
		default:
			d.cmd = 0
			d.sense = ch.SenseCMDREJ
		}
	case 4: // Sense
		d.cmd = cmd
		if cmd == 0x0c { // Read backward
			d.sense = 0
			d.count = 0
		} else if cmd == 0x4 { // Sense
			ev.AddEvent(d, d.callback, 10, 1)
			return 0
		} else {
			d.cmd = 0
			d.sense = ch.SenseCMDREJ
		}
	default:
		d.sense = ch.SenseCMDREJ
	}

	d.halt = false
	if d.sense != 0 {
		r = ch.CStatusChnEnd | ch.CStatusDevEnd | ch.CStatusCheck
	} else if (r & ch.CStatusChnEnd) == 0 {
		ev.AddEvent(d, d.callback, 10, 1)
	}
	return r
}

// Handle HIO instruction
func (d *Test_dev) HaltIO() uint8 {
	d.halt = true
	return 1
}

// Initialize a device.
func (d *Test_dev) InitDev() uint8 {
	d.cmd = 0
	d.count = 0
	d.Max = 0
	d.sense = 0
	d.Sms = false
	return 0
}

// Handle channel operations
func (d *Test_dev) callback(iarg int) {
	var v uint8
	var e bool

	switch d.cmd {
	case 0x01: // Write
		r := ch.CStatusChnEnd | ch.CStatusDevEnd
		if d.Sms {
			r |= ch.CStatusSMS
		}
		if d.count > d.Max {
			d.cmd = 0
			d.Sms = false
			ch.ChanEnd(d.Addr, r)
			return
		}
		if d.halt {
			ch.ChanEnd(d.Addr, ch.CStatusChnEnd|ch.CStatusDevEnd)
			d.cmd = 0
			d.halt = false
			return
		}
		v, e = ch.ChanReadByte(d.Addr)
		if e {
			d.Data[d.count] = v
			d.count++
			d.cmd = 0
			d.Sms = false
			ch.ChanEnd(d.Addr, r)
			return
		}
		d.Data[d.count] = v
		d.count++
		ev.AddEvent(d, d.callback, 10, 1)
	case 0x02, 0x0c: // Read and Read backwards
		r := ch.CStatusChnEnd | ch.CStatusDevEnd
		if d.Sms {
			r |= ch.CStatusSMS
		}
		if d.count >= d.Max {
			d.cmd = 0
			d.Sms = false
			ch.ChanEnd(d.Addr, r)
			return
		}
		if d.halt {
			ch.ChanEnd(d.Addr, ch.CStatusChnEnd|ch.CStatusDevEnd)
			d.cmd = 0
			d.halt = false
			return
		}
		if ch.ChanWriteByte(d.Addr, d.Data[d.count]) {
			d.cmd = 0
			d.Sms = false
			ch.ChanEnd(d.Addr, r)
		} else {
			d.count++
			ev.AddEvent(d, d.callback, 10, 1)
		}
	case 0x0b:
		d.cmd = 0x13
		d.Data[0], _ = ch.ChanReadByte(d.Addr)
		ch.ChanEnd(d.Addr, ch.CStatusChnEnd)
		ev.AddEvent(d, d.callback, 10, 1)
	case 0x04:
		d.cmd = 0
		if ch.ChanWriteByte(d.Addr, d.sense) {
			ch.ChanEnd(d.Addr, ch.CStatusChnEnd|ch.CStatusDevEnd|ch.CStatusExpt)
		} else {
			ch.ChanEnd(d.Addr, ch.CStatusChnEnd|ch.CStatusDevEnd)
		}
	case 0x13: // Return channel end
		d.cmd = 0
		ch.SetDevAttn(d.Addr, ch.CStatusDevEnd)
	}
}
