/*
 * S370 - telnet server, handle connection and link to device.
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
	"strconv"
	"sync"

	"github.com/rcornwell/S370/emu/master"
)

// const (
// 	TelConnect = 1 + iota
// 	TelRec
// 	TelDiscon
// 	TimeClock
// 	IPLdevice
// 	Shutdown
// )

// Data held in map of available connections.
type termMap struct {
	dev   Telnet // Device pointer
	model byte   // Device model (0 = line mode)
	group string // Group device belongs to.
	inUse bool   // Device is in use.
}

type portMap struct {
	port    string    // Port to connect to.
	group   string    // Group this port serves.
	devices []termMap // List of devices on this port
	server  *Server   // Server listening on this port
}

var mapLock sync.Mutex

var terminals = map[uint16]termMap{}

var ports = map[string]portMap{}

// Register a port and group.
func RegisterPort(group string, port string) {
	_, ok := ports[port] // See if exists.

	// If it does not exist, find port with no group.
	if !ok {
		ports[port] = portMap{port: port, group: group}
	}
}

// Register a device of type.
func RegisterTerminal(dev Telnet, devNum uint16, model byte, group string) {
	// No need to lock map here since this will be used during configuration
	// Also should be no duplicates sent here.
	terminals[devNum] = termMap{dev: dev, model: model, group: group}
	for _, p := range ports { // We can find it.
		if p.group == group {
			p.devices = append(p.devices, terminals[devNum])
			return
		}
	}
	fmt.Printf("no port found")
}

// Find a terminal by type.
func (state *tnState) findTerminalByType() bool {
	// Lock the terminal map before searching it.
	mapLock.Lock()
	defer mapLock.Unlock() // Make sure we unlock it
	for devNum, term := range terminals {
		if term.inUse {
			continue
		}
		if term.model != state.model {
			continue
		}
		// Found device
		state.dev = term.dev
		state.devNum = devNum
		term.inUse = true
		return true
	}
	fmt.Printf("Unable to find suitable terminal")
	return false
}

// Find a terminal by group name and type.
func (state *tnState) findTerminalByGroup() bool {
	// Lock the terminal map before searching it.
	mapLock.Lock()
	defer mapLock.Unlock() // Make sure we unlock it
	for devNum, term := range terminals {
		if term.inUse {
			continue
		}
		if term.group != state.group || term.model != state.model {
			continue
		}
		// Found device
		state.dev = term.dev
		state.devNum = devNum
		term.inUse = true
		return true
	}
	fmt.Printf("Unable to find suitable terminal")
	return false
}

// Find a terminal by device number.
func (state *tnState) findTerminalByDevice() bool {
	// Make sure valid hexdecimal number
	devNum, err := strconv.ParseUint(state.group, 16, 16)
	// Check if device number valid.
	if err != nil {
		fmt.Printf("Device %s is not proper device", state.group)
		return false
	}
	if devNum > 0x0fff {
		fmt.Printf("Device %s too large for device number", state.group)
		return false
	}

	// Lock the terminal map before searching it.
	mapLock.Lock()
	defer mapLock.Unlock() // Make sure we unlock it
	term, ok := terminals[uint16(devNum)]
	if !ok {
		fmt.Printf("Device %s is not a terminal", state.group)
		return false
	}
	if term.model != state.model {
		fmt.Printf("Device %s is not same type", state.group)
		return false
	}
	if term.inUse {
		fmt.Printf("Device %s is in use", state.group)
		return false
	}

	// Found device
	state.dev = term.dev
	state.devNum = uint16(devNum)
	term.inUse = true
	return true
}

// Find terminal to connect to.
func (state *tnState) findTerminal() bool {
	if state.group == "" {
		return state.findTerminalByType()
	}
	// Check if group is device number.
	if state.group[0] == '0' {
		return state.findTerminalByDevice()
	}
	return state.findTerminalByGroup()
}

// Send connection message.
func (state *tnState) SendConnect() {
	packet := master.Packet{DevNum: state.devNum, Msg: master.TelConnect, Conn: state.conn}
	state.master <- packet
}

// Send disconnect message.
func (state *tnState) SendDisconnect() {
	packet := master.Packet{DevNum: state.devNum, Msg: master.TelDisconnect}
	state.master <- packet
}

// Send receive strings.
func (state *tnState) SendReceiveChar(data []byte) {
	packet := master.Packet{DevNum: state.devNum, Msg: master.TelReceive, Data: data}
	state.master <- packet
}
