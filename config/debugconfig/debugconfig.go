/*
 * S370 - Debug options configuration.
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

package debugconfig

import (
	"errors"
	"strconv"
	"strings"

	config "github.com/rcornwell/S370/config/configparser"
	"github.com/rcornwell/S370/emu/cpu"
	dev "github.com/rcornwell/S370/emu/device"
	ch "github.com/rcornwell/S370/emu/sys_channel"
	"github.com/rcornwell/S370/util/tape"
)

// register a device on initialize.
func init() {
	config.RegisterModel("DEBUG", config.TypeOptions, setDebug)
}

// Set default port.
func setDebug(devNum uint16, device string, options []config.Option) error {
	switch strings.ToUpper(device) {
	case "CHANNEL":
		// Process Channel debug options

		if len(options) < 1 {
			return errors.New("debug channel requires a number first")
		}
		number := uint64(0)
		for i, opt := range options {
			if i == 0 {
				if options[0].EqualOpt != "" || len(options[0].Value) != 0 {
					return errors.New("debug channel number can't have equals or values")
				}
				var err error
				number, err = strconv.ParseUint(options[0].Name, 10, 4)
				if err != nil {
					return errors.New("channel number must be a number: " + options[0].Name)
				}
				continue
			}
			err := ch.Debug(int(number), strings.ToUpper(opt.Name))
			if err != nil {
				return err
			}
			if len(opt.Value) != 0 {
				for _, value := range opt.Value {
					err = ch.Debug(int(number), strings.ToUpper(*value))
					if err != nil {
						return err
					}
				}
			}
		}

	case "CPU":
		// Process CPU debug options
		for _, opt := range options {
			err := cpu.Debug(strings.ToUpper(opt.Name))
			if err != nil {
				return err
			}
			if len(opt.Value) != 0 {
				for _, value := range opt.Value {
					err = cpu.Debug(strings.ToUpper(*value))
					if err != nil {
						return err
					}
				}
			}
		}

	case "TAPE":
		// Process tape debug options
		for _, opt := range options {
			err := tape.Debug(strings.ToUpper(opt.Name))
			if err != nil {
				return err
			}
			if len(opt.Value) != 0 {
				for _, value := range opt.Value {
					err = tape.Debug(strings.ToUpper(*value))
					if err != nil {
						return err
					}
				}
			}
		}

	default:
		if devNum == dev.NoDev {
			return errors.New("debug option invalid: " + device)
		}
		dev, err := ch.GetDevice(devNum)
		if err != nil {
			return err
		}

		for _, opt := range options {
			err := dev.Debug(strings.ToUpper(opt.Name))
			if err != nil {
				return err
			}
			if len(opt.Value) != 0 {
				for _, value := range opt.Value {
					err = dev.Debug(strings.ToUpper(*value))
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
