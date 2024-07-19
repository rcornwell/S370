/* IBM 2400 and 3400 tape drive emulation.

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

package modelTape

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rcornwell/S370/command/command"
	config "github.com/rcornwell/S370/config/configparser"
	dev "github.com/rcornwell/S370/emu/device"
	event "github.com/rcornwell/S370/emu/event"
	ch "github.com/rcornwell/S370/emu/sys_channel"
	debug "github.com/rcornwell/S370/util/debug"
	"github.com/rcornwell/S370/util/tape"
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

type Model2400ctx struct {
	addr      uint16        // Current device address
	halt      bool          // Halt current operation
	busy      bool          // Tape is busy
	rewind    bool          // Tape is rewinding
	unload    bool          // Tape will unload after rewind
	density   int           // Tape density setting
	odd       bool          // Odd parity
	trans     bool          // Translator turned on
	conv      bool          // Convert to byte
	cc        int           // Current character for data converter
	hold      uint8         // Hold point for current character.
	seven     bool          // 7 track tape.
	skip      bool          // Skip to EOR
	mark      bool          // Last read detected a mark.
	frameTime int           // Time for each frame based on density setting
	sense     [6]uint8      // Sense data
	senseLen  int           // Number of sense bytes
	context   *tape.Context // Context for tape drive
	debugMsk  int           // Debug options mask
}

// Translate BCD to EBCIDC
var bcdToEbcdic = [64]byte{
	0x40, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7,
	0xf8, 0xf9, 0xf0, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f,
	0x7a, 0x61, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
	0xe8, 0xe9, 0xe0, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f,
	0x60, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7,
	0xd8, 0xd9, 0xd0, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
	0x50, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7,
	0xc8, 0xc9, 0xc0, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f,
}

const (
	// Command codes.
	cmdREW       uint8 = 0x07 // Rewind command
	cmdRUN       uint8 = 0x0f // Rewind and unload
	cmdERG       uint8 = 0x17 // Erase Gap
	cmdWTM       uint8 = 0x1f // Write Tape Mark
	cmdBSR       uint8 = 0x27 // Back space record
	cmdBSF       uint8 = 0x2f // Back space file
	cmdFSR       uint8 = 0x37 // Forward space record
	cmdFSF       uint8 = 0x3f // Forward space file
	cmdSecureERA uint8 = 0x97 // 3400 Security erase
	cmdSenseRES  uint8 = 0xf4 // 3400 Sense reserve
	cmdSenseREL  uint8 = 0xb4 // 3400 Sense release
	cmdReqTIE    uint8 = 0x1b // Request track in error
	cmdMODE      uint8 = 0x03 // Mode command
	cmdMODEMSK   uint8 = 0x07 // Mode Mask

	// Sense Byte 0 values.
	senseZero uint8 = dev.SenseOVRRUN

	// Sense Byte 1 values.
	senseNoise   uint8 = 0x80 // Noise record detected
	senseTUAStA  uint8 = 0x40 // Selected and ready
	senseTUBSTA  uint8 = 0x20 // Note ready, rewinding
	sense7Track  uint8 = 0x10 // 7 track drive
	senseLoad    uint8 = 0x08 // Tape at load point
	senseWrite   uint8 = 0x04 // Unit write
	senseNoRing  uint8 = 0x02 // No write ring
	senseDensity uint8 = 0x01 // Density error (9 track only)

	// Sense byte 2 values.
	senseByte2 uint8 = 0x03 // Not supported feature

	// Sense byte 3 values.
	senseVRC     uint8 = 0x80 // Virtical parity error
	senseLRCR    uint8 = 0x40 // Logituntial parity error
	senseSkew    uint8 = 0x20 // Skew
	senseCRC     uint8 = 0x10 // CRC error (9 track only)
	senseSkewVRC uint8 = 0x08 // VRC Skew
	sensePE      uint8 = 0x04 // Phase error
	senseBack    uint8 = 0x1  // Tape in backward status
)

// Handle start of CCW chain.
func (device *Model2400ctx) StartIO() uint8 {
	// If busy return busy status right away
	if device.busy || device.rewind {
		return dev.CStatusBusy
	}

	return 0
}

// Start the card punch to punch one card.
func (device *Model2400ctx) StartCmd(cmd uint8) uint8 {
	// If busy return busy status right away
	if device.busy {
		return dev.CStatusBusy
	}

	//	fmt.Printf(" Tape: cmd %02x\n", cmd)
	// Decode command
	switch cmd & 0xF {
	case 0:
		return 0

	// Data transfer commands.

	case dev.CmdRead, dev.CmdWrite, dev.CmdRDBWD:
		// Clear sense data.
		for i := range device.sense {
			device.sense[i] = 0
		}
		if device.rewind || !device.context.Attached() {
			device.sense[0] |= dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		if cmd == dev.CmdRDBWD && device.context.TapeAtLoadPt() {
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		device.busy = true
		event.AddEvent(device, device.callback, 100, int(cmd))
		return 0

	// Tape motion.
	case 0x7, 0xf:
		// Clear sense data.
		for i := range device.sense {
			device.sense[i] = 0
		}
		if device.rewind || !device.context.Attached() {
			device.sense[0] |= dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		if (cmd == cmdBSF || cmd == cmdBSR) && device.context.TapeAtLoadPt() {
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		device.busy = true
		device.mark = false
		event.AddEvent(device, device.callback, 100, int(cmd))
		return dev.CStatusChnEnd

	// Queue up sense command
	case dev.CmdSense:
		// Only sense is supported.
		if cmd != dev.CmdSense {
			device.sense[0] |= dev.SenseCMDREJ
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}

		if device.rewind || !device.context.Attached() {
			device.sense[0] |= dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		device.busy = true
		event.AddEvent(device, device.callback, 10, int(cmd))
		return 0

	// Mode set commands.
	case dev.CmdCTL, 0xb:
		if device.rewind || !device.context.Attached() {
			device.sense[0] |= dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}
		if device.seven {
			device.sense[1] |= sense7Track
			setdensity := false
			switch cmd & 0x38 {
			case 0x00, 0x08: // Nop
			case 0x10: // Reset condition
				device.trans = false
				device.conv = true
				device.odd = true
				setdensity = true
			case 0x18: // 9 track NRZI- nop on 7 track
			case 0x20:
				device.trans = false
				device.conv = false
				device.odd = false
				setdensity = true
			case 0x28:
				device.trans = true
				device.conv = false
				device.odd = false
				setdensity = true
			case 0x30:
				device.trans = false
				device.conv = false
				device.odd = true
				setdensity = true
			case 0x38:
				device.trans = true
				device.conv = false
				device.odd = true
				setdensity = true
			}
			if setdensity {
				switch cmd & 0xc0 {
				case 0:
					device.density = tape.Density200
				case 0x40:
					device.density = tape.Density556
				case 0x80:
					device.density = tape.Density800
				case 0xc0:
				}
			}
		} else {
			device.density = 0
			switch cmd & 0xf8 {
			case 0:
				device.density = tape.Density1600
			case 0x08:
				device.density = tape.Density6250
			default:
				device.sense[0] |= dev.SenseCMDREJ
				return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
			}
		}
		for i := range device.sense {
			device.sense[i] = 0
		}
		return dev.CStatusChnEnd | dev.CStatusDevEnd
	default:
		device.sense[0] |= dev.SenseCMDREJ
	}

	status := dev.CStatusChnEnd | dev.CStatusDevEnd
	if device.sense[0] != 0 {
		status |= dev.CStatusCheck
	}
	device.halt = false
	return status
}

// Handle HIO instruction.
func (device *Model2400ctx) HaltIO() uint8 {
	device.halt = true
	return 1
}

// Initialize a device.
func (device *Model2400ctx) InitDev() uint8 {
	device.busy = false
	device.halt = false
	return 0
}

// Shutdown device.
func (device *Model2400ctx) Shutdown() {
	_ = device.context.Detach()
}

// Enable debug options.
func (device *Model2400ctx) Debug(opt string) error {
	flag, ok := debugOption[opt]
	if !ok {
		return errors.New("2400 debug option invalid: " + opt)
	}
	device.debugMsk |= flag
	return nil
}

// Options for attach command.
func (device *Model2400ctx) Options(_ string) []command.Options {
	formats := tape.GetFormatList()
	return []command.Options{
		{
			Name:        "file",
			OptionType:  command.OptionFile,
			OptionValid: command.ValidAttach | command.ValidShow,
		},
		{
			Name:        "fmt",
			OptionType:  command.OptionList,
			OptionValid: command.ValidAttach | command.ValidSet | command.ValidShow,
			OptionList:  formats,
		},
		{
			Name:        "format",
			OptionType:  command.OptionList,
			OptionValid: command.ValidAttach | command.ValidSet,
			OptionList:  formats,
		},
		{
			Name:        "ro",
			OptionType:  command.OptionSwitch,
			OptionValid: command.ValidAttach,
		},
		{
			Name:        "rw",
			OptionType:  command.OptionSwitch,
			OptionValid: command.ValidAttach,
		},
		{
			Name:        "ring",
			OptionType:  command.OptionSwitch,
			OptionValid: command.ValidAttach | command.ValidSet | command.ValidShow,
		},
		{
			Name:        "noring",
			OptionType:  command.OptionSwitch,
			OptionValid: command.ValidAttach | command.ValidShow,
		},
		{
			Name:        "7track",
			OptionType:  command.OptionSwitch,
			OptionValid: command.ValidSet,
		},
		{
			Name:        "9track",
			OptionType:  command.OptionSwitch,
			OptionValid: command.ValidSet,
		},
		{
			Name:        "type",
			OptionType:  command.OptionSwitch,
			OptionValid: command.ValidShow,
		},
	}
}

// Attach file to device.
func (device *Model2400ctx) Attach(opts []*command.CmdOption) error {
	err := device.Detach()
	if err != nil {
		return err
	}

	for _, opt := range opts {
		switch opt.Name {
		case "file":
			if opt.EqualOpt == "" {
				return errors.New("file requires file name")
			}
			if device.context.Attached() {
				return errors.New("only one file name option allowd")
			}

			err = device.context.Attach(opt.EqualOpt)
			if err != nil {
				break
			}

		case "fmt", "format":
			if opt.EqualOpt == "" {
				return errors.New("format requires option type")
			}
			err = device.context.SetFormat(opt.EqualOpt)
			if err != nil {
				break
			}

		case "ro", "noring":
			device.context.SetNoRing()

		case "rw", "ring":
			device.context.SetRing()

		default:
			return errors.New("invalid option: " + opt.Name)
		}
	}
	return err
}

// Detach device.
func (device *Model2400ctx) Detach() error {
	return device.context.Detach()
}

// Set command.
func (device *Model2400ctx) Set(unset bool, opts []*command.CmdOption) error {
	for _, opt := range opts {
		switch opt.Name {
		case "fmt", "format":
			if opt.EqualOpt == "" {
				return errors.New("format requires option type")
			}
			err := device.context.SetFormat(opt.EqualOpt)
			if err != nil {
				return err
			}

		case "noring":
			if unset {
				return errors.New("unset not valid for ring")
			}
			device.context.SetNoRing()

		case "ring":
			if unset {
				device.context.SetNoRing()
			} else {
				device.context.SetRing()
			}

		case "7track":
			if unset {
				device.context.Set9Track()
			} else {
				device.context.Set7Track()
			}

		case "9track":
			if unset {
				device.context.Set7Track()
			} else {
				device.context.Set9Track()
			}

		default:
			return errors.New("invalid option: " + opt.Name)
		}
	}
	return nil
}

// Show command.
func (device *Model2400ctx) Show(opts []*command.CmdOption) (string, error) {
	flags := 0

	str := fmt.Sprintf("%03x:", device.addr)
	for _, opt := range opts {
		switch opt.Name {
		case "file":
			flags |= 1
		case "fmt", "format":
			flags |= 2
		case "ring":
			flags |= 4
		case "type":
			flags |= 8
		default:
			return "", errors.New("invalid option: " + opt.Name)
		}
	}

	if flags == 0 {
		flags = 0xf
	}
	if (flags & 2) != 0 {
		str += " FMT=" + device.context.GetFormat()
	}
	if (flags & 4) != 0 {
		if device.context.TapeRing() {
			str += " RING"
		} else {
			str += " NORING"
		}
	}
	if (flags & 8) != 0 {
		if device.context.Tape9Track() {
			str += " 9 Track"
		} else {
			str += " 7 Track"
		}
	}
	if (flags & 1) != 0 {
		if device.context.Attached() {
			str += " " + device.context.FileName()
		} else {
			str += " not attached"
		}
	}

	return str, nil
}

// Callback for reqind commands.
func (device *Model2400ctx) callbackRewind(cmd int) {
	if device.context.RewindFrames(10000) {
		if device.unload {
			err := device.context.Detach()
			if err != nil {
				fmt.Println(err)
			}
			device.unload = false
		}
		device.rewind = false
		event.AddEvent(device, device.callbackFinish, 1000, cmd)
	} else {
		event.AddEvent(device, device.callbackRewind, 1000, cmd)
	}
}

// Read a frame of data.
func (device *Model2400ctx) readFrame(cmd int) {
	if device.halt {
		event.AddEvent(device, device.callbackData, 100, cmd)
		device.skip = true
		return
	}
	data, err := device.context.ReadFrame()
	if errors.Is(err, tape.TapeEOR) {
		//		fmt.Println("Tape EOR")
		event.AddEvent(device, device.callbackFinish, 1000, cmd)
		return
	}

	// Skip until end of record.
	if device.skip {
		event.AddEvent(device, device.callbackData, 100, cmd)
		return
	}

	// Set up call to finish
	if device.seven {
		mode := uint8(0o100)
		if device.odd {
			mode = 0
		}
		if (xlat.ParityTable[data&0o77] ^ (data & 0o100) ^ mode) == 0 {
			device.sense[0] |= dev.SenseDATCHK
			device.sense[2] |= senseVRC
		}
		data &= 0o77
		if device.trans {
			data = bcdToEbcdic[data]
		}

		// Data converter does not work in read backwards.
		if device.conv && cmd == int(dev.CmdRead) {
			hold := data
			switch device.cc {
			case 0:
				device.cc = 1
			case 1:
				data = (device.hold << 2) | ((data >> 4) & 0o3)
				device.cc = 2
			case 2:
				data = ((device.hold & 0o17) << 4) | ((data >> 4) & 0o17)
				device.cc = 3
			case 3:
				data |= (device.hold & 0o3) << 6
				device.cc = 0
				hold = 0
			}
			device.hold = hold
			// For first character nothing to send to CPU
			if device.cc == 0 {
				event.AddEvent(device, device.callbackData, 100, cmd)
				return
			}
		}
	}

	// Send data to CPU.
	if ch.ChanWriteByte(device.addr, data) {
		//			fmt.Println("Tape stop I/O")
		device.skip = true
	}

	event.AddEvent(device, device.callbackData, 100, cmd)
}

// Write a frame.
func (device *Model2400ctx) writeFrame(cmd int) {
	// Only can occur if 7 track converter enabled.
	// Send out last data value.
	if device.cc == 3 {
		err := device.context.WriteFrame(device.hold)
		device.cc = 0
		if err != nil {
			slog.Error(err.Error())
			event.AddEvent(device, device.callbackFinish, 1000, cmd)
		} else {
			event.AddEvent(device, device.callbackData, 100, cmd)
		}
		return
	}

	if device.halt {
		event.AddEvent(device, device.callbackFinish, 100, cmd)
		return
	}

	// Grab next data byte from channel.
	data, end := ch.ChanReadByte(device.addr)

	if end {
		event.AddEvent(device, device.callbackFinish, 1000, cmd)
		return
	}
	// Handle converter
	if device.seven {
		mode := uint8(0o100)
		if device.odd {
			mode = 0
		}

		if device.trans {
			data |= (data & 0xf) | ((data & 0x30) ^ 0x30)
		}

		if device.conv {
			hold := data
			switch device.cc {
			case 0:
				data >>= 2
				device.cc = 1
			case 1:
				data = ((device.hold & 0o3) << 4) | ((data >> 4) & 0o17)
				device.cc = 2
			case 2:
				data = ((device.hold & 0o17) << 2) | ((data >> 6) & 0o3)
				device.cc = 3
				hold = data & 0o77
				hold = xlat.ParityTable[hold&0o77] ^ mode
			case 3:
			}
			device.hold = hold
		}
		data = xlat.ParityTable[data&0o77] ^ mode
	}

	err := device.context.WriteFrame(data)
	if err != nil {
		//		fmt.Println(err)
		event.AddEvent(device, device.callbackFinish, 1000, cmd)
		return
	}
	// Indicate we wrote at least one character.
	device.sense[0] &= ^senseZero
	event.AddEvent(device, device.callbackData, 100, cmd)
}

// Callback to handle data transfers.
func (device *Model2400ctx) callbackData(cmd int) {
	switch uint8(cmd) {
	case dev.CmdRead, dev.CmdRDBWD:
		device.readFrame(cmd)
		return

	case dev.CmdWrite:
		device.writeFrame(cmd)
		return

	case cmdFSF, cmdFSR, cmdBSF, cmdBSR:
		data, err := device.context.ReadFrame()
		debug.DebugDevf(device.addr, device.debugMsk, debugCmd, "space %02x %02x", cmd, data)
		if !errors.Is(err, tape.TapeEOR) {
			if err != nil {
				slog.Debug(err.Error())
				event.AddEvent(device, device.callbackFinish, 1000, cmd)
			} else {
				event.AddEvent(device, device.callbackData, 100, cmd)
			}
			return
		}
		debug.DebugDevf(device.addr, device.debugMsk, debugCmd, "Space EOR %02x", cmd)
		// Search file, finish current read and start another.
		if cmd == int(cmdBSR) || cmd == int(cmdFSR) {
			event.AddEvent(device, device.callbackFinish, 1000, cmd)
			return
		}

		err = device.context.FinishRecord()
		if err != nil {
			fmt.Println(err)
			event.AddEvent(device, device.callbackFinish, 1000, cmd)
			return
		}

		device.mark = false
		if cmd == int(cmdBSF) {
			if device.context.TapeAtLoadPt() {
				event.AddEvent(device, device.callbackFinish, 1000, cmd)
				return
			}
			err = device.context.ReadBackStart()
		} else {
			err = device.context.ReadForwStart()
		}

		if errors.Is(err, tape.TapeMARK) {
			device.mark = true
			event.AddEvent(device, device.callbackFinish, 1000, cmd)
			return
		}
		event.AddEvent(device, device.callbackData, 100, cmd)
	case cmdWTM: // Write Tape Mark
		err := device.context.WriteMark()
		if err != nil {
			slog.Error(err.Error())
		}
		event.AddEvent(device, device.callbackFinish, 1000, cmd)
		return
	}

	// event.AddEvent(device, device.callbackData, 100, cmd)
}

// Callback to handle finish of data transfer.
func (device *Model2400ctx) callbackFinish(cmd int) {
	device.busy = false
	device.halt = false
	device.skip = false
	//	fmt.Printf("Tape finish: %02x\n", cmd)
	switch uint8(cmd) {
	case dev.CmdRead, dev.CmdRDBWD, dev.CmdWrite:
		err := device.context.FinishRecord()
		if err != nil {
			slog.Error(err.Error())
		}

		//	fmt.Printf("Finish read %t\n", device.mark)
		if device.mark {
			ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd|dev.CStatusExpt)
			device.mark = false
		} else {
			ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd)
		}

	case cmdFSF, cmdBSF:
		err := device.context.FinishRecord()
		if err != nil {
			slog.Error(err.Error())
		}

		//	fmt.Printf("Finish space %t\n", device.mark)
		device.mark = false
		ch.SetDevAttn(device.addr, dev.CStatusDevEnd)

	case cmdWTM:
		ch.SetDevAttn(device.addr, dev.CStatusDevEnd)

	case cmdFSR, cmdBSR:
		err := device.context.FinishRecord()
		if err != nil {
			slog.Error(err.Error())
		}

		if device.mark {
			ch.SetDevAttn(device.addr, dev.CStatusDevEnd|dev.CStatusExpt)
			device.mark = false
		} else {
			ch.SetDevAttn(device.addr, dev.CStatusDevEnd)
		}
	}
}

// Process tape operations.
func (device *Model2400ctx) callback(cmd int) {
	switch uint8(cmd) {
	case dev.CmdSense:
		device.halt = false
		device.busy = false
		if device.seven {
			device.sense[1] |= sense7Track
		}
		if device.context.Attached() {
			if !device.context.TapeRing() {
				device.sense[1] |= senseNoRing
			}
			if device.context.TapeAtLoadPt() {
				device.sense[1] |= senseLoad
			}
			device.sense[1] |= senseTUAStA
		}
		device.sense[2] = senseByte2
		if !device.seven {
			device.sense[3] |= sensePE
		}
		for i := range device.senseLen {
			var by uint8
			switch i {
			case 0, 1, 2, 3, 4, 5:
				by = device.sense[i]

			case 6:
				// Model 3 support dual density.
				by = uint8(0x23)
				if device.seven {
					by |= 0x80 // Indicate 7 track
				}

			case 10:
				by = device.sense[0] & dev.SenseCMDREJ

			case 13:
				if device.seven {
					by = 0x40 // Indicate 7 track
				} else {
					by = 0x80 // Indicate 9 track
				}

			default:
				by = 0
			}
			if ch.ChanWriteByte(device.addr, by) {
				break
			}
		}
		ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd)
		device.busy = false
	case dev.CmdRead:
		err := device.context.ReadForwStart()
		if errors.Is(err, tape.TapeMARK) {
			device.busy = false
			device.halt = false
			device.mark = true
			event.AddEvent(device, device.callbackFinish, 1000, cmd)
			ch.ChanEnd(device.addr, dev.CStatusChnEnd)
			return
		}
		if err != nil {
			slog.Error(err.Error())
			device.busy = false
			device.halt = false
			event.AddEvent(device, device.callbackFinish, 1000, cmd)
			ch.ChanEnd(device.addr, dev.CStatusChnEnd)
			return
		}
		device.cc = 0
		event.AddEvent(device, device.callbackData, 100, cmd)
	case dev.CmdRDBWD:
		if device.context.TapeAtLoadPt() {
			ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd|dev.CStatusCheck)
			return
		}
		err := device.context.ReadBackStart()
		if errors.Is(err, tape.TapeMARK) {
			device.busy = false
			device.halt = false
			device.mark = true
			event.AddEvent(device, device.callbackFinish, 1000, cmd)
			ch.ChanEnd(device.addr, dev.CStatusChnEnd)
			return
		}
		if err != nil {
			device.busy = false
			device.halt = false
			slog.Error(err.Error())
			event.AddEvent(device, device.callbackFinish, 1000, cmd)
			ch.ChanEnd(device.addr, dev.CStatusChnEnd)
			return
		}
		device.cc = 0
		device.sense[3] |= senseBack
		event.AddEvent(device, device.callbackData, 100, cmd)
	case dev.CmdWrite:
		if !device.context.TapeRing() {
			device.sense[0] |= dev.SenseCMDREJ
			device.busy = false
			device.halt = false
			ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd|dev.CStatusCheck)
			return
		}
		err := device.context.WriteStart()
		if err != nil {
			slog.Error(err.Error())
			device.busy = false
			device.halt = false
			ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd|dev.CStatusCheck)
			return
		}
		device.sense[0] |= senseZero
		device.sense[1] |= senseWrite
		device.cc = 0
		event.AddEvent(device, device.callbackData, 100, cmd)

	case cmdRUN: // Rewind and unload
		device.unload = true
		fallthrough
	case cmdREW: // Rewind command
		device.rewind = true
		err := device.context.StartRewind()
		if err != nil {
			slog.Error(err.Error())
			device.busy = false
			device.halt = false
			event.AddEvent(device, device.callbackRewind, 1000, cmd)
		} else {
			event.AddEvent(device, device.callbackRewind, 1000, cmd)
		}

	case cmdERG: // Erase Gap
		event.AddEvent(device, device.callbackFinish, 1000, cmd)

	case cmdWTM: // Write Tape Mark
		if !device.context.TapeRing() {
			device.sense[0] |= dev.SenseCMDREJ
			event.AddEvent(device, device.callbackFinish, 100, cmd)
		} else {
			event.AddEvent(device, device.callbackData, 100, cmd)
		}
	case cmdFSF, cmdFSR:
		device.mark = false
		err := device.context.ReadForwStart()
		if errors.Is(err, tape.TapeMARK) {
			device.mark = true
			event.AddEvent(device, device.callbackFinish, 100, cmd)
			return
		}
		if err != nil {
			slog.Error(err.Error())
			event.AddEvent(device, device.callbackFinish, 100, cmd)
			return
		}
		event.AddEvent(device, device.callbackData, 100, cmd)
	case cmdBSR, cmdBSF:
		device.mark = false
		err := device.context.ReadBackStart()
		if errors.Is(err, tape.TapeMARK) {
			device.mark = true
			event.AddEvent(device, device.callbackFinish, 100, cmd)
			return
		}
		if err != nil {
			slog.Error(err.Error())
			event.AddEvent(device, device.callbackFinish, 100, cmd)
			return
		}
		event.AddEvent(device, device.callbackData, 100, cmd)
	}
}

// register a device on initialize.
func init() {
	config.RegisterModel("2400", config.TypeModel, create)
}

// Create a card punch device.
func create(devNum uint16, _ string, options []config.Option) error {
	device := Model2400ctx{addr: devNum}
	err := ch.AddDevice(&device, &device, devNum)
	if err != nil {
		return fmt.Errorf("unable to create 2400 at %03x: %w", devNum, err)
	}
	device.context = tape.NewTapeContext()
	device.conv = true
	device.odd = true
	device.senseLen = 6
	for _, option := range options {
		switch strings.ToUpper(option.Name) {
		case "FORMAT", "FMT":
			err = device.context.SetFormat(option.EqualOpt)
			if err != nil {
				return errors.New("invalid Tape format type: " + option.EqualOpt)
			}

		case "-r", "RO", "NORING":
			device.context.SetNoRing()

		case "-rw", "RW", "RING":
			device.context.SetRing()

		case "7TRACK":
			device.context.Set7Track()
			device.seven = true

		case "9TRACK":
			device.context.Set9Track()

		case "FILE":
			if option.EqualOpt == "" {
				return errors.New("file option missing filename")
			}
			err := device.context.Attach(option.EqualOpt)
			if err != nil {
				return err
			}

		default:
			return errors.New("2400 invalid option " + option.Name)
		}
		if option.Value != nil {
			return errors.New("extra options not supported on: " + option.Name)
		}
	}
	return nil
}
