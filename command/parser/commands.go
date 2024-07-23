/*
 * S370 - Command executer.
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

package parser

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	command "github.com/rcornwell/S370/command/command"
	config "github.com/rcornwell/S370/config/configparser"
	core "github.com/rcornwell/S370/emu/core"
	ch "github.com/rcornwell/S370/emu/sys_channel"
)

var cmdList = []cmd{
	{Name: "attach", Min: 2, Process: attach, Complete: attachComplete},
	{Name: "detach", Min: 2, Process: detach, Complete: func(line *cmdLine) []string {
		return line.matchDevice(command.ValidAttach, false)
	}},
	{Name: "set", Min: 3, Process: set, Complete: setComplete},
	{Name: "unset", Min: 4, Process: unset, Complete: setComplete},
	{Name: "quit", Min: 4, Process: quit},
	{Name: "stop", Min: 3, Process: stop},
	{Name: "continue", Min: 1, Process: cont},
	{Name: "start", Min: 3, Process: start},
	{Name: "show", Min: 2, Process: show, Complete: showComplete},
	{Name: "examine", Min: 2, Process: examine},
	{Name: "deposit", Min: 2, Process: deposit},
	{Name: "ipl", Min: 1, Process: ipl, Complete: func(line *cmdLine) []string {
		return line.matchDevice(command.ValidIPL, false)
	}},
	{Name: "rewind", Min: 3, Process: rewind, Complete: func(line *cmdLine) []string {
		return line.matchDevice(command.ValidRewind, false)
	}},
	{Name: "reset", Min: 5, Process: reset, Complete: DeviceComplete},
}

// Handle attach commands.
func attach(line *cmdLine, core *core.Core) (bool, error) {
	slog.Debug("Command Attach")

	// Get device number make sure it is valid.
	device, err := line.getDevice()
	if err != nil {
		return false, err
	}

	optlist, err := line.getOptions(device, command.ValidAttach)
	if err != nil {
		return false, err
	}
	if len(optlist) == 0 {
		return false, errors.New("no options give to attach command")
	}
	err = device.Attach(optlist)
	if err != nil {
		return false, err
	}

	core.SendDeviceEnd(device.GetAddr())
	return true, nil
}

// Attach command completion.
func attachComplete(line *cmdLine) []string {
	return line.scanDevice(command.ValidAttach)
}

// Handle detach command.
func detach(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Debug("Command Detach")

	// Get device number make sure it is valid.
	device, err := line.getDevice()
	if err != nil {
		return false, err
	}
	return false, device.Detach()
}

// Handle set commands.
func set(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Debug("Command Set")

	// Get device number make sure it is valid.
	device, err := line.getDevice()
	if err != nil {
		return false, err
	}

	optlist, err := line.getOptions(device, command.ValidSet)
	if err != nil {
		return false, err
	}
	if len(optlist) == 0 {
		return false, errors.New("no options give to set command")
	}
	return false, device.Set(false, optlist)
}

// Set/Unset command completion.
func setComplete(line *cmdLine) []string {
	return line.scanDevice(command.ValidSet)
}

// Handle unset commands.
func unset(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Debug("Command Unset")

	// Get device number make sure it is valid.
	device, err := line.getDevice()
	if err != nil {
		return false, err
	}

	optlist, err := line.getOptions(device, command.ValidSet)
	if err != nil {
		return false, err
	}
	if len(optlist) == 0 {
		return false, errors.New("no options give to unset command")
	}
	return false, device.Set(true, optlist)
}

// Handle commands that quit simulation.
func quit(_ *cmdLine, _ *core.Core) (bool, error) {
	slog.Debug("Command Quit")
	return true, nil
}

// Stop the CPU.
func stop(_ *cmdLine, core *core.Core) (bool, error) {
	slog.Debug("Command Stop")
	core.SendStop()
	return false, nil
}

// Continue CPU from where it left off.
func cont(_ *cmdLine, core *core.Core) (bool, error) {
	slog.Debug("Command Continue")
	core.SendStart()
	return false, nil
}

// Start the CPU.
func start(_ *cmdLine, core *core.Core) (bool, error) {
	slog.Debug("Command Start")
	core.SendStart()
	return false, nil
}

// Process the show command.
func show(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Debug("Command Show")
	// Get device number make sure it is valid.
	devNum, err := line.getHex()
	if err != nil || line.isEOL() {
		name := line.getWord(false)
		if name != "all" {
			return false, errors.New("set must be device number, empty or all")
		}

		optList := []*command.CmdOption{}
		for _, devName := range config.ModelList {
			devNum, ok := strconv.ParseUint(devName, 16, 12)
			if ok != nil {
				continue
			}

			device, nfnd := ch.GetCommand(uint16(devNum))
			if nfnd != nil {
				continue
			}

			out, noOpt := device.Show(optList)
			if noOpt != nil {
				continue
			}
			fmt.Println(out)
		}
		return false, nil
	}

	// Get pointer to device.
	device, err := ch.GetCommand(uint16(devNum))
	if err != nil {
		return false, err
	}

	optlist, err := line.getShowOptions(device)
	if err != nil {
		return false, err
	}

	out, err := device.Show(optlist)
	if err != nil {
		return false, err
	}

	fmt.Println(out)
	return false, nil
}

// Set/Unset command completion.
func showComplete(line *cmdLine) []string {
	devices := line.matchDevice(command.ValidShow, false)
	if len(devices) != 1 {
		return devices
	}

	device, err := line.getDevice()
	if err != nil {
		return devices
	}

	return line.scanOpts(device, command.ValidShow)
}

// IPL the simulator.
func ipl(line *cmdLine, core *core.Core) (bool, error) {
	slog.Debug("Command IPL")
	// Get device number make sure it is valid.
	device, err := line.getDevice()
	if err != nil {
		return false, err
	}
	core.SendIPL(device.GetAddr())
	return false, nil
}

// Rewind a Tape type device.
func rewind(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Debug("Command Rewind")
	device, err := line.getDevice()
	if err != nil {
		return false, err
	}
	return false, device.Rewind()
}

// Reset a device.
func reset(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Debug("Command Reset")
	// Get device number make sure it is valid.
	devNum, err := line.getHex()
	if err != nil || line.isEOL() {
		name := line.getWord(false)
		if name != "all" {
			return false, errors.New("set must be device number, empty or all")
		}

		// If no unit number of all reset all devices.
		for _, str := range config.ModelList {
			devNum, ok := strconv.ParseUint(str, 16, 12)
			if ok == nil {
				device, deverr := ch.GetCommand(uint16(devNum))
				if deverr == nil {
					_ = device.Reset()
				}
			}
		}
		return false, nil
	}

	device, err := ch.GetCommand(uint16(devNum))
	if err != nil {
		return false, err
	}
	return false, device.Reset()
}
