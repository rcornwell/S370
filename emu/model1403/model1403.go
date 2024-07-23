/* IBM 1403 Line printer.

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

package model1403

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/rcornwell/S370/command/command"
	config "github.com/rcornwell/S370/config/configparser"
	dev "github.com/rcornwell/S370/emu/device"
	event "github.com/rcornwell/S370/emu/event"
	ch "github.com/rcornwell/S370/emu/sys_channel"
	"github.com/rcornwell/S370/util/xlat"
)

const (
	// Debug options.
	debugCmd = 1 << iota
	debugData
	debugDetail
)

var debugOption = map[string]int{
	"CMD":    debugCmd,
	"DATA":   debugData,
	"DETAIL": debugDetail,
}

type Model1403ctx struct {
	addr     uint16      // Current device address.
	busy     bool        // Reader busy.
	halt     bool        // Signal halt requested.
	sense    uint8       // Current sense byte.
	file     *os.File    // Printer file.
	fcb      [100]uint16 // FCB tape.
	fcbName  string      // Name of current FCB.
	lpp      uint32      // Lines per page
	lineNum  uint32      // Current line number.
	detachk  bool        // Don't return data-check.
	ch12     bool        // Channel 12 sense.
	buffer   [140]uint8  // buffer.
	bufPtr   int         // Pointer to where in buffer we are.
	full     bool        // Buffer full.
	debugMsk int         // Debug option mask.
}

var legacy = []uint16{
	/* 1      2      3      4      5      6      7      8      9     10       lines  */
	0x800, 0x000, 0x000, 0x000, 0x000, 0x000, 0x400, 0x000, 0x000, 0x000, /*  1 - 10 */
	0x000, 0x000, 0x200, 0x000, 0x000, 0x000, 0x000, 0x000, 0x100, 0x000, /* 11 - 20 */
	0x000, 0x000, 0x000, 0x000, 0x080, 0x000, 0x000, 0x000, 0x000, 0x000, /* 21 - 30 */
	0x040, 0x000, 0x000, 0x000, 0x000, 0x000, 0x020, 0x000, 0x000, 0x000, /* 31 - 40 */
	0x000, 0x000, 0x010, 0x000, 0x000, 0x000, 0x000, 0x000, 0x004, 0x000, /* 41 - 50 */
	0x000, 0x000, 0x000, 0x000, 0x002, 0x000, 0x000, 0x000, 0x000, 0x000, /* 51 - 60 */
	0x001, 0x000, 0x008, 0x000, 0x000, 0x001, 0x1000, /* 61 - 66 */
}

/*
PROGRAMMMING NOTE:  the below cctape value SHOULD match

	the same corresponding fcb value!
*/
var std1 = []uint16{
	/* 1      2      3      4      5      6      7      8      9     10       lines  */
	0x800, 0x000, 0x000, 0x000, 0x000, 0x000, 0x400, 0x000, 0x000, 0x000, /*  1 - 10 */
	0x000, 0x000, 0x200, 0x000, 0x000, 0x000, 0x000, 0x000, 0x100, 0x000, /* 11 - 20 */
	0x000, 0x000, 0x000, 0x000, 0x080, 0x000, 0x000, 0x000, 0x000, 0x000, /* 21 - 30 */
	0x040, 0x000, 0x000, 0x000, 0x000, 0x000, 0x020, 0x000, 0x000, 0x000, /* 31 - 40 */
	0x000, 0x000, 0x010, 0x000, 0x000, 0x000, 0x000, 0x000, 0x008, 0x000, /* 41 - 50 */
	0x000, 0x000, 0x000, 0x000, 0x004, 0x000, 0x000, 0x000, 0x000, 0x000, /* 51 - 60 */
	0x002, 0x000, 0x001, 0x000, 0x000, 0x001, 0x1000, /* 61 - 66 */
}

var none = []uint16{
	/* 1      2      3      4      5      6      7      8      9     10       lines  */
	0x800, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, /*  1 - 10 */
	0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, /* 11 - 20 */
	0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, /* 21 - 30 */
	0x040, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, /* 31 - 40 */
	0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, /* 41 - 50 */
	0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, 0x000, /* 51 - 60 */
	0x002, 0x000, 0x000, 0x000, 0x000, 0x001, 0x1000, /* 61 - 66 */
}

var fcbTables = map[string][]uint16{
	"LEGACY": legacy,
	"STD1":   std1,
	"NONE":   none,
}

// Handle start of CCW chain.
func (device *Model1403ctx) StartIO() uint8 {
	return 0
}

// Start the line printer.
func (device *Model1403ctx) StartCmd(cmd uint8) uint8 {
	var status uint8

	// If busy return busy status right away
	if device.busy {
		return dev.CStatusBusy
	}

	switch cmd & 3 {
	case dev.CmdWrite:
		if device.file == nil {
			device.sense = dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		device.bufPtr = 0
		device.sense = 0
		device.busy = true
		event.AddEvent(device, device.callback, 100, int(cmd))
	case dev.CmdCTL:
		if cmd == dev.CmdCTL { // Nop is always immediate.
			return dev.CStatusChnEnd | dev.CStatusDevEnd
		}
		if device.file == nil {
			device.sense = dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		device.bufPtr = 0
		device.sense = 0
		device.busy = true
		event.AddEvent(device, device.callback, 100, int(cmd))
	case 0: // Sense command
		if cmd != dev.CmdSense {
			device.sense |= dev.SenseCMDREJ
		} else {
			device.busy = true
			event.AddEvent(device, device.callback, 10, int(cmd))
			status = 0
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
func (device *Model1403ctx) HaltIO() uint8 {
	device.halt = true
	return 1
}

// Initialize a device.
func (device *Model1403ctx) InitDev() uint8 {
	device.sense = 0
	device.busy = false
	device.halt = false
	return 0
}

// Shutdown device.
func (device *Model1403ctx) Shutdown() {
	_ = device.Detach()
}

// Enable debug options.
func (device *Model1403ctx) Debug(opt string) error {
	flag, ok := debugOption[opt]
	if !ok {
		return errors.New("1403 debug option invalid: " + opt)
	}
	device.debugMsk |= flag
	return nil
}

// List of valid options.
func (device *Model1403ctx) Options(_ string) []command.Options {
	fcbList := []string{}
	for k := range fcbTables {
		fcbList = append(fcbList, strings.ToLower(k))
	}
	return []command.Options{
		{
			Name:        "file",
			OptionType:  command.OptionFile,
			OptionValid: command.ValidAttach | command.ValidShow,
		},
		{
			Name:        "lpp",
			OptionType:  command.OptionNumber,
			OptionValid: command.ValidAttach | command.ValidSet,
		},
		{
			Name:        "linesperpage",
			OptionType:  command.OptionNumber,
			OptionValid: command.ValidAttach | command.ValidSet | command.ValidShow,
		},
		{
			Name:        "fcb",
			OptionType:  command.OptionList,
			OptionValid: command.ValidAttach | command.ValidSet | command.ValidShow,
			OptionList:  fcbList,
		},
	}
}

// Attach file to device.
func (device *Model1403ctx) Attach(opts []*command.CmdOption) error {
	_ = device.Detach()
	for _, opt := range opts {
		switch opt.Name {
		case "file":
			if opt.EqualOpt == "" {
				return errors.New("file requires file name")
			}
			if device.file != nil {
				return errors.New("only one file name option allowd")
			}
			var err error
			device.file, err = os.Create(opt.EqualOpt)
			if err != nil {
				return err
			}

		case "lpp", "linesperpage":
			if opt.EqualOpt == "" {
				return errors.New("lines per page requires number")
			}
			if opt.Value == 0 || opt.Value > 100 {
				return errors.New("number of lines per page to large")
			}
			device.lpp = opt.Value

		case "fcb":
			if opt.EqualOpt == "" {
				return errors.New("fcb requires name")
			}

			table, ok := fcbTables[opt.EqualOpt]
			if !ok {
				return errors.New("invalid fcb name")
			}

			for i, v := range table {
				device.fcb[i] = v
				device.lpp++
				if (v & 0x1000) != 0 {
					break
				}
			}
		default:
			return errors.New("invalid option: " + opt.Name)
		}
	}
	return nil
}

// Detach device.
func (device *Model1403ctx) Detach() error {
	if device.file != nil {
		device.file.Close()
		device.file = nil
	}
	return nil
}

// Set command.
func (device *Model1403ctx) Set(unset bool, opts []*command.CmdOption) error {
	if unset {
		return errors.New("unset option not supported")
	}

	for _, opt := range opts {
		switch opt.Name {
		case "lpp", "linesperpage":
			if opt.EqualOpt == "" {
				return errors.New("lines per page requires number")
			}
			if opt.Value == 0 || opt.Value > 100 {
				return errors.New("number of lines per page to large")
			}
			device.lpp = opt.Value

		case "fcb":
			if opt.EqualOpt == "" {
				return errors.New("fcb requires name")
			}

			table, ok := fcbTables[opt.EqualOpt]
			if !ok {
				return errors.New("invalid fcb name")
			}

			for i, v := range table {
				device.fcb[i] = v
				device.lpp++
				if (v & 0x1000) != 0 {
					break
				}
			}
			device.fcbName = opt.EqualOpt

		default:
			return errors.New("invalid option: " + opt.Name)
		}
	}
	return nil
}

// Show command.
func (device *Model1403ctx) Show(opts []*command.CmdOption) (string, error) {
	flags := 0

	str := fmt.Sprintf("%03x:", device.addr)
	for _, opt := range opts {
		switch opt.Name {
		case "file":
			flags |= 1

		case "lpp", "linesperpage":
			flags |= 2

		case "fcb":
			flags |= 4
		default:
			return "", errors.New("invalid option: " + opt.Name)
		}
	}

	if flags == 0 {
		flags = 7
	}
	if (flags & 2) != 0 {
		str += fmt.Sprintf(" lpp=%d", device.lpp)
	}
	if (flags & 4) != 0 {
		str += " fcb=" + strings.ToLower(device.fcbName)
	}
	if (flags & 1) != 0 {
		if device.file != nil {
			str += " " + device.file.Name()
		} else {
			str += " not attached"
		}
	}

	return str, nil
}

// Rewind tape to start.
func (device *Model1403ctx) Rewind() error {
	return command.NotSupported
}

// Reset a device.
func (device *Model1403ctx) Reset() error {
	if device.InitDev() != 0 {
		return errors.New("device failed to reset")
	}
	return nil
}

// Return device address.
func (device *Model1403ctx) GetAddr() uint16 {
	return device.addr
}

// Print a line of text.
func (device *Model1403ctx) printLine(cmd int) {
	// If buffer full print line.

	space := (cmd >> 3) & 0x1f
	if device.full {
		out := ""

		// Convert line to EBCDIC and output.
		for i := range device.bufPtr {
			ch := device.buffer[i]
			ch = xlat.EBCDICToASCII[ch]
			if !unicode.IsPrint(rune(ch)) {
				ch = '.'
			}
			out += string(ch)
		}

		// Remove trailing blanks.
		out = strings.TrimRightFunc(out, unicode.IsSpace)

		// Print out the line.
		fmt.Fprint(device.file, out)
		device.bufPtr = 0
		device.full = false
	}

	// Short spacing.
	if space < 4 {
		for space != 0 {
			fmt.Fprintln(device.file)
			fcb := device.fcb[device.lineNum]
			if (cmd & 3) != 1 {
				if (fcb & (0x1000 >> 9)) != 0 {
					device.sense |= dev.SenseOPRCHK // Channel 9
				}
				if (fcb & (0x1000 >> 12)) != 0 {
					device.ch12 = true
				}
			}
			if (fcb&0x1000) != 0 || device.lineNum > device.lpp {
				fmt.Fprintln(device.file, "")
				fmt.Fprintln(device.file, "\f")
				device.lineNum = 0
			} else {
				device.lineNum++
			}
			space--
		}
		return
	}

	// Handle skip to channel.
	mask := uint16(0x1000) >> (space & 0xf)
	line := 0
	for (device.fcb[device.lineNum] & mask) == 0 {
		line++
		device.lineNum++
		if (device.fcb[device.lineNum]&0x1000) != 0 ||
			device.lineNum > device.lpp {
			fmt.Fprintln(device.file, "")
			fmt.Fprintln(device.file, "\f")
			return
		}
	}

	if (device.fcb[device.lineNum] & mask) != 0 {
		for line > 0 {
			fmt.Fprintln(device.file, "")
			line--
		}
	}
}

// Process card punch operations.
func (device *Model1403ctx) callback(cmd int) {
	// Process sense command.
	if cmd == int(dev.CmdSense) {
		device.busy = false
		device.halt = false
		_ = ch.ChanWriteByte(device.addr, device.sense)
		ch.ChanEnd(device.addr, (dev.CStatusChnEnd | dev.CStatusDevEnd))
		return
	}

	if cmd == 7 {
		device.bufPtr = 0
		device.busy = false
		device.halt = false
		_, _ = ch.ChanReadByte(device.addr)
		ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd)
		return
	}

	// Handle Block-Data-Check.
	if (cmd & 0xf7) == 0x73 {
		if (cmd & 0x8) != 0 {
			device.detachk = false
		} else {
			device.detachk = true
		}
		device.bufPtr = 0
		device.busy = false
		device.halt = false
		_, _ = ch.ChanReadByte(device.addr)
		ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd)
		return
	}

	// Handle load UCS.
	if (cmd & 0xf7) == 0xf3 {
		for range 250 {
			_, end := ch.ChanReadByte(device.addr)
			if end {
				break
			}
		}
		device.busy = false
		device.halt = false
		_, _ = ch.ChanReadByte(device.addr)
		ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd)
	}

	space := (cmd >> 3) & 0x1f
	// Check for valid form motion.
	if (cmd&0x6) == 1 && ((space > 3 && space < 0x10) || space > 0x1d) {
		device.sense |= dev.SenseCMDREJ
		device.busy = false
		device.halt = false
		if (cmd & 0x7) == 3 {
			ch.SetDevAttn(device.addr, dev.CStatusDevEnd|dev.CStatusCheck)
		} else {
			ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd|dev.CStatusCheck)
		}
		return
	}

	if device.full || (cmd&7) == 3 {
		device.printLine(cmd)
		device.full = false
		device.bufPtr = 0
		status := dev.CStatusDevEnd
		if device.ch12 {
			status |= dev.CStatusExpt
			device.ch12 = false
		}
		if device.sense != 0 {
			status |= dev.CStatusCheck
		}
		ch.SetDevAttn(device.addr, status)
		return
	}

	// If halt raised.
	if device.halt {
		device.full = true
		ch.ChanEnd(device.addr, dev.CStatusChnEnd)
		device.halt = false
		event.AddEvent(device, device.callback, 5000, cmd)
		return
	}

	// Copy next column over.
	data, end := ch.ChanReadByte(device.addr)
	if end {
		device.full = true
	} else {
		device.buffer[device.bufPtr] = data
		device.bufPtr++
	}
	if device.full || device.bufPtr > 132 {
		ch.ChanEnd(device.addr, dev.CStatusChnEnd)
		device.halt = false
		device.full = true
		event.AddEvent(device, device.callback, 5000, cmd)
		return
	}
	event.AddEvent(device, device.callback, 20, cmd)
}

// register a device on initialize.
func init() {
	config.RegisterModel("1403", config.TypeModel, create)
}

// Create a card punch device.
func create(devNum uint16, _ string, options []config.Option) error {
	device := Model1403ctx{addr: devNum}
	err := ch.AddDevice(&device, &device, devNum)
	if err != nil {
		return fmt.Errorf("unable to create 1403 at %03x", devNum)
	}
	fcb := ""
	for _, option := range options {
		switch strings.ToUpper(option.Name) {
		case "FCB":
			if option.EqualOpt == "" {
				return errors.New(("set fcb missing name"))
			}
			if fcb != "" {
				return errors.New("fcb duplicated")
			}
			f := strings.ToUpper(option.EqualOpt)
			_, ok := fcbTables[f]
			if !ok {
				return errors.New("fcb not in available")
			}
			fcb = f
		case "LPP", "LINESPERPAGE":
			if option.EqualOpt == "" {
				return errors.New(("set fcb missing name"))
			}
			if device.lpp != 0 {
				return errors.New("lines per page duplicated")
			}
			lines, errx := strconv.ParseUint(option.Name, 10, 7)
			if errx != nil {
				return errors.New("lines per page not a number")
			}
			device.lpp = uint32(lines)
		case "FILE":
			if device.file != nil {
				return errors.New("file option duplicated")
			}
			if option.EqualOpt == "" {
				return errors.New("file option missing filename")
			}
			device.file, err = os.Create(option.EqualOpt)
			if err != nil {
				return err
			}
		default:
			return errors.New("printer invalid option " + option.Name)
		}
		if option.Value != nil {
			return errors.New("extra options not supported on: " + option.Name)
		}
	}

	// Set Lines per page to default value.
	if device.lpp == 0 {
		device.lpp = 56
	}

	// Copy over the FCB table.
	if fcb == "" {
		fcb = "NONE"
	}
	table := fcbTables[fcb]
	for i, v := range table {
		device.fcb[i] = v
		device.lpp++
		if (v & 0x1000) != 0 {
			break
		}
	}

	device.fcbName = fcb
	return nil
}
