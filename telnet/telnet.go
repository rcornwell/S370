/*
 * S370 - telnet server
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

package telnet

import (
	"fmt"
	"net"
	"strings"

	D "github.com/rcornwell/S370/emu/device"
	"github.com/rcornwell/S370/emu/master"
)

// Telnet protocol constants - negatives are for init'ing signed char data

const (
	tnIAC     byte = 255 // protocol delim
	tnDONT    byte = 254 // dont
	tnDO      byte = 253 // do
	tnWONT    byte = 252 // wont
	tnWILL    byte = 251 // will
	tnSB      byte = 250 // Sub negotiations begin
	tnGA      byte = 249 // Go ahead
	tnIP      byte = 244 // Interrupt process
	tnBRK     byte = 243 // break
	tnSE      byte = 240 // Sub negotiations end
	tnIS      byte = 0
	tnSend    byte = 1
	tnInfo    byte = 2
	tnVar     byte = 0
	tnValue   byte = 1
	tnEsc     byte = 2
	tnUserVar byte = 3

	// Telnet line states.

	tnStateData int = 1 + iota // normal
	tnStateIAC                 // IAC seen
	tnStateWILL                // WILL seen
	tnStateDO                  // DO seen
	tnStateDONT                // DONT seen
	tnStateWONT                // WONT seen
	tnStateSKIP                // skip next cmd
	tnStateSB                  // Start of SB expect type
	tnStateSE                  // Waiting for SE
	tnStateSBIS                // Waiting for IS
	// tnStateWaitVar                // Wait for Var or Value.
	tnStateSBData // Data for SB until IS
	tnStateSTerm  // Grab terminal type
	// tnStateEnv                    // Grab environment type.

	// tnStateUser // Grab user name.

	// Telnet options.
	tnOptionBinary byte = 0  // Binary data transfer
	tnOptionEcho   byte = 1  // Echo
	tnOptionSGA    byte = 3  // Send Go Ahead
	tnOptionTerm   byte = 24 // Request Terminal Type
	tnOptionEOR    byte = 25 // Handle end of record
	tnOptionNAWS   byte = 31 // Negotiate about terminal size
	tnOptionLINE   byte = 34 // line mode
	tnOptionENV    byte = 39 // Environment

	// Telnet flags.
	tnFlagDo   uint8 = 0x01 // Do received
	tnFlagDont uint8 = 0x02 // Don't received
	tnFlagWill uint8 = 0x04 // Will received
	tnFlagWont uint8 = 0x08 // Wont received

)

// Interface for receiving telnet messages.
type Telnet interface {
	Connect(conn net.Conn)
	ReceiveChar(data []byte)
	Disconnect()
}

var initString = []byte{
	tnIAC, tnWONT, tnOptionLINE,
	tnIAC, tnWILL, tnOptionEcho,
	tnIAC, tnWILL, tnOptionSGA,
	tnIAC, tnWILL, tnOptionBinary,
	tnIAC, tnDO, tnOptionTerm,
}

// Convert option number to string.
func optName(opt byte) string {
	switch opt {
	case tnOptionBinary:
		return "bin"
	case tnOptionEcho:
		return "echo"
	case tnOptionSGA:
		return "sga"
	case tnOptionTerm:
		return "term"
	case tnOptionEOR:
		return "eor"
	case tnOptionNAWS:
		return "naws"
	case tnOptionLINE:
		return "line"
	case tnOptionENV:
		return "env"
	}
	return "unknown"
}

type tnState struct {
	optionState [256]uint8 // Current state of telnet session
	sbtype      byte       // Type of SB being received
	state       int        // Current line State
	model       byte       // Terminal model
	extatr      byte       // Extra type
	group       string     // Group user wants
	//	luname      []byte             // Current user name
	dev    Telnet             // Pointer to where to send data.
	devNum uint16             // Device address
	conn   net.Conn           // Client connection.
	master chan master.Packet // Pointer to channel to send messages to.
}

// Send a response to server, and log what we sent.
func (state *tnState) sendOption(setState, option byte) {
	data := []byte{tnIAC, setState, option}
	_, _ = state.conn.Write(data)
	switch setState {
	case tnWILL:
		state.optionState[option] |= tnFlagWill
	case tnWONT:
		state.optionState[option] |= tnFlagWont
	case tnDO:
		state.optionState[option] |= tnFlagDo
	case tnDONT:
		state.optionState[option] |= tnFlagDont
	}
}

// Handle DO response.
func (state *tnState) handleDO(input byte) {
	switch input {
	case tnOptionTerm:
		fmt.Println("Do Term")

	case tnOptionSGA:
		fmt.Println("Do SGA")
		if (state.optionState[input] & tnFlagWill) != 0 {
			//	if (state.optionState[input] & tnFlagDo) == 0 {
			state.optionState[input] |= tnFlagDont
			//		}
		}
	case tnOptionEcho:
		fmt.Println("Do Echo")
		if (state.optionState[input] & tnFlagWill) != 0 {
			//	if (state.optionState[input] & tnFlagDo) == 0 {
			state.optionState[input] |= tnFlagDont
			//		}
		}
	case tnOptionEOR:
		fmt.Println("Do EOR")
		state.optionState[input] |= tnFlagDo
	case tnOptionBinary:
		fmt.Println("Do Binary")
		if (state.optionState[input] & tnFlagDo) == 0 {
			state.sendOption(tnDO, input)
		}
	default:
		if (state.optionState[input] & tnFlagWont) == 0 {
			state.sendOption(tnWONT, input)
		}
	}
}

// Handle WILL response.
func (state *tnState) handleWILL(input byte) {
	switch input {
	case tnOptionTerm: // Collect option
		fmt.Println("Will Term")
		if (state.optionState[input] & tnFlagWill) == 0 {
			state.optionState[input] |= tnFlagWill
			fmt.Println("Send term request")
			send := []byte{tnIAC, tnSB, tnOptionTerm, tnSend, tnIAC, tnSE}
			_, err := state.conn.Write(send)
			if err != nil {
				fmt.Println("Send error: ", err)
			}
		}
	case tnOptionENV:
		if (state.optionState[input] & tnFlagWill) == 0 {
			state.optionState[input] |= tnFlagWill
			fmt.Println("Send env request")
			send := []byte{tnIAC, tnSB, tnOptionENV, tnSend, tnVar, 'U', 'S', 'E', 'R', tnIAC, tnSE}
			_, err := state.conn.Write(send)
			if err != nil {
				fmt.Println("Send error: ", err)
			}
		}
	case tnOptionEOR:
		fmt.Println("Will EOR")
		if (state.optionState[input] & tnFlagWill) == 0 {
			state.optionState[input] |= tnFlagWill
			//			state.sendOption(tnWILL, tnOptionBinary)
			//			state.optionState[tnOptionBinary] &= ^tnFlagWill
			//		state.sendOption(tnDO, tnOptionBinary)
		}
	case tnOptionSGA:
		fmt.Print("Will SGA")
		if (state.optionState[input] & tnFlagWill) == 0 {
			state.sendOption(tnDO, input)
			//		send := []byte{tnIAC, tnDO, input}
			//		_, err := state.conn.Write(send)
			//		if err != nil {
			//			fmt.Println("Send error: ", err)
			// }
		}
	case tnOptionEcho:
		fmt.Println("Will Echo")
		if (state.optionState[input] & tnFlagWill) == 0 {
			state.optionState[input] |= tnFlagWill
			state.sendOption(tnDONT, input)
			state.sendOption(tnWONT, input)
		}
	case tnOptionBinary:
		fmt.Println("Will Bin")
		if (state.optionState[input] & tnFlagWill) == 0 {
			state.optionState[input] |= tnFlagWill
			// Send clear screen to 3270 terminals
		}
	default:
		if (state.optionState[input] & tnFlagDont) == 0 {
			state.sendOption(tnDONT, input)
		}
	}
}

func (state *tnState) handleSE(term []byte) {
	if state.sbtype == tnOptionTerm {
		state.determineTerm(term)
		fmt.Printf("Terminal Model: %c ext: %c group: %s\n", state.model, state.extatr, state.group)
		if !state.findTerminal() {
			fmt.Fprintf(state.conn, "No matching terminal type found\n\r")
			state.conn.Close()
			return
		}
		state.SendConnect()
	}
}

// Handle client connection.
func handleClient(conn net.Conn, master chan master.Packet) {
	defer conn.Close()
	var out []byte

	state := tnState{conn: conn, state: tnStateData, devNum: D.NoDev}
	buffer := make([]byte, 1024)
	term := []byte{}
	state.master = master
	defer state.SendDisconnect()

	_, _ = state.conn.Write(initString)
	for {
		num, err := state.conn.Read(buffer)
		if err != nil {
			// Tell device we got an error
			fmt.Print("Error: ", err)
			return
		}
		out = []byte{}
		for i := range num {
			input := buffer[i]
			switch state.state {
			case tnStateData: // normal
				if input == tnIAC {
					state.state = tnStateIAC
					fmt.Println("data: IAC")
				} else {
					fmt.Printf("data: %02x %c\n", input, input)
					out = append(out, input)
				}
			// Otherwise send character to device.
			case tnStateIAC: // IAC seen
				switch input {
				case tnIAC:
					// Send character to device
					state.state = tnStateData
					fmt.Println("IAC")
				case tnBRK:
					state.state = tnStateData
					fmt.Println("BRK")
				case tnWILL:
					state.state = tnStateWILL
					fmt.Println("WILL")
				case tnWONT:
					state.state = tnStateWONT
					fmt.Println("WONT")
				case tnDO:
					state.state = tnStateDO
					fmt.Println("DO")
				case tnDONT:
					state.state = tnStateDONT
					fmt.Println("DONT")
				case tnSB:
					state.state = tnStateSB
					fmt.Println("SB")
				default:
					fmt.Printf("IAC Char: %02x\n", input)
					state.state = tnStateSKIP
				}

			case tnStateWILL: // WILL seen
				fmt.Printf("Will %s\n", optName(input))
				state.handleWILL(input)
				state.state = tnStateData

			case tnStateWONT: // WONT seen
				fmt.Printf("Wont %s\n", optName(input))
				if (state.optionState[input] & tnFlagWont) == 0 {
					state.sendOption(tnWONT, input)
				}
				state.state = tnStateData

			case tnStateDO: // DO seen
				fmt.Printf("Do %s\n", optName(input))
				state.handleDO(input)
				state.state = tnStateData

			case tnStateDONT:
				fmt.Printf("Dont %s\n", optName(input))
				state.state = tnStateData

			case tnStateSKIP: // skip next cmd
				fmt.Print("Skip")
				state.state = tnStateData

			case tnStateSB: // Start of SB expect type
				fmt.Printf("SB: %s\n", optName(input))
				state.sbtype = input
				state.state = tnStateSBIS

			case tnStateSBIS: // Waiting for IS
				fmt.Printf("SB IS %s\n", optName(state.sbtype))
				switch state.sbtype {
				case tnOptionTerm:
					state.state = tnStateSTerm
					//				case tnOptionENV:
					//					state.state = tnStateWaitVar
				default:
					state.state = tnStateSE
				}

			case tnStateSTerm:
				if input == tnIAC {
					state.state = tnStateSE
					fmt.Println("term type: ", string(term))
				} else {
					term = append(term, input)
				}
			// case tnStateWaitVar:
			// 	switch input {
			// 	case tnVar:
			// 		fmt.Println("VAR")
			// 		state.state = tnStateEnv
			// 	case tnValue:
			// 		fmt.Println("Value")
			// 		state.state = tnStateUser
			// 	case tnIAC:
			// 		fmt.Println("IAC")
			// 		state.state = tnStateSE
			// 	default:
			// 		fmt.Println("Input")
			// 		state.state = tnStateData
			// 	}
			// case tnStateEnv:
			// 	fmt.Printf("env: %02x %c\n", input, input)
			// 	switch input {
			// 	case tnIAC:
			// 		state.state = tnStateSE
			// 	case tnValue:
			// 		state.state = tnStateUser
			// 	default:
			// 	}
			// case tnStateUser:
			// 	fmt.Printf("user: %02x %c\n", input, input)
			// 	switch input {
			// 	case tnIAC:
			// 		state.state = tnStateSE
			// 		fmt.Println("user: ", string(state.luname))
			// 	case tnVar:
			// 		state.state = tnStateEnv
			// 		fmt.Println("var user: ", string(state.luname))
			// 	default:
			// 		state.luname = append(state.luname, input)
			// 	}
			case tnStateSE:
				if input == tnSE {
					state.state = tnStateData
					fmt.Println("SE")
					state.handleSE(term)
				}
			}
		}
		if len(out) != 0 {
			// sent to master.
			state.SendReceiveChar(out)
		}
	}
}

// Map of terminal types that support 3270 protocol.
var term = map[string]byte{
	"3277": '2', "3270": '2', "3178": '2', "3278": '2', "3179": '2', "3180": '2', "3287": '2', "3279": '2',
}

// Determine type of terminal.
func (state *tnState) determineTerm(termType []byte) {
	termStr := string(termType)
	i := strings.Index(termStr, "@")
	if i >= 0 {
		state.group = termStr[i+1:]
	}
	if termStr[0:4] == "IBM-" {
		state.model = term[termStr[4:8]]
		state.extatr = 'N'
		if termType[8] != '-' {
			return
		}
		if termType[9] < '1' || termType[9] > '5' {
			return // Model 0 is line mode terminal.
		}
		state.model = termType[9]
		if termStr[4:7] == "328" {
			state.model = '2'
		}
		if termStr[10:11] == "-E" {
			state.extatr = 'Y'
		}
	}
}
