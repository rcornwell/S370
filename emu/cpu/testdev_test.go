package cpu

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

import (
	Ev "github.com/rcornwell/S370/emu/event"
	Ch "github.com/rcornwell/S370/emu/sys_channel"
)

type Test_dev struct {
	addr  uint16     // Current device address
	mask  uint16     // Mask for device address
	cmd   uint8      // Current command
	data  [256]uint8 // Data to read/write
	count int        // Pointer to input/output
	max   int        // Maximum size of date
	sense uint8      // Current sense byte
	halt  bool       // Halt I/O requested
	sms   bool       // Return SMS at end of command
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
		return Ch.SNS_BSY
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
			r = Ch.SNS_CHNEND | Ch.SNS_DEVEND
			d.cmd = 0
		case 0x0b: // Grab a data byte
		case 0x13: // Issue channel end
			r = Ch.SNS_CHNEND
			Ev.AddEvent(d, d.callback, 10, 1)
		default:
			d.cmd = 0
			d.sense = Ch.SNS_CMDREJ
		}
	case 4: // Sense
		d.cmd = cmd
		if cmd == 0x0c { // Read backward
			d.sense = 0
			d.count = 0
		} else if cmd == 0x4 { // Sense
			Ev.AddEvent(d, d.callback, 10, 1)
			return 0
		} else {
			d.cmd = 0
			d.sense = Ch.SNS_CMDREJ
		}
	default:
		d.sense = Ch.SNS_CMDREJ
	}

	d.halt = false
	if d.sense != 0 {
		r = Ch.SNS_CHNEND | Ch.SNS_DEVEND | Ch.SNS_UNITCHK
	} else if (r & Ch.SNS_CHNEND) == 0 {
		Ev.AddEvent(d, d.callback, 10, 1)
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
	d.max = 0
	d.sense = 0
	d.sms = false
	return 0
}

// Handle channel operations
func (d *Test_dev) callback(iarg int) {
	var v uint8
	var e bool

	switch d.cmd {
	case 0x01: // Write
		r := Ch.SNS_CHNEND | Ch.SNS_DEVEND
		if d.sms {
			r |= Ch.SNS_SMS
		}
		if d.count > d.max {
			d.cmd = 0
			d.sms = false
			Ch.ChanEnd(d.addr, r)
			return
		}
		if d.halt {
			Ch.ChanEnd(d.addr, Ch.SNS_CHNEND|Ch.SNS_DEVEND)
			d.cmd = 0
			d.halt = false
			return
		}
		v, e = Ch.ChanReadByte(d.addr)
		if e {
			d.data[d.count] = v
			d.count++
			d.cmd = 0
			d.sms = false
			Ch.ChanEnd(d.addr, r)
			return
		}
		d.data[d.count] = v
		d.count++
		Ev.AddEvent(d, d.callback, 10, 1)
	case 0x02, 0x0c: // Read and Read backwards
		r := Ch.SNS_CHNEND | Ch.SNS_DEVEND
		if d.sms {
			r |= Ch.SNS_SMS
		}
		if d.count >= d.max {
			d.cmd = 0
			d.sms = false
			Ch.ChanEnd(d.addr, r)
			return
		}
		if d.halt {
			Ch.ChanEnd(d.addr, Ch.SNS_CHNEND|Ch.SNS_DEVEND)
			d.cmd = 0
			d.halt = false
			return
		}
		if Ch.ChanWriteByte(d.addr, d.data[d.count]) {
			d.cmd = 0
			d.sms = false
			Ch.ChanEnd(d.addr, r)
		} else {
			d.count++
			Ev.AddEvent(d, d.callback, 10, 1)
		}
	case 0x0b:
		d.cmd = 0x13
		d.data[0], _ = Ch.ChanReadByte(d.addr)
		Ch.ChanEnd(d.addr, Ch.SNS_CHNEND)
		Ev.AddEvent(d, d.callback, 10, 1)
	case 0x04:
		d.cmd = 0
		if Ch.ChanWriteByte(d.addr, d.sense) {
			Ch.ChanEnd(d.addr, Ch.SNS_CHNEND|Ch.SNS_DEVEND|Ch.SNS_UNITEXP)
		} else {
			Ch.ChanEnd(d.addr, Ch.SNS_CHNEND|Ch.SNS_DEVEND)
		}
	case 0x13: // Return channel end
		d.cmd = 0
		Ch.SetDevAttn(d.addr, Ch.SNS_DEVEND)
	}
}
