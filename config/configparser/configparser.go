/*
 * S370 - Configuration file parser
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

package configparser

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"unicode"

	D "github.com/rcornwell/S370/emu/device"
)

// List of options to pass to create routine.
type Option struct {
	Name     string    // Name of option.
	EqualOpt string    // Value of string after =.
	Value    []*string // Value of option.
}

// Model specification.
type modelName struct {
	model string // value of model.
	// slash byte   // Slash optional value.
	// dash  byte   // Dash optinal value.
}

// Option after model.
type FirstOption struct {
	devNum uint16 // Value of option if hex.
	isAddr bool   // Valid address in devNum
	value  string // String value of option.
}

// Current option line being parsed.
type optionLine struct {
	line string // Current option line.
	pos  int    // Current position in line.
}

/* Configuration file format:
 *
 * '#' indicates comment, rest of line is ignored.
 * <line> := <model> <whitespace> <address> <whitespace> <options> |
 *            'logfile' <quoteopt> |
 *            'log' <string> *(<commaopt>)
 * <model> := <string> ['-' <letter>|<number>] ['/' <letter>|<number>]
 * <address> ::= <string> | <hexnumber>| <number><K|M>
 * <options> ::= *(<option> *(<whitespace>))
 * <option> ::= *<value> (<whitespace> | <eol>
 * <value> ::= <opt> *(',' *(<whitespace>) <string>
 * <opt> := <valueopt> | <string>
 * <commaopt> ::= ',' *(<whitespace>) <string>
 * <optstring> ::= <string>
 * <optvalue> ::= <string>' =' <quoteopt>
 * <quoteopt> ::= <string> | '"' *(<letter> | <whitespace>) '"'
 * <string> ::= *(<letter> | <number>)
 */

const (
	TypeModel   = 1 + iota // Generic device.
	TypeDash               // Device accepts Dash option.
	TypeSlash              // Device accepts Slash option.
	TypeOption             // Accepts a option parameter.
	TypeOptions            // Accepts a list of options.
	TypeSwitch             // Option only used to set a flag.
	TypeFile               // Option is file name, including quotes.
)

// Model creation list.
type modelDef struct {
	create func(uint16, string, []Option) error
	ty     int
}

var models = map[string]modelDef{}

var ModelList []string

type SetOption struct {
	Name   string // Name of device.
	Option string // Option value.
}

var SetList []SetOption

var lineNumber int

// Return type of model or 0 if no model.
func getModel(mod string) int {
	model, ok := models[mod]
	if !ok {
		return 0
	}
	return model.ty
}

// Register should be called from init functions.
func RegisterModel(mod string, ty int, fn func(uint16, string, []Option) error) {
	mod = strings.ToUpper(mod)
	slog.Debug("Registering device: " + mod)
	model := modelDef{create: fn, ty: ty}
	models[mod] = model
}

// Register should be called from init functions.
func RegisterSwitch(mod string, fn func(uint16, string, []Option) error) {
	mod = strings.ToUpper(mod)
	slog.Debug("Registering switch: " + mod)
	model := modelDef{create: fn, ty: TypeSwitch}
	models[mod] = model
}

func RegisterSet(name string, option string) {
	SetList = append(SetList, SetOption{Name: name, Option: option})
}

// Register should be called from init functions.
func RegisterOption(mod string, fn func(uint16, string, []Option) error) {
	mod = strings.ToUpper(mod)
	slog.Debug("Registering simple option: " + mod)
	model := modelDef{create: fn, ty: TypeOption}
	models[mod] = model
}

// Register should be called from init functions.
func RegisterFile(mod string, fn func(uint16, string, []Option) error) {
	mod = strings.ToUpper(mod)
	slog.Debug("Registering file option: " + mod)
	model := modelDef{create: fn, ty: TypeFile}
	models[mod] = model
}

// Create a device of type model.
// func createModel(mod string, dash byte, slash byte, first *FirstOption, options []*Option) bool {.
func createModel(mod string, first *FirstOption, options []Option) error {
	mod = strings.ToUpper(mod)
	model, ok := models[mod]

	if !ok {
		return errors.New("Unknown model: " + mod)
	}

	if model.ty != TypeModel {
		return errors.New("Not a device type: " + mod)
	}

	if first.devNum == D.NoDev {
		return errors.New("Model: " + mod + " requires device number")
	}
	slog.Debug("Creating device " + mod + " number " + first.value)
	err := model.create(first.devNum, "", options)
	if err == nil {
		ModelList = append(ModelList, first.value)
	}
	return err
}

// Create a option with one parameter.
func createOption(mod string, first *FirstOption) error {
	mod = strings.ToUpper(mod)
	model, ok := models[mod]
	if !ok {
		return errors.New("Unknown option: " + mod)
	}
	if model.ty != TypeOption {
		return errors.New("Not a optional type: " + mod)
	}
	options := []Option{}
	if first.isAddr {
		return model.create(first.devNum, first.value, options)
	}
	return model.create(D.NoDev, first.value, options)
}

// Create a option with options.
func createOptions(mod string, first *FirstOption, options []Option) error {
	mod = strings.ToUpper(mod)
	model, ok := models[mod]
	if !ok {
		return errors.New("Unknown option: " + mod)
	}
	if model.ty != TypeOptions {
		return errors.New("Not a options type: " + mod)
	}
	if first.isAddr {
		return model.create(first.devNum, first.value, options)
	}
	return model.create(D.NoDev, first.value, options)
}

// Create switch option.
func createSwitch(mod string) error {
	mod = strings.ToUpper(mod)
	model, ok := models[mod]
	if !ok {
		return errors.New("Unknown switch: " + mod)
	}
	if model.ty != TypeSwitch {
		return errors.New("Not a switch type: " + mod)
	}
	return model.create(0, "", nil)
}

// Create file option.
func createFile(mod string, fileName string) error {
	mod = strings.ToUpper(mod)
	model, ok := models[mod]
	if !ok {
		return errors.New("Unknown file: " + mod)
	}
	if model.ty != TypeFile {
		return errors.New("Not a file type: " + mod)
	}
	return model.create(D.NoDev, fileName, nil)
}

// Load in a configuration file.
func LoadConfigFile(name string) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()

	lineNumber = 0

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	line := optionLine{}
	for scanner.Scan() {
		line.line += scanner.Text()
		line.pos = 0
		lineNumber++

		// If last character is \, append next line.
		if line.line != "" && line.line[len(line.line)-1:] == "\\" {
			line.line = line.line[:len(line.line)-1]
			continue
		}

		// We can skip blank lines and lines that begin with a #.
		if line.line == "" || line.line[0] == '#' {
			line.line = ""
			continue
		}
		msg := fmt.Sprintf("line %d: %s", lineNumber, line.line)
		slog.Debug(msg)
		err = line.parseLine()
		if err != nil {
			err := fmt.Errorf("line %d: %w", lineNumber, err)
			return err
		}
		line.line = ""
	}
	slog.Debug(strings.Join(ModelList, ", "))
	return nil
}

// Parse one line from file.
func (line *optionLine) parseLine() error {
	model := line.parseModel()
	if model == nil {
		return nil
	}
	switch getModel(model.model) {
	case TypeModel, TypeDash, TypeSlash:
		// Get device number
		first := line.parseFirst()
		if first == nil || !first.isAddr {
			return fmt.Errorf("device %s: requires device address, line: %d", model.model, lineNumber)
		}

		// Get any remaining options.
		options, err := line.parseOptions()
		if err != nil {
			return err
		}

		// Try and create the device.
		err = createModel(model.model, first, options)
		if err != nil {
			return fmt.Errorf("device %s: %w, line: %d %s", model.model, err, lineNumber, line.line)
		}

	case TypeOption:
		first := line.parseFirst()
		line.skipSpace()
		if !line.isEOL() || first == nil {
			return fmt.Errorf("option %s: not followed by value. line: %d", model.model, lineNumber)
		}
		err := createOption(model.model, first)
		if err != nil {
			return fmt.Errorf("option %s: %w, line: %d %s", model.model, err, lineNumber, line.line)
		}

	case TypeOptions:
		first := line.parseFirst()
		if first == nil {
			return fmt.Errorf("option %s: not followed by value, line: %d", model.model, lineNumber)
		}
		options, err := line.parseOptions()
		if err != nil {
			return err
		}
		err = createOptions(model.model, first, options)
		if err != nil {
			return fmt.Errorf("option %s: %w, line: %d %s", model.model, err, lineNumber, line.line)
		}

	case TypeSwitch:
		line.skipSpace()
		if !line.isEOL() {
			return fmt.Errorf("switch %s: followed by options, line: %d", model.model, lineNumber)
		}
		err := createSwitch(model.model)
		if err != nil {
			return fmt.Errorf("switch %s: %w, line: %d %s", model.model, err, lineNumber, line.line)
		}

	case TypeFile:
		line.skipSpace()
		line.pos-- // Back up one position.
		if line.isEOL() {
			return fmt.Errorf("file %s: requires a file name, line %d", model.model, lineNumber)
		}
		v, ok := line.parseQuoteString()
		if !ok {
			return fmt.Errorf("invalid quoted string line: %d [%d]", lineNumber, line.pos)
		}
		err := createFile(model.model, v)
		if err != nil {
			return fmt.Errorf("file %s: %w, line: %d %s", model.model, err, lineNumber, line.line)
		}

	case 0:
		return fmt.Errorf("no type: %s registered, line: %d", model.model, lineNumber)
	}
	return nil
}

// Skip forward over line until none whitespace character found.
func (line *optionLine) skipSpace() {
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
func (line *optionLine) isEOL() bool {
	if line.pos >= len(line.line) {
		return true
	}

	if line.line[line.pos] == '#' {
		return true
	}
	return false
}

// Return next letter or digit in line. 0 if EOL or space.
func (line *optionLine) getNext() byte {
	line.pos++
	if line.isEOL() {
		return 0
	}
	return line.line[line.pos]
}

// Peek at next character.
func (line *optionLine) getPeek() byte {
	if (line.pos + 1) >= len(line.line) {
		return 0
	}
	return line.line[line.pos+1]
}

// Parse model option.
func (line *optionLine) parseModel() *modelName {
	// Skip leading space
	line.skipSpace()
	// Check if end of line.
	if line.isEOL() {
		return nil
	}

	model := modelName{}

	// Get model name
	for {
		if line.isEOL() {
			break
		}
		by := line.line[line.pos]
		if unicode.IsLetter(rune(by)) || unicode.IsNumber(rune(by)) {
			model.model += string([]byte{by})
			line.pos++
			continue
		}
		break
	}

	model.model = strings.ToUpper(model.model)
	return &model
}

// Parse first option parameter.
func (line *optionLine) parseFirst() *FirstOption {
	// Skip leading space
	line.skipSpace()
	// Check if end of line.
	if line.isEOL() {
		return nil
	}

	value := ""
	for {
		if line.isEOL() {
			break
		}
		by := line.line[line.pos]
		if unicode.IsLetter(rune(by)) || unicode.IsNumber(rune(by)) {
			value += string([]byte{by})
			line.pos++
			continue
		}
		break
	}

	option := FirstOption{devNum: D.NoDev, value: value}

	devNum, ok := strconv.ParseUint(value, 16, 12)

	if ok == nil {
		option.devNum = uint16(devNum)
		option.isAddr = true
	}
	return &option
}

// Parse string that is "string" or just string.
func (line *optionLine) parseQuoteString() (string, bool) {
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
		// Space or comma terminates a no quoted string.
		if !inQuote && (space || by == 0 || by == ',') {
			return value, true
		}

		value += string(by)
		// If we hit end of line, stop processing.
		if line.isEOL() {
			return value, !inQuote
		}
	}
}

// Parse option name.
func (line *optionLine) getName() (string, error) {
	// Check if end of line.
	if line.isEOL() {
		return "", nil
	}

	// First character must be alphanumeric.
	by := line.line[line.pos]
	if !unicode.IsLetter(rune(by)) && !unicode.IsNumber(rune(by)) {
		if !line.isEOL() {
			return "", fmt.Errorf("invalid option encountered line: %d [%d]", lineNumber, line.pos)
		}
		return "", nil
	}

	value := ""
	// Already verified that first character is letter,
	// so grab until not letter or number.
	for {
		value += string([]byte{by})
		by = line.getNext()
		if line.isEOL() || unicode.IsSpace(rune(by)) || by == '=' || by == ',' {
			break
		}
	}

	return value, nil
}

// Parse options for a line.
func (line *optionLine) parseOption() (*Option, error) {
	// Skip leading space
	line.skipSpace()

	// Grab option name
	value, err := line.getName()
	if value == "" {
		return nil, err
	}

	// Empty option.
	option := Option{Name: value}

	// If at end of line done.
	if line.isEOL() {
		return &option, nil
	}

	// Check if equals option.
	if line.line[line.pos] == '=' {
		v, ok := line.parseQuoteString()
		if ok {
			option.EqualOpt = v
		} else {
			return nil, fmt.Errorf("invalid quoted string line: %d [%d]", lineNumber, line.pos)
		}
	}

	// Skip any spaces.
	line.skipSpace()

	// Grab all , options
	for !line.isEOL() && line.line[line.pos] == ',' {
		line.pos++ // Skip comma
		// Skip space between , and next option
		line.skipSpace()
		v, err := line.getName()
		if err != nil {
			return nil, err
		}
		if v != "" {
			option.Value = append(option.Value, &v)
		}
		// Skip any trailing spaces.
		line.skipSpace()
	}

	return &option, nil
}

// Collect all options for line.
func (line *optionLine) parseOptions() ([]Option, error) {
	options := []Option{}
	for {
		option, err := line.parseOption()
		if err != nil {
			return nil, err
		}
		if option == nil {
			break
		}
		options = append(options, *option)
	}
	return options, nil
}
