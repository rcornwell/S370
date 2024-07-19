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
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode"

	command "github.com/rcornwell/S370/command/command"
	config "github.com/rcornwell/S370/config/configparser"
	core "github.com/rcornwell/S370/emu/core"
	ch "github.com/rcornwell/S370/emu/sys_channel"
)

type cmd struct {
	name     string // Command name.
	min      int    // Minimum match size.
	process  func(*cmdLine, *core.Core) (bool, error)
	complete func(*cmdLine) []string
}

type cmdLine struct {
	line string // Current command.
	pos  int    // Position in line.
}

var cmdList = []cmd{
	{name: "attach", min: 2, process: attach, complete: attachComplete},
	{name: "detach", min: 2, process: detach},
	{name: "set", min: 3, process: set, complete: setComplete},
	{name: "unset", min: 4, process: unset, complete: setComplete},
	{name: "quit", min: 4, process: quit},
	{name: "stop", min: 3, process: stop},
	{name: "continue", min: 1, process: cont},
	{name: "start", min: 3, process: start},
	{name: "show", min: 2, process: show},
	{name: "ipl", min: 1, process: ipl},
}

// Execute the command line given.
func ProcessCommand(commandLine string, core *core.Core) (bool, error) {
	line := cmdLine{line: commandLine}
	command := line.getWord(false)

	match := matchList(command)
	if len(match) == 0 {
		return false, errors.New("command not found: " + command)
	}

	if len(match) > 1 {
		return false, errors.New("unique command not found: " + command)
	}

	return match[0].process(&line, core)
}

// Called to complete a command line, during line editing.
func CompleteCmd(commandLine string) []string {
	line := cmdLine{line: commandLine}
	name := line.getWord(false)

	// We have a command, let it try and complete it.
	if !line.isEOL() && line.line[line.pos] == ' ' {
		// Skip leading spaces.
		line.skipSpace()
		// See if there is a completer for this command.
		match := matchList(name)
		if len(match) == 0 || len(match) > 1 {
			return nil
		}

		if match[0].complete != nil {
			return match[0].complete(&line)
		}
		return nil
	}

	matchList := matchList(name)
	matches := make([]string, len(matchList))
	for i, m := range matchList {
		matches[i] = m.name
	}

	return matches
}

// Check if command matches at least to minimum length.
func matchCommand(match cmd, command string) bool {
	l := 0
	for l = range len(command) {
		if match.name[l] != command[l] {
			return false
		}
	}
	return (l + 1) >= match.min
}

// Match for device address.
func matchDevice(appendStart bool, line cmdLine) []string {
	leading := ""
	device := ""
	pos := line.pos
	if appendStart {
		leading = line.line[:pos]
	}

	// Collect device.
	for pos < len(line.line) && line.line[pos] != ' ' && line.line[pos] != '#' {
		device += string(line.line[pos])
		pos++
	}

	devices := []string{}
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
		devices = append(devices, leading+str+" ")
	}
	return devices
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

// Peek at next character.
func (line *cmdLine) getPeek() byte {
	if (line.pos + 1) >= len(line.line) {
		return 0
	}
	return line.line[line.pos+1]
}

// Parse string that is "string" or just string.
func (line *cmdLine) parseQuoteString() (string, bool) {
	inQuote := false
	value := ""

	// If quote, set we are in quoted string
	if line.getPeek() == '"' {
		inQuote = true
		_ = line.getNext()
	}

	for {
		by := line.getNext()
		// If processing a quoted string "" gets replaced by signal quote
		if by == '"' && inQuote {
			by = line.getNext()
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
			return value, !inQuote
		}
	}
}

// Parse device number.
func (line *cmdLine) getDevNum() string {
	line.skipSpace()

	// Check if end of line.
	if line.isEOL() {
		return ""
	}

	// Characters must be alphabetic
	value := ""
	by := line.line[line.pos]
	for {
		if !unicode.IsLetter(rune(by)) && !unicode.IsDigit(rune(by)) {
			return ""
		}
		value += string([]byte{by})
		by = line.getNext()
		if line.isEOL() || unicode.IsSpace(rune(by)) {
			break
		}
	}

	return strings.ToLower(value)
}

// Parse option name.
func (line *cmdLine) getWord(equal bool) string {
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

// Get an option.
func (line *cmdLine) getOption(opts []command.Options, cmdType int) (*command.CmdOption, error) {
	name := line.getWord(true)
	// Get a word, stoping at equal or space.
	opt := command.CmdOption{Name: name}

	if name == "" {
		if cmdType == command.ValidAttach {
			// For attach commands, if there is a valid name, consider it a file name.
			if !line.isEOL() && !unicode.IsSpace(rune(line.line[line.pos])) {
				line.pos--
				file, ok := line.parseQuoteString()
				if !ok {
					return nil, errors.New("invalid option")
				}
				opt.Name = "file"
				opt.EqualOpt = file
			}
		}
		return &opt, nil
	}

	match := matchOption(name, opts, cmdType)
	switch match.OptionType {
	case -1:
		return nil, errors.New("unknown option: " + name)
	case command.OptionSwitch:
		if line.isEOL() || line.line[line.pos] != ' ' {
			break
		}
		return nil, errors.New("switch option can't have arguments: " + name)
	case command.OptionFile:
		file, ok := line.parseQuoteString()
		if !ok {
			return nil, errors.New("file name not valid: " + name)
		}
		opt.EqualOpt = file
	case command.OptionNumber:
		if line.isEOL() || line.line[line.pos] != '=' {
			return nil, errors.New("number options must be followed by number: " + name)
		}
		numStr := line.getWord(false)
		num, err := strconv.ParseUint(numStr, 10, 32)
		if err != nil {
			return nil, errors.New("number options must be followed by number: " + name)
		}
		opt.Value = int(num)
	case command.OptionList:
		if line.isEOL() || line.line[line.pos] != '=' {
			return nil, errors.New("number options must be followed by number: " + name)
		}
		// Skip equal sign.
		_ = line.getNext()
		listStr := line.getWord(false)
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
	for {
		name := line.getDevNum()
		if line.isEOL() {
			break
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

// Scan a option list element.
func (line *cmdLine) scanList() string {
	// Characters must be alphabetic
	value := ""
	for {
		if line.isEOL() {
			return strings.ToLower(value)
		}
		by := line.line[line.pos]
		if unicode.IsSpace(rune(by)) {
			return value
		}
		line.pos++
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
		str, skip = line.parseQuoteString()
	case command.OptionNumber:
		str = line.getWord(false)
		skip = str != ""
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
		name := line.getWord(true)

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

// Scan device style commands.
func (line *cmdLine) scanDevice(cmdType int) []string {
	devices := matchDevice(true, *line)
	if len(devices) != 1 {
		return devices
	}

	devName := line.getDevNum()
	list := []string{}
	devNum, ok := strconv.ParseUint(devName, 16, 12)
	if ok != nil {
		slog.Debug("Unable to convert " + devName + " " + ok.Error())
		return []string{}
	}

	// Get pointer to device.
	device, err := ch.GetCommand(uint16(devNum))
	if err != nil {
		slog.Debug("Unable to find device: " + devName + " error: " + err.Error())
		return list
	}

	return line.scanOptions(device, cmdType)
}

// // Skip over command name.
// func skipCmd(line string) string {
// 	var pos int
// 	// Skip leading spaces.
// 	for pos < len(line) {
// 		if line[pos] != ' ' {
// 			break
// 		}
// 		pos++
// 	}

// 	// Skip command.
// 	for pos < len(line) {
// 		if line[pos] == ' ' {
// 			break
// 		}
// 		pos++
// 	}

// 	// Skip trailing space.
// 	for pos < len(line) {
// 		if line[pos] != ' ' {
// 			break
// 		}
// 		pos++
// 	}
// 	return line[pos:]
// }

// Handle attach commands.
func attach(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Info("Command Attach")

	// Get device number make sure it is valid.
	devName := line.getDevNum()
	devNum, ok := strconv.ParseUint(devName, 16, 12)
	if ok != nil {
		return false, errors.New("attach device must be number: " + devName)
	}

	// Get pointer to device.
	device, err := ch.GetCommand(uint16(devNum))
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
	return false, device.Attach(optlist)
}

// Attach command completion.
func attachComplete(line *cmdLine) []string {
	return line.scanDevice(command.ValidAttach)
}

// Handle detach command.
func detach(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Info("Command Detach")

	// Get device number make sure it is valid.
	devName := line.getDevNum()
	devNum, ok := strconv.ParseUint(devName, 16, 12)
	if ok != nil {
		return false, errors.New("Attach device must be number: " + devName)
	}

	// Get pointer to device.
	device, err := ch.GetCommand(uint16(devNum))
	if err != nil {
		return false, err
	}
	return false, device.Detach()
}

// Handle set commands.
func set(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Info("Command Set")

	// Get device number make sure it is valid.
	devName := line.getDevNum()
	devNum, ok := strconv.ParseUint(devName, 16, 12)
	if ok != nil {
		return false, errors.New("set device must be number: " + devName)
	}

	// Get pointer to device.
	device, err := ch.GetCommand(uint16(devNum))
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
	slog.Info("Command Unset")

	// Get device number make sure it is valid.
	devName := line.getDevNum()
	devNum, ok := strconv.ParseUint(devName, 16, 12)
	if ok != nil {
		return false, errors.New("unset device must be number: " + devName)
	}

	// Get pointer to device.
	device, err := ch.GetCommand(uint16(devNum))
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
	slog.Info("Command Quit")
	return true, nil
}

// Stop the CPU.
func stop(_ *cmdLine, core *core.Core) (bool, error) {
	slog.Info("Command Stop")
	core.SendStop()
	return false, nil
}

// Continue CPU from where it left off.
func cont(_ *cmdLine, core *core.Core) (bool, error) {
	slog.Info("Command Continue")
	core.SendStart()
	return false, nil
}

// Start the CPU.
func start(_ *cmdLine, core *core.Core) (bool, error) {
	slog.Info("Command Start")
	core.SendStart()
	return false, nil
}

// Process the show command.
func show(line *cmdLine, _ *core.Core) (bool, error) {
	slog.Info("Command Show")
	// Get device number make sure it is valid.
	devName := line.getDevNum()
	if devName == "" && line.isEOL() {
		optList := []*command.CmdOption{}
		for _, devName = range config.ModelList {
			devNum, ok := strconv.ParseUint(devName, 16, 12)
			if ok != nil {
				continue
			}

			device, err := ch.GetCommand(uint16(devNum))
			if err != nil {
				continue
			}

			out, err := device.Show(optList)
			if err != nil {
				continue
			}
			fmt.Println(out)
		}
		return false, nil
	}

	devNum, ok := strconv.ParseUint(devName, 16, 12)
	if ok != nil {
		return false, errors.New("set device must be number: " + devName)
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

// IPL the simulator.
func ipl(line *cmdLine, core *core.Core) (bool, error) {
	slog.Info("Command IPL")
	// Get device number make sure it is valid.
	devName := line.getDevNum()
	devNum, ok := strconv.ParseUint(devName, 16, 12)
	if ok != nil {
		return false, errors.New("ipl device must be number: " + devName)
	}
	_, err := ch.GetDevice(uint16(devNum))
	if err != nil {
		return false, err
	}
	core.SendIPL(uint16(devNum))
	return false, nil
}
