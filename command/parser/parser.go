/*
 * S370 - Command parser.
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
	"strings"
	"unicode"

	command "github.com/rcornwell/S370/command/command"
	core "github.com/rcornwell/S370/emu/core"
	ch "github.com/rcornwell/S370/emu/sys_channel"
)

type cmd struct {
	Name     string // Command name.
	Min      int    // Minimum match size.
	Process  func(*cmdLine, *core.Core) (bool, error)
	Complete func(*cmdLine) []string
}

type cmdLine struct {
	line string // Current command.
	pos  int    // Position in line.
}

// Execute the command line given.
func ProcessCommand(commandLine string, core *core.Core) (bool, error) {
	line := cmdLine{line: commandLine}
	command := line.getWord(false)
	if command == "" {
		return false, nil
	}

	match := matchList(command)
	if len(match) == 0 {
		return false, errors.New("command not found: " + command)
	}

	if len(match) > 1 {
		return false, errors.New("unique command not found: " + command)
	}

	return match[0].Process(&line, core)
}

// Check if command matches at least to minimum length.
func matchCommand(match cmd, command string) bool {
	l := 0
	for l = range len(command) {
		if match.Name[l] != command[l] {
			return false
		}
	}
	return (l + 1) >= match.Min
}

// Check if command matches one of the commands.
func matchList(command string) []cmd {
	// If command empty just return.
	if command == "" {
		return []cmd{}
	}

	// Try and match one command.
	var match []cmd
	for _, m := range cmdList {
		if matchCommand(m, command) {
			match = append(match, m)
		}
	}
	return match
}

// Match list of options.
func matchOption(option string, optList []command.Options, cmdType int) command.Options {
	for _, opt := range optList {
		if (opt.OptionValid & cmdType) == 0 {
			continue
		}
		if opt.Name == option {
			return opt
		}
	}
	return command.Options{OptionType: -1}
}

// Skip forward over line until none whitespace character found.
func (line *cmdLine) skipSpace() {
	for {
		if line.pos >= len(line.line) {
			return
		}
		if unicode.IsSpace(rune(line.line[line.pos])) {
			line.pos++
			continue
		}
		return
	}
}

// Check if at end of line.
func (line *cmdLine) isEOL() bool {
	if line.pos >= len(line.line) {
		return true
	}

	if line.line[line.pos] == '#' {
		return true
	}
	return false
}

// Return next letter or digit in line. 0 if EOL or space.
func (line *cmdLine) getNext() byte {
	line.pos++
	if line.isEOL() {
		return 0
	}
	return line.line[line.pos]
}

// Return current digit and advance to next.
func (line *cmdLine) getCurrent() byte {
	if line.isEOL() {
		return 0
	}
	by := line.line[line.pos]
	line.pos++
	return by
}

// // Peek at next character.
// func (line *cmdLine) peekNext() byte {
// 	if (line.pos + 1) >= len(line.line) {
// 		return 0
// 	}
// 	return line.line[line.pos+1]
// }

// Parse string that is "string" or just string.
func (line *cmdLine) parseQuoteString() (string, bool) {
	inQuote := false
	value := ""

	// If quote, set we are in quoted string
	by := line.getCurrent()
	if by == 0 {
		return "", false
	}

	if by == '"' {
		inQuote = true
		by = line.getCurrent()
	}

	for by != 0 {
		// If processing a quoted string "" gets replaced by signal quote
		if by == '"' && inQuote {
			by = line.getCurrent()
			// Single quote terminates string.
			if by != '"' {
				// Hit end of string.
				return value, true
			}
		}

		if inQuote {
			value += string(by)
		} else // Space terminates a no quoted string.
		if by != 0 && unicode.IsSpace(rune(by)) {
			return value, true
		}

		value += string(by)
		// If we hit end of line, stop processing.
		by = line.getCurrent()
	}
	return value, !inQuote
}

// Parse parse a number.
func (line *cmdLine) getNumber() (uint32, error) {
	line.skipSpace()

	// Check if end of line.
	if line.isEOL() {
		return 0, errors.New("not a number")
	}

	value := uint32(0)
	// Characters must be alphabetic
	by := line.getCurrent()
	for by != 0 {
		if !unicode.IsDigit(rune(by)) {
			return 0, errors.New("not a number")
		}
		value = (value * 10) + uint32(by-'0')
		by = line.getCurrent()
		if by != 0 && unicode.IsSpace(rune(by)) {
			break
		}
	}

	return value, nil
}

const hex = "0123456789abcdef"

// Parse hex number.
func (line *cmdLine) getHex() (uint32, error) {
	line.skipSpace()

	pos := line.pos
	value := uint32(0)
	// Characters must be alphabetic
	by := line.getCurrent()
	for by != 0 {
		digit := strings.Index(hex, strings.ToLower(string(by)))
		if digit == -1 {
			line.pos = pos
			return 0, errors.New("not a number")
		}
		value = (value << 4) + uint32(digit)
		by = line.getCurrent()
		if by != 0 && unicode.IsSpace(rune(by)) {
			break
		}
	}

	return value, nil
}

// Parse option name.
// Return string and whether last charcter was = or not.
func (line *cmdLine) getWord(equal bool) string {
	line.skipSpace()

	// Characters must be alphabetic
	value := ""
	pos := line.pos
	by := line.getCurrent()
	for by != 0 {
		if !unicode.IsLetter(rune(by)) {
			line.pos = pos
			return ""
		}
		value += string([]byte{by})
		by = line.getCurrent()
		if by != 0 && unicode.IsSpace(rune(by)) {
			break
		}
		if by == '=' && equal {
			return strings.ToLower(value)
		}
	}

	return strings.ToLower(value)
}

// Get an option.
func (line *cmdLine) getOption(opts []command.Options, cmdType int) (*command.CmdOption, error) {
	// Get a word, stoping at equal or space.
	name := line.getWord(true)

	// Get command interface
	opt := command.CmdOption{Name: name}

	if name == "" && !line.isEOL() {
		if cmdType == command.ValidAttach {
			// For attach commands, if there is a valid name, consider it a file name.
			file, ok := line.parseQuoteString()
			if !ok {
				return nil, errors.New("invalid option")
			}
			opt.Name = "file"
			opt.EqualOpt = file
		}
		return &opt, nil
	}

	match := matchOption(name, opts, cmdType)
	switch match.OptionType {
	case -1:
		return nil, errors.New("unknown option: " + name)
	case command.OptionSwitch:
		if !line.isEOL() && !unicode.IsSpace(rune(line.getCurrent())) {
			return nil, errors.New("switch options must be followed by separator: " + name)
		}
		return nil, errors.New("switch option can't have arguments: " + name)
	case command.OptionFile:
		file, ok := line.parseQuoteString()
		if !ok {
			return nil, errors.New("file name not valid: " + name)
		}
		opt.EqualOpt = file
	case command.OptionNumber:
		if line.getCurrent() != '=' {
			return nil, errors.New("number options must be followed by number: " + name)
		}
		num, err := line.getNumber()
		if err != nil {
			return nil, errors.New("number options must be followed by number: " + name)
		}
		opt.Value = num

	case command.OptionHex:
		if line.getCurrent() != '=' {
			return nil, errors.New("hex options must be followed by hexdecimal numbe: " + name)
		}
		num, err := line.getHex()
		if err != nil {
			return nil, errors.New("hex options must be followed by hexdecimal number: " + name)
		}
		opt.Value = num

	case command.OptionList:
		if line.getCurrent() != '=' {
			return nil, errors.New("number options must be followed by name: " + name)
		}
		listStr := line.getWord(false)
		if !line.isEOL() && !unicode.IsSpace(rune(line.getCurrent())) {
			return nil, errors.New("number options must be followed by name: " + name)
		}
		opt.EqualOpt = listStr
		for _, mod := range match.OptionList {
			if strings.ToLower(mod) == listStr {
				return &opt, nil
			}
		}
		return nil, errors.New("option not valid for type: " + name)
	default:
		return nil, errors.New("invalid option type: " + name)
	}
	return &opt, nil
}

// Get options for show commands.
func (line *cmdLine) getShowOptions(device command.Command) ([]*command.CmdOption, error) {
	optlist := []*command.CmdOption{}
	opts := device.Options("")
	_, err := line.getHex()
	if err != nil {
		return nil, err
	}
	if line.isEOL() {
		return nil, nil
	}

	for {
		name := line.getWord(false)

		if name == "" {
			break
		}
		if !line.isEOL() && !unicode.IsSpace(rune(line.getCurrent())) {
			return nil, errors.New("set command does not take modifies")
		}
		// Get a word, stoping at equal or space.
		match := matchOption(name, opts, command.ValidShow)
		if match.OptionType == -1 {
			return nil, errors.New("invalid option")
		}
		opt := command.CmdOption{Name: name}
		optlist = append(optlist, &opt)
	}
	return optlist, nil
}

// Scan options and return a list of options.
func (line *cmdLine) getOptions(device command.Command, cmdType int) ([]*command.CmdOption, error) {
	optlist := []*command.CmdOption{}
	opts := device.Options("")
	for {
		opt, err := line.getOption(opts, cmdType)
		if err != nil {
			return optlist, err
		}
		if opt != nil && opt.Name != "" {
			optlist = append(optlist, opt)
		} else {
			break
		}
	}
	return optlist, nil
}

// Return pointer to command interface to device.
func (line *cmdLine) getDevice() (command.Command, error) {
	// Get device number make sure it is valid.
	devNum, ok := line.getHex()
	if ok != nil {
		return nil, errors.New("device must be number ")
	}

	if devNum > 0xfff {
		return nil, errors.New("device number too large")
	}

	// Get pointer to device.
	return ch.GetCommand(uint16(devNum))
}
