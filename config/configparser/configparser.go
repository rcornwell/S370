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
	"io"
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
)

// Model creation list.
type modelDef struct {
	// create func(uint16, byte, byte, string, []*Option) bool
	create func(uint16, string, []Option) error
	ty     int
}

var models = map[string]modelDef{}

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
// func RegisterModel(mod string, ty int, fn func(uint16, byte, byte, string, []*Option) bool) {.
func RegisterModel(mod string, ty int, fn func(uint16, string, []Option) error) {
	mod = strings.ToUpper(mod)
	fmt.Println("Registering device: ", mod)
	model := modelDef{create: fn, ty: ty}
	models[mod] = model
}

// Register should be called from init functions.
func RegisterSwitch(mod string, fn func(uint16, string, []Option) error) {
	mod = strings.ToUpper(mod)
	fmt.Println("Registering switch: ", mod)
	model := modelDef{create: fn, ty: TypeSwitch}
	models[mod] = model
}

// Register should be called from init functions.
func RegisterOption(mod string, fn func(uint16, string, []Option) error) {
	mod = strings.ToUpper(mod)
	fmt.Println("Registering simple option: ", mod)
	model := modelDef{create: fn, ty: TypeOption}
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
	return model.create(first.devNum, "", options)
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

// Load in a configuration file.
func LoadConfigFile(name string) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()

	lineNumber = 0
	reader := bufio.NewReader(file)
	for {
		var err error

		line := optionLine{}
		line.line, err = reader.ReadString('\n')
		lineNumber++
		if len(line.line) == 0 && err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		err = line.parseLine()
		if err != nil {
			return err
		}
	}
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
			err := fmt.Sprintf("Device %s requires device address, line: %d\n", model.model, lineNumber)
			return errors.New(err)
		}
		// Get any remaining options.
		options, err := line.parseOptions()
		if err != nil {
			return err
		}

		// Try and create the device.
		return createModel(model.model, first, options)

	case TypeOption:
		first := line.parseFirst()
		line.skipSpace()
		if !line.isEOL() || first == nil {
			err := fmt.Sprintf("Option: %s not followed by value. line: %d\n", model.model, lineNumber)
			return errors.New(err)
		}
		return createOption(model.model, first)

	case TypeOptions:
		first := line.parseFirst()
		if first == nil {
			err := fmt.Sprintf("Option: %s not followed by value, line: %d\n", model.model, lineNumber)
			return errors.New(err)
		}
		options, err := line.parseOptions()
		if err != nil {
			return err
		}
		return createOptions(model.model, first, options)

	case TypeSwitch:
		line.skipSpace()
		if !line.isEOL() {
			err := fmt.Sprintf("Switch Option: %s followed by options, line: %d\n", model.model, lineNumber)
			return errors.New(err)
		}
		return createSwitch(model.model)
	case 0:
		err := fmt.Sprintf("No type: %s registered, line: %d\n", model.model, lineNumber)
		return errors.New(err)
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
func (line *optionLine) getNext(inQuote bool) byte {
	line.pos++
	if line.isEOL() {
		return 0
	}
	by := line.line[line.pos]
	if unicode.IsLetter(rune(by)) || unicode.IsNumber(rune(by)) || inQuote {
		return by
	}
	return 0
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
	// // Check if either - or / option following
	// for {
	// 	by := line.line[line.pos]
	// 	// Check for dash option
	// 	if by == '-' {
	// 		if model.dash != 0 {
	// 			fmt.Printf("Model contains more then one - option: line %s\n", line.line)
	// 			return nil
	// 		}
	// 		model.dash = line.getNext()
	// 		continue
	// 	}

	// 	// Check for minue option
	// 	if by == '/' {
	// 		if model.slash != 0 {
	// 			fmt.Printf("Model contains more then one / option: line %s\n", line.line)
	// 			return nil
	// 		}
	// 		model.slash = line.getNext()
	// 		continue
	// 	}
	// 	break
	// }
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
		_ = line.getNext(true)
	}

	for {
		by := line.getNext(inQuote)
		// If processing a quoted string "" gets replaced by signal quote
		if by == '"' && inQuote {
			by = line.getNext(inQuote)
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

	// First character must be alphabetic.
	by := line.line[line.pos]
	if !unicode.IsLetter(rune(by)) {
		if !line.isEOL() {
			err := fmt.Sprintf("Invalid option encountered line: %d [%d]\n", lineNumber, line.pos)
			return "", errors.New(err)
		}
		return "", nil
	}
	value := ""

	// Already verified that first character is letter,
	// so grab until not letter or number.
	for {
		value += string([]byte{by})
		by = line.getNext(false)
		if by == 0 {
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
			err := fmt.Sprintf("Invalid quoted string line: %d [%d]\n", lineNumber, line.pos)
			return nil, errors.New(err)
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
