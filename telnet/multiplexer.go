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
	"errors"
	"fmt"
	"strconv"
	"sync"

	config "github.com/rcornwell/S370/config/configparser"
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
	dev    Telnet // Device pointer
	devNum uint16 // Device number
	model  byte   // Device model (0 = line mode)
	port   string // Port device is listening on.
	group  string // Group device belongs to.
	inUse  bool   // Device is in use.
}

type portMap struct {
	port    string     // Port to connect to.
	group   string     // Group these ports belong too.
	devices []*termMap // List of devices on this port
}

var mapLock sync.Mutex

var terminals = map[uint16]*termMap{}

var ports = map[string][]*portMap{}

var groups = map[string]string{}

var defaultPort string

// Send connection message.
func (state *tnState) SendConnect() {
	packet := master.Packet{DevNum: state.devNum, Msg: master.TelConnect, Conn: state.conn}
	state.master <- packet
}

// Send disconnect message.
func (state *tnState) SendDisconnect() {
	packet := master.Packet{DevNum: state.devNum, Msg: master.TelDisconnect}
	state.master <- packet
	fmt.Printf("Device: %03x disconnected\n", state.devNum)
	term := terminals[state.devNum]
	term.inUse = false
	state.devNum = 0
}

// Send receive strings.
func (state *tnState) SendReceiveChar(data []byte) {
	packet := master.Packet{DevNum: state.devNum, Msg: master.TelReceive, Data: data}
	state.master <- packet
}

// Register a device of type.
func RegisterTerminal(dev Telnet, devNum uint16, model byte, port string, group string) error {
	// No need to lock map here since this will be used during configuration
	// Also should be no duplicates sent here.
	terminals[devNum] = &termMap{dev: dev, devNum: devNum, model: model, port: port, group: group}

	// If we have port, use it. Otherwise use default
	if port == "" {
		// See if maybe a group given.
		if group != "" {
			grpPort, ok := groups[group]
			if ok {
				port = grpPort // Use group port.
			}
		}
		// Still no port use default
		if port == "" {
			port = defaultPort
		}
	}

	if port == "" {
		return errors.New("no port specified and no default port")
	}

	// Register this port.
	pm := registerPort(port, group)
	if pm == nil {
		return errors.New("duplicate group found")
	}
	pm.devices = append(pm.devices, terminals[devNum])

	if pm.group != "" {
		fmt.Printf("Registering %03x on port: %s group: %s\n", devNum, pm.port, pm.group)
	} else {
		fmt.Printf("Registering %03x on port: %s no group\n", devNum, pm.port)
	}
	return nil
}

// Find terminal to connect to.
func (state *tnState) findTerminal() bool {
	// Lock the terminal map before searching it.
	mapLock.Lock()
	defer mapLock.Unlock()      // Make sure we unlock it
	pm, ok := ports[state.port] // See if exists.
	if !ok {
		fmt.Println("Connection from unregistered port: " + state.port)
		return false
	}

	if state.group != "" {
		devNum, err := strconv.ParseUint(state.group, 16, 16)
		// If hex number, see if port is available and matches type.
		if err == nil {
			term := terminals[uint16(devNum)]
			if term.inUse {
				fmt.Println("Terminal already in use")
				return false
			}
			if term.model != state.model {
				fmt.Printf("Terminal types don't match")
				return false
			}
			state.dev = term.dev
			state.devNum = term.devNum
			term.inUse = true
			return true
		}
		// Have a group not a device number.
		for _, pmap := range pm {
			if pmap.group != state.group {
				continue
			}
			for _, term := range pmap.devices {
				if term.inUse || term.model != state.model {
					continue
				}
				state.devNum = term.devNum
				state.dev = term.dev
				term.inUse = true
				return true
			}
		}
	}

	// Did not find matching device in group.
	for _, pmap := range pm {
		for _, term := range pmap.devices {
			if term.inUse {
				continue
			}
			if term.model != state.model {
				continue
			}
			state.devNum = term.devNum
			state.dev = term.dev
			term.inUse = true
			return true
		}
	}
	return false
}

// Register a port and group.
func registerPort(port string, group string) *portMap {
	// See if group exists.
	groupPort, okgrp := groups[group]
	if okgrp {
		if port != "" && port != groupPort {
			fmt.Printf("Duplicate group found on another port: " + groupPort)
			return nil
		}
	}

	// If it does not exist, find port with no group.
	pm, ok := ports[port] // See if exists.
	if !ok {
		fmt.Printf("Registering port: %s group: %s\n", port, group)
		newmap := &portMap{port: port, group: group}
		ports[port] = append(ports[port], newmap)
		if group != "" {
			groups[group] = port
		}
		return newmap
	}

	// Exists, scan down list of maps to see if this one already exists.
	if group != "" {
		for _, m := range pm {
			if m.group == group {
				return m
			}
		}
	}

	// We did not find one, append a new one.
	newmap := &portMap{port: port, group: group}
	ports[port] = append(ports[port], newmap)
	return newmap
}

// register a device on initialize.
func init() {
	config.RegisterModel("PORT", config.TypeOptions, setPort)
}

// Set default port.
func setPort(_ uint16, port string, options []config.Option) error {
	group := ""
	_, err := strconv.ParseUint(port, 10, 32)
	if err != nil {
		return fmt.Errorf("port requires number: %s", port)
	}
	if len(options) == 1 {
		if options[1].EqualOpt != "" || len(options[1].Value) != 0 {
			return errors.New("group name does not take options")
		}
	} else if len(options) != 0 {
		return errors.New("port only takes an optional group name")
	}
	_ = registerPort(port, group)
	if group == "" {
		if defaultPort != "" {
			return errors.New("can't have more then one default port")
		}
		defaultPort = port
	}
	return nil
}
