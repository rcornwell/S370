/* IBM 360 Inquiry console.

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

package model1052

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/rcornwell/S370/command/command"
	config "github.com/rcornwell/S370/config/configparser"
	core "github.com/rcornwell/S370/emu/core"
	dev "github.com/rcornwell/S370/emu/device"
	ev "github.com/rcornwell/S370/emu/event"
	ch "github.com/rcornwell/S370/emu/sys_channel"
	"github.com/rcornwell/S370/telnet"
	"github.com/rcornwell/S370/util/debug"
	xlat "github.com/rcornwell/S370/util/xlat"
)

const (
	// Commands.
	cmdWrite    = 0x01 // Write to terminal
	cmdWriteACR = 0x09 // Write to terminal, returning carrage after
	cmdRead     = 0x0a // Read from terminal
	cmdAlarm    = 0x0b // Ring alarm bell
)

const (
	// Debug options.
	debugCmd    = 1 << iota // Log any commands.
	debugLine               // Log output by lines.
	debugDetail             // Low level details.
)

var debugOption = map[string]int{
	"CMD":    debugCmd,
	"LINE":   debugLine,
	"DETAIL": debugDetail,
}

type Model1052ctx struct {
	addr     uint16        // Current device address.
	col      int           // Current column.
	busy     bool          // Reader busy.
	halt     bool          // Signal halt requested.
	sense    uint8         // Current sense byte.
	read     bool          // Currently waiting on read.
	request  bool          // Console request.
	input    bool          // Input mode.
	output   bool          // Output mode.
	cr       bool          // Output CR.
	cancel   bool          // Cancel ^C pressed.
	inPtr    int           // Input pointer.
	inSize   int           // Size of input pending input.
	inBuff   [512]byte     // Place to save pending input.
	port     string        // Port number attached to.
	telctx   *model1052tel // Pointer to telnet device.
	debugMsk int           // Debug option mask.
	outLine  string        // Line being output for debug purposes.
}

type model1052tel struct {
	ctx       *Model1052ctx // Point to device context
	connected bool          // Connected to input
	conn      net.Conn      // Channel to write output to
}

// Handle start of CCW chain.
func (device *Model1052ctx) StartIO() uint8 {
	return 0
}

// Handle start of new command.
func (device *Model1052ctx) StartCmd(cmd uint8) uint8 {
	var r uint8
	var err error

	// If busy return busy status right away
	if device.busy {
		return dev.CStatusBusy
	}

	tel := device.telctx
	// Decode command
	switch cmd {
	case 0:
	case cmdRead:
		device.halt = false
		// If not connected, return Unit Check status.
		if !tel.connected {
			device.sense = dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}

		// If request pending, post it.
		if device.request {
			device.request = false
			return dev.CStatusAttn
		}

		// If not in input, print prompt
		if !device.input && (device.inPtr == 0 || device.cr) {
			// Active input so we can get response
			if device.output {
				// send \r\n
				out := []byte("\r\n")
				_, err = tel.conn.Write(out)
				if err != nil {
					fmt.Println("Telnet error: ", err)
				}
				device.output = false
			}
			// send 'I '
			out := []byte("I ")
			_, err = tel.conn.Write(out)
			if err != nil {
				fmt.Println("Telnet error: ", err)
			}
		}

		// Set up for read command.
		device.inPtr = 0
		device.sense = 0
		device.busy = true
		device.read = true
		ev.AddEvent(device, device.callback, 10, int(cmd))
		debug.DebugDevf(device.addr, device.debugMsk, debugCmd, "Cmd: %02x", cmd)

	case cmdWrite, cmdWriteACR:
		device.halt = false

		// If not connected return unit check.
		if !tel.connected {
			device.sense = dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}

		// If request pending, send attention
		if device.request {
			device.request = false
			return dev.CStatusAttn
		}

		// If last command put out a carrage return, put new reply notice.
		if device.cr {
			// send 'R '
			device.cr = false
			device.output = true
			out := []byte("R ")
			_, err = tel.conn.Write(out)
			if err != nil {
				fmt.Println("Telnet error: ", err)
			}
		}

		// Start device
		device.busy = true
		device.outLine = ""
		ev.AddEvent(device, device.callback, 10, int(cmd))
		debug.DebugDevf(device.addr, device.debugMsk, debugCmd, "Cmd: %02x", cmd)

	case dev.CmdSense:
		// Queue up sense command
		device.halt = false
		device.busy = true
		ev.AddEvent(device, device.callback, 10, int(cmd))
		debug.DebugDevf(device.addr, device.debugMsk, debugCmd, "Cmd: %02x", cmd)

	case cmdAlarm:
		device.halt = false

		// If not connected send unit check status
		if !tel.connected {
			device.sense = dev.SenseINTVENT
			return dev.CStatusChnEnd | dev.CStatusDevEnd | dev.CStatusCheck
		}

		// If request pending, sent attention.
		if device.request {
			device.request = false
			return dev.CStatusAttn
		}

		// Send '\b'
		r = dev.CStatusChnEnd
		device.sense = 0
		device.busy = true
		ev.AddEvent(device, device.callback, 1000, int(cmd))
		debug.DebugDevf(device.addr, device.debugMsk, debugCmd, "Cmd: %02x", cmd)

	case dev.CmdCTL:
		r = dev.CStatusChnEnd | dev.CStatusDevEnd
		debug.DebugDevf(device.addr, device.debugMsk, debugCmd, "Cmd: %02x", cmd)
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
func (device *Model1052ctx) HaltIO() uint8 {
	device.halt = true
	return 1
}

// Initialize a device.
func (device *Model1052ctx) InitDev() uint8 {
	device.col = 0
	device.sense = 0
	device.busy = false
	device.halt = false
	return 0
}

// Shutdown device.
func (device *Model1052ctx) Shutdown() {
}

// Enable debug options.
func (device *Model1052ctx) Debug(opt string) error {
	flag, ok := debugOption[opt]
	if !ok {
		return errors.New("1052 debug option invalid: " + opt)
	}
	device.debugMsk |= flag
	return nil
}

// List of valid options.
func (device *Model1052ctx) Options(_ string) []command.Options {
	return []command.Options{}
}

// Attach file to device.
func (device *Model1052ctx) Attach(_ []*command.CmdOption) error {
	return errors.New("attach command not supported")
}

// Detach device.
func (device *Model1052ctx) Detach() error {
	return errors.New("detach command not supported")
}

// Set command.
func (device *Model1052ctx) Set(_ bool, _ []*command.CmdOption) error {
	return errors.New("set command not supported")
}

// Show command.
func (device *Model1052ctx) Show(_ []*command.CmdOption) (string, error) {
	str := fmt.Sprintf("%03x: port=%s", device.addr, device.port)
	if device.telctx.connected {
		str += " connected"
	}
	return str, nil
}

// Finish write command.
func (device *Model1052ctx) finishWrite() {
	line := ""
	for i := range device.inSize {
		line += string(xlat.EBCDICToASCII[device.inBuff[i]])
	}
	debug.DebugDevf(device.addr, device.debugMsk, debugLine, "Send: %s", line)
	device.input = false
	device.inPtr = 0
	device.inSize = 0
	if device.cancel {
		device.cancel = false
		ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd|dev.CStatusExpt)
	} else {
		ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd)
	}
	device.read = false
	device.busy = false
}

// Handle channel operations.
func (device *Model1052ctx) callback(cmd int) {
	var err error

	tel := device.telctx
	switch uint8(cmd) {
	case dev.CmdSense:
		device.busy = false
		device.halt = false
		_ = ch.ChanWriteByte(device.addr, device.sense)
		ch.ChanEnd(device.addr, (dev.CStatusChnEnd | dev.CStatusDevEnd))
		return

	case cmdAlarm:
		device.busy = false
		ch.SetDevAttn(device.addr, dev.CStatusDevEnd)
		return

	case cmdWrite, cmdWriteACR:
		by, end := ch.ChanReadByte(device.addr)
		//	fmt.Printf("Write character %02x\n", by)
		if end {
			if uint8(cmd) == cmdWriteACR {
				// send \r\n
				out := []byte("\r\n")
				_, err = tel.conn.Write(out)
				if err != nil {
					fmt.Println("Telnet error: ", err)
				}
				device.cr = true
				device.output = false
			}
			device.busy = false
			debug.DebugDevf(device.addr, device.debugMsk, debugLine, "Output: %s", device.outLine)
			ch.ChanEnd(device.addr, dev.CStatusChnEnd|dev.CStatusDevEnd)
			return
		}
		if by == 0x15 {
			out := []byte("\r\n")
			_, err = tel.conn.Write(out)
			if err != nil {
				fmt.Println("Telnet error: ", err)
			}
			device.cr = true
			debug.DebugDevf(device.addr, device.debugMsk, debugLine, "Output: %s", device.outLine)
			device.outLine = ""
		} else {
			out := xlat.EBCDICToASCII[by]
			if out != 0 {
				if !strconv.IsPrint(rune(out)) {
					out = '_'
				}
				device.outLine += string(out)
				// send out
				_, err = tel.conn.Write([]byte{out})
				if err != nil {
					fmt.Println("Telnet error: ", err)
				}
				device.output = false
			}
		}

	case cmdRead:
		if !device.input {
			break
		}

		//	fmt.Printf("Terminal send line\n")
		device.request = false
		// Check for empty line, or end of data
		if device.inSize == 0 || device.inPtr == device.inSize {
			device.finishWrite()
			return
		}
		// Grab next character to send to CPU
		by := device.inBuff[device.inPtr]
		device.inPtr++
		end := ch.ChanWriteByte(device.addr, by)
		if end {
			device.finishWrite()
			return
		}
	}
	ev.AddEvent(device, device.callback, 1000, cmd)
}

// Connect to new terminal.
func (telConn *model1052tel) Connect(conn net.Conn) {
	telConn.connected = true
	telConn.conn = conn
}

// Disconnect from connection.
func (telConn *model1052tel) Disconnect() {
	telConn.connected = false
	telConn.conn = nil
}

// Input send from telnet process.
func (telConn *model1052tel) ReceiveChar(data []byte) {
	var err error
	device := telConn.ctx
	for _, by := range data {
		//	fmt.Printf("Terminal got: %02x\n", by)
		if !device.input {
			switch by {
			case '\r', '\n':
				debug.DebugDevf(device.addr, device.debugMsk, debugDetail, "End of record\n")
				device.input = true // Have input
				device.cr = true    // Received a carrage return.
				device.output = false
				device.inSize = device.inPtr
				device.inPtr = 0
				out := []byte("\r\n")
				_, err = telConn.conn.Write(out)
				if err != nil {
					fmt.Println("Telnet error: ", err)
				}
				fallthrough

			case 0o033: // Esc, request key
				debug.DebugDevf(device.addr, device.debugMsk, debugDetail, "Terminal request\n")
				if !device.read {
					device.request = true
				}

			case 0o177, '\b':
				if device.inPtr != 0 {
					device.inPtr--
					out := []byte("\b \b")
					_, err = telConn.conn.Write(out)
					if err != nil {
						fmt.Println("Telnet error: ", err)
					}
				}

			case 0o030: // ^X set external interrupt
				core.PostExtIrq()

			case 0o003: // ^C set cancel.
				device.input = true
				device.cancel = true
				device.output = false
				device.inSize = device.inPtr
				device.inPtr = 0
				if !device.read {
					device.request = true
				}

			case 0o025: // ^U clear input line
				out := []byte("\b \b")
				for device.inPtr > 0 {
					device.inPtr--
					_, err = telConn.conn.Write(out)
					if err != nil {
						fmt.Println("Telnet error: ", err)
					}
				}

			default:
				if device.inPtr < len(device.inBuff) {
					inChar := xlat.ASCIIToEBCDIC[by]
					if inChar == 0xff {
						_, err = telConn.conn.Write([]byte{'\007'})
						if err != nil {
							fmt.Println("Telnet error: ", err)
						}
					} else {
						// Convert back to ascii
						replyChar := xlat.EBCDICToASCII[inChar]
						// send out
						device.inBuff[device.inPtr] = inChar
						device.inPtr++
						_, err = telConn.conn.Write([]byte{replyChar})
						if err != nil {
							fmt.Println("Telnet error: ", err)
						}
					}
				}
			}
		} else {
			if device.read && by == 0o003 { // Cancel
				device.input = true
				device.inPtr = 0
				device.inSize = 0
				device.cancel = true
			} else {
				if by == 0o030 { // ^X Post external interrupt
					// Post external interrupt
					core.PostExtIrq()
				} else if !device.read {
					device.request = true
					_, err = telConn.conn.Write([]byte{'\007'})
					if err != nil {
						fmt.Println("Telnet error: ", err)
					}
				}
			}
		}
	}
	//	fmt.Printf("Done processing input: %t request: %t\n", device.busy, device.request)
	if !device.busy && device.request {
		ch.SetDevAttn(device.addr, dev.CStatusAttn)
		device.request = false
	}
}

// register a device on initialize.
func init() {
	config.RegisterModel("1052", config.TypeModel, create)
}

// Create a device.
func create(devNum uint16, _ string, options []config.Option) error {
	dev := Model1052ctx{addr: devNum}
	err := ch.AddDevice(&dev, &dev, devNum)
	if err != nil {
		return fmt.Errorf("unable to create 1052 at %03x", devNum)
	}
	console := model1052tel{ctx: &dev}
	dev.telctx = &console
	port := ""
	group := ""
	for _, option := range options {
		if option.EqualOpt != "" {
			return errors.New("equal option not supported on: " + option.Name)
		}
		_, err := strconv.ParseUint(option.Name, 10, 32)
		if err != nil { // If not number, assume group.
			if group != "" {
				return errors.New("only one group allowed: " + group)
			}
			group = option.Name
		} else { // Port number
			if port != "" {
				return errors.New("only one group allowed: " + group)
			}
			port = option.Name
		}

		if option.Value != nil {
			return errors.New("extra options not supported on: " + option.Name)
		}
	}

	ch.SetTelnet(&console, devNum)
	console.ctx.port = port
	return telnet.RegisterTerminal(&console, devNum, 0, port, group)
}
