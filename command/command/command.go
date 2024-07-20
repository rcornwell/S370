/*
 * S370 - Command interface
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

package command

import "errors"

// List of options to pass to set or show function
type CmdOption struct {
	Name     string // Name of option.
	EqualOpt string // Value of string after =.
	Value    int    // Numberic value.
}

// List of option types.
const (
	OptionSwitch = 1 + iota
	OptionFile
	OptionNumber
	OptionHex
	OptionName
	OptionList
)

const (
	ValidAttach = 1 << iota
	ValidSet
	ValidShow
)

var (
	NotSupported = errors.New("command not supported")
	NoOption     = errors.New("option not supported")
	InvalidArg   = errors.New("invalid value for option")
	NotAttached  = errors.New("device not attached")
	Attached     = errors.New("device attached")
)

type Options struct {
	Name        string   // Name of option.
	OptionType  int      // Type of argument.
	OptionValid int      // Option valid for command type.
	OptionList  []string // List of valid options for this options.
}

type Command interface {
	Options(opt string) []Options              // Return list of supported options.
	Attach(options []*CmdOption) error         // Attach device to file.
	Detach() error                             // Detach a device.
	Set(set bool, options []*CmdOption) error  // Do set/ unset command.
	Show(options []*CmdOption) (string, error) // Do show command.
	Rewind() error                             // Rewind tape device.
	Reset() error                              // Reset device.
	GetAddr() uint16                           // Return device address.
}
