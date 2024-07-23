/*
 * S370 - Command completion functions.
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
	"slices"
	"strconv"
	"strings"
	"unicode"

	command "github.com/rcornwell/S370/command/command"
	config "github.com/rcornwell/S370/config/configparser"
	ch "github.com/rcornwell/S370/emu/sys_channel"
)

// Called to complete a command line, during line editing.
func CompleteCmd(commandLine string) []string {
	line := cmdLine{line: commandLine}
	name := line.getWord(false)

	// We have a command, let it try and complete it.
	if !line.isEOL() && !unicode.IsSpace(rune(line.getCurrent())) {
		// See if there is a completer for this command.
		match := matchList(name)
		if len(match) == 0 || len(match) > 1 {
			return nil
		}

		if match[0].Complete != nil {
			return match[0].Complete(&line)
		}
		return nil
	}

	// Try and match one command.
	var matches []string
	for _, m := range cmdList {
		if strings.HasPrefix(m.Name, name) {
			matches = append(matches, m.Name)
		}
	}
	slices.Sort(matches)
	return matches
}

// Match for device address.
func (line *cmdLine) matchDevice(cmdType int, all bool) []string {
	device := ""
	pos := line.pos
	leading := line.line[:pos]

	// Collect device.
	for pos < len(line.line) {
		by := line.getCurrent()
		if by == 0 || unicode.IsSpace(rune(by)) {
			break
		}
		device += string(line.line[pos])
	}

	devices := []string{}
	line.pos = pos // Restore position before number
outer:
	for _, str := range config.ModelList {
		for j, by := range device {
			if j > len(str) {
				continue outer
			}
			if str[j] != byte(by) {
				continue outer
			}
		}

		if all {
			devices = append(devices, leading+str+" ")
			continue
		}

		devNum, _ := strconv.ParseUint(str, 16, 12)

		cmd, ok := ch.GetCommand(uint16(devNum))
		if ok != nil {
			continue
		}

		opts := cmd.Options("")
		valid := false
		for _, opt := range opts {
			if (opt.OptionValid & cmdType) != 0 {
				valid = true
				break
			}
		}

		if valid {
			devices = append(devices, leading+str+" ")
		}
	}
	return devices
}

// Scan a number.
func (line *cmdLine) scanNumber() (string, bool) {
	line.skipSpace()

	// Check if end of line.
	if line.isEOL() {
		return "", false
	}

	pos := line.pos
	// Characters must be numeric
	value := ""
	by := line.line[line.pos]
	for {
		if !unicode.IsDigit(rune(by)) {
			line.pos = pos
			return "", false
		}
		value += string([]byte{by})
		by = line.getNext()
		if line.isEOL() || unicode.IsSpace(rune(by)) {
			break
		}
	}

	return value, true
}

// Scan a hexnumber.
func (line *cmdLine) scanHex() (string, bool) {
	line.skipSpace()

	// Check if end of line.
	if line.isEOL() {
		return "", false
	}

	pos := line.pos
	// Characters must be numeric
	value := ""
	by := line.line[line.pos]
	for {
		if strings.Contains(hex, strings.ToLower(string(by))) {
			line.pos = pos
			return "", false
		}
		value += string([]byte{by})
		by = line.getNext()
		if line.isEOL() || unicode.IsSpace(rune(by)) {
			break
		}
	}

	return value, true
}

// Scan a word.
func (line *cmdLine) scanWord(equal bool) string {
	line.skipSpace()

	// Check if end of line.
	if line.isEOL() {
		return ""
	}

	pos := line.pos
	// Characters must be alphabetic
	value := ""
	by := line.line[line.pos]
	for {
		if !unicode.IsLetter(rune(by)) {
			line.pos = pos
			return ""
		}
		value += string([]byte{by})
		by = line.getNext()
		if line.isEOL() || unicode.IsSpace(rune(by)) {
			break
		}
		if by == '=' {
			if equal {
				break
			}
			line.pos = pos
			return ""
		}
	}

	return strings.ToLower(value)
}

// Parse string that is "string" or just string.
// Return true if terminated.
func (line *cmdLine) scanQuoteString() (string, bool) {
	inQuote := false
	value := ""

	by := line.getCurrent()
	// If quote, set we are in quoted string
	if by == '"' {
		inQuote = true
		by = line.getCurrent()
	}

	for by != 0 {
		// If processing a quoted string "" gets replaced by signal quote
		if by == '"' && inQuote {
			by = line.getCurrent()
			if by != '"' {
				// Hit end of string.
				return value, true
			}
		}

		space := unicode.IsSpace(rune(by))
		// Space terminates a no quoted string.
		if !inQuote && (space || by == 0) {
			return value, true
		}

		value += string(by)
		// If we hit end of line, stop processing.
		if line.isEOL() {
			break
		}
	}
	return value, !inQuote
}

// Scan a option list element.
func (line *cmdLine) scanList() string {
	// Characters must be alphabetic
	value := ""
	for {
		if line.isEOL() {
			return strings.ToLower(value)
		}
		by := line.getCurrent()
		if unicode.IsSpace(rune(by)) {
			return value
		}
		if !unicode.IsLetter(rune(by)) {
			return ""
		}
		value += string([]byte{by})
	}
}

// Scan a string for an option.
func scanOpt(name string, opts []command.Options, cmdType int) []command.Options {
	matches := []command.Options{}
	for _, opt := range opts {
		if (opt.OptionValid & cmdType) == 0 {
			continue
		}
		if opt.Name == name {
			matches = []command.Options{{Name: opt.Name, OptionType: opt.OptionType, OptionList: opt.OptionList}}
			return matches
		}

		if name == "" || strings.HasPrefix(opt.Name, name) {
			option := command.Options{Name: opt.Name, OptionType: opt.OptionType, OptionList: opt.OptionList}
			matches = append(matches, option)
		}
	}

	return matches
}

// Get an option.
func (line *cmdLine) scanOption(opt command.Options) ([]string, bool) {
	skip := false
	str := ""
	switch opt.OptionType {
	case command.OptionSwitch:

	case command.OptionFile:
		str, skip = line.scanQuoteString()

	case command.OptionNumber:
		str, skip = line.scanNumber()

	case command.OptionHex:
		str, skip = line.scanHex()

	case command.OptionList:
		modName := line.scanList()
		mods := []string{}
		for _, mod := range opt.OptionList {
			mod = strings.ToLower(mod)
			if modName == mod {
				return []string{mod + " "}, true
			}
			if modName == "" || strings.HasPrefix(mod, modName) {
				mods = append(mods, mod+" ")
			}
		}
		return mods, false
	}
	return []string{str}, skip
}

// Scan to find last option.
func (line *cmdLine) scanOptions(device command.Command, cmdType int) []string {
	opts := device.Options("")
	matches := []string{}
	for {
		line.skipSpace()
		leading := ""
		if line.pos == (len(line.line) - 1) {
			leading = line.line
		} else {
			leading = line.line[:line.pos]
		}
		name := line.scanWord(true)

		matchOpts := scanOpt(name, opts, cmdType)
		line.skipSpace()
		if len(matchOpts) > 1 {
			leading = line.line[:line.pos-len(name)]
			for _, opt := range matchOpts {
				matches = append(matches, leading+opt.Name)
			}
			return matches
		}
		eq := " "
		if matchOpts[0].OptionType != command.OptionSwitch {
			eq = "="
		}

		if matchOpts[0].Name != name {
			return []string{leading + matchOpts[0].Name + eq}
		}

		if matchOpts[0].OptionType != command.OptionSwitch {
			if line.pos == len(line.line) {
				line.line += eq
			}
			if line.line[line.pos] == eq[0] {
				line.pos++
			}
		}
		leading = line.line[:line.pos]
		// Scan rest of options until one not complete.
		optMatch, skip := line.scanOption(matchOpts[0])
		if !skip {
			for _, opt := range optMatch {
				matches = append(matches, leading+opt)
			}
			return matches
		}
	}
}

// Scan to find last option name.
func (line *cmdLine) scanOpts(device command.Command, cmdType int) []string {
	opts := device.Options("")
	matches := []string{}
	//	for {
	line.skipSpace()
	// leading := line.line
	// if line.pos < (len(line.line) - 1) {
	// 	leading = line.line[:line.pos]
	// }
	name := line.scanWord(true)

	matchOpts := scanOpt(name, opts, cmdType)
	line.skipSpace()
	if len(matchOpts) > 1 {
		leading := line.line[:line.pos-len(name)]
		for _, opt := range matchOpts {
			matches = append(matches, leading+opt.Name)
		}
		return matches
	}

	leading := line.line[:line.pos]
	for _, opt := range matchOpts {
		matches = append(matches, leading+opt.Name)
	}
	return matches
	// }
}

// Scan device style commands.
func (line *cmdLine) scanDevice(cmdType int) []string {
	devices := line.matchDevice(cmdType, false)
	if len(devices) != 1 {
		return devices
	}

	device, err := line.getDevice()
	if err != nil {
		return devices
	}

	return line.scanOptions(device, cmdType)
}

// Complete commands that only need device number.
func DeviceComplete(line *cmdLine) []string {
	return line.matchDevice(0, true)
}
