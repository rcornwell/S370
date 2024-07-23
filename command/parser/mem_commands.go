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
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	assembler "github.com/rcornwell/S370/emu/assemble"
	core "github.com/rcornwell/S370/emu/core"
	"github.com/rcornwell/S370/emu/cpu"
	Dv "github.com/rcornwell/S370/emu/device"
	disassembler "github.com/rcornwell/S370/emu/disassemble"
	"github.com/rcornwell/S370/emu/memory"
	"github.com/rcornwell/S370/util/xlat"
)

type memoryOpts struct {
	file      *os.File // File to output too.
	long      bool     // Long floating point values.
	wordSize  int      // Size to print values in.
	char      bool     // Characters.
	virtual   bool     // Virtual address.
	set       bool     // Flag set.
	decimal   bool     // Dump in decimal.
	regType   int      // Type of register to display.
	prefix    string   // Prefix for register display.
	high      bool     // High value defined.
	lowRange  uint32   // Lower start to display
	highRange uint32   // Highest value to display.
}

type regdef struct {
	regType int    // Type of register.
	next    byte   // Next value required.
	name    string // Name if in error.
	prefix  string // Prefix to print.
	size    int    // Size of register.
}

var regType = map[string]regdef{
	"reg":   {Dv.Register, '[', "", "R", 15},
	"r":     {Dv.Register, '[', "", "R", 15},
	"fpreg": {Dv.FPRegister, '[', "floating point", "F", 6},
	"fp":    {Dv.FPRegister, '[', "floating point", "F", 6},
	"ctl":   {Dv.CtlRegister, '[', "control", "Ctl", 15},
	"psw":   {Dv.PSWRegister, 0, "PSW", "", 1},
	"pc":    {Dv.PCRegister, 0, "PC", "PC=", 1},
}

// Parse hex number.
func parseHexValue(text string) (uint32, bool) {
	var value uint32
	// Characters must be alphabetic
	for _, by := range text {
		digit := strings.Index(hex, string(by))
		if digit == -1 {
			return 0, false
		}
		value = (value << 4) + uint32(digit)
	}

	return value, true
}

// Parse decimal number.
func parseDecimalValue(text string) (uint32, bool) {
	var value uint32
	// Characters must be alphabetic
	for _, by := range text {
		digit := strings.Index(hex, string(by))
		if digit == -1 || digit >= 10 {
			return 0, false
		}
		value = (value * 10) + uint32(digit)
	}

	return value, true
}

// Parse memory option word.
func (line *cmdLine) parseMemoryValue(by byte) (string, byte) {
	if by == ' ' {
		line.skipSpace()
		by = line.getCurrent()
	}

	// Accept any letter or digit up to [ or : or -
	value := ""
	for by != 0 {
		if by == '-' || by == ':' || by == '[' || unicode.IsSpace(rune(by)) {
			break
		}
		if !unicode.IsLetter(rune(by)) && !unicode.IsDigit(rune(by)) {
			break
		}
		value += string([]byte{by})
		by = line.getCurrent()
	}

	return strings.ToLower(value), by
}

// Get options for memory reference command.
func (line *cmdLine) parseMemoryOptions(options *memoryOpts, file bool) (byte, error) {
	var char byte

	for {
		char = line.getCurrent()
		if char == 0 {
			if line.isEOL() {
				return 0, nil
			}
			continue
		}
		if unicode.IsSpace(rune(char)) {
			continue
		}

		// If file name given.
		if file && char == '@' {
			if options.file != nil {
				return 0, errors.New("can't specify one then on file to use")
			}
			fileName, ok := line.parseQuoteString()
			if ok {
				file, err := os.OpenFile(fileName, os.O_CREATE, 0o660)
				if err != nil {
					return 0, err
				}
				options.file = file
			}
		}

		// If not an option, return what we got.
		if char != '-' {
			return char, nil
		}

		// Collect options given.
		char = line.getCurrent()
		for !unicode.IsSpace(rune(char)) {
			char = strings.ToLower(string(char))[0]
			switch char {
			case 'h': // Half word.
				if options.wordSize != 0 {
					return 0, errors.New("wordsize already defined")
				}
				options.wordSize = 2
				options.set = true

			case 'b': // bytes.
				if options.wordSize != 0 {
					return 0, errors.New("wordsize already defined")
				}
				options.wordSize = 1
				options.set = true

			case 's': // symbolic.
				options.regType = Dv.Symbolic
				//	options.halfWord = true
				options.set = true

			case 'l': // Long floating point.
				options.long = true

			case 'c': // characters
				options.char = true

			case 'f': // full words, default
				if options.wordSize != 0 {
					return 0, errors.New("wordsize already defined")
				}
				options.wordSize = 4
				options.set = true

			case 'v': // virtual address.
				options.virtual = true

			case 'd': // Dump in decimal.
				options.decimal = true

			default:
				return 0, errors.New("option invalid: -" + string(char))
			}
			char = line.getCurrent()
		}
	}
}

// Parse a deposit item.
func (line *cmdLine) parseDepositReg(num int) ([]uint32, error) {
	memData := []uint32{}
	line.skipSpace()

	// If we shift non-zero out of this digit we have overflow.
	maxDigit := uint32(0xf0000000)

	haveNum := false
	inSpace := true
	word := uint32(0)
	for {
		by := line.getCurrent()
		if !haveNum && by == 0 {
			break
		}

		if by == ',' || by == ';' || by == 0 || (!inSpace && unicode.IsSpace(rune(by))) {
			memData = append(memData, word)

			if len(memData) > (num + 1) {
				return []uint32{}, fmt.Errorf("too many register values: %d", len(memData))
			}
			inSpace = true
			haveNum = false
			word = 0
			continue
		}

		if inSpace && unicode.IsSpace(rune(by)) {
			continue
		}
		digit := strings.Index(hex, string(by))
		if digit == -1 {
			return []uint32{}, fmt.Errorf("non hex digit encountered: '%c'", by)
		}
		haveNum = true
		inSpace = false
		if (word & maxDigit) != 0 {
			return []uint32{}, fmt.Errorf("value out of range: %x", word)
		}
		word = (word << 4) + uint32(digit)
	}
	return memData, nil
}

// Parse a deposit item.
func (line *cmdLine) parseDepositHex(wordSize int) ([]byte, error) {
	memData := []byte{}
	line.skipSpace()

	// If we shift non-zero out of this digit we have overflow.
	maxDigit := uint32(0xf0000000)
	if wordSize == 2 {
		maxDigit >>= 16
	}
	if wordSize == 1 {
		maxDigit >>= 24
	}

	inSpace := true
	haveNum := false
	word := uint32(0)
	for {
		by := line.getCurrent()
		if !haveNum && by == 0 {
			break
		}

		if by == ',' || by == ';' || by == 0 || (!inSpace && unicode.IsSpace(rune(by))) {
			switch wordSize {
			case 4:
				memData = append(memData, byte((word>>24)&0xff))
				memData = append(memData, byte((word>>16)&0xff))
				fallthrough
			case 2:
				memData = append(memData, byte((word>>8)&0xff))
				fallthrough
			case 1:
				memData = append(memData, byte((word>>0)&0xff))
			}
			word = 0
			haveNum = false
			inSpace = true
			if line.isEOL() {
				break
			}
			continue
		}

		if inSpace && unicode.IsSpace(rune(by)) {
			continue
		}

		digit := strings.Index(hex, string(by))
		if digit == -1 {
			return []byte{}, fmt.Errorf("non digit encountered: '%c'", by)
		}
		haveNum = true
		inSpace = false
		if (word & maxDigit) != 0 {
			return []byte{}, fmt.Errorf("value out of range: %x", word)
		}
		word = (word << 4) + uint32(digit)
	}
	return memData, nil
}

// Parse a deposit item.
func (line *cmdLine) parseDepositDecimal(wordSize int) ([]byte, error) {
	memData := []byte{}
	line.skipSpace()

	// If we shift non-zero out of this digit we have overflow.
	maxDigit := uint32(0x80000000)
	if wordSize == 2 {
		maxDigit >>= 16
	}
	if wordSize == 1 {
		maxDigit >>= 24
	}

	inSpace := true
	haveNum := false
	word := uint32(0)
	for {
		by := line.getCurrent()
		if !haveNum && by == 0 {
			break
		}

		if by == ',' || by == ';' || by == 0 || (!inSpace && unicode.IsSpace(rune(by))) {
			switch wordSize {
			case 4:
				memData = append(memData, byte((word>>24)&0xff))
				memData = append(memData, byte((word>>16)&0xff))
				fallthrough
			case 2:
				memData = append(memData, byte((word>>8)&0xff))
				fallthrough
			case 1:
				memData = append(memData, byte((word>>0)&0xff))
			}
			word = 0
			haveNum = false
			inSpace = true
			if line.isEOL() {
				break
			}
			continue
		}

		if inSpace && unicode.IsSpace(rune(by)) {
			continue
		}

		digit := strings.Index(hex, string(by))
		if digit == -1 || digit >= 10 {
			return []byte{}, fmt.Errorf("non digit encountered: '%c'", by)
		}
		inSpace = false
		if (word & maxDigit) != 0 {
			return []byte{}, fmt.Errorf("value out of range: %d", word)
		}
		word = (word * 10) + uint32(digit)
		haveNum = true
	}
	return memData, nil
}

// Parse a deposit item.
func (line *cmdLine) parseChar() []byte {
	memData := []byte{}
	line.skipSpace()

	var inQuote byte

	by := line.getCurrent()
	if by == '\'' || by == '"' {
		inQuote = by
		by = line.getCurrent()
	}

	for by != 0 {
		if by == inQuote {
			by = line.getCurrent()
			if by != inQuote {
				break
			}
		}
		memData = append(memData, xlat.ASCIIToEBCDIC[by])
		by = line.getCurrent()
	}
	return memData
}

// Parse a symbolic item.
func (line *cmdLine) parseSymbolic() ([]byte, error) {
	memData := []byte{}
	line.skipSpace()

	// Accept characters up to ; : or EOL
	value := ""
	for {
		by := line.getCurrent()
		if value == "" && by == 0 {
			break
		}

		if by == ';' || by == ':' || by == 0 {
			inst, err := assembler.Assemble(value)
			if err != nil {
				return []byte{}, err
			}
			memData = append(memData, inst...)
			value = ""
			if line.isEOL() {
				break
			}
		} else {
			value += string([]byte{by})
		}
	}
	return memData, nil
}

// Get range and type of access.
func (line *cmdLine) parseMemoryRange(options *memoryOpts, char byte) error {
	// Collect type or memory range.
	var value string

	hexOption := true
	final := ' '
	value, char = line.parseMemoryValue(char)
	// // If character is blank skip forward to non blank
	// if unicode.IsSpace(rune(char)) {
	// 	line.skipSpace()
	// 	char = line.getCurrent()
	// }

	// See if register identifier.
	reg, ok := regType[value]
	if ok {
		options.regType = reg.regType
		options.prefix = reg.prefix
		hexOption = false
		options.set = true

		// If end of line, return all registers.
		if char == 0 {
			options.lowRange = 0
			options.highRange = uint32(reg.size)
			return nil
		}
		// Get next none blank char
		if reg.next != 0 && char != reg.next {
			return errors.New(reg.name + "register requires " + string(reg.next))
		}
		if reg.next == 0 {
			// if !line.isEOL() {
			// 	return errors.New(reg.name + "can't have index pr range")
			// }
			return nil
		}
		final = ']'
		char = line.getCurrent()
		value, char = line.parseMemoryValue(char)
	}

	// Grab start and possible end address.
	for {
		var numeric uint32

		// Parse first word gotten as number
		ok := false
		if hexOption {
			numeric, ok = parseHexValue(value)
			if !ok {
				return errors.New("not valid hex number")
			}
		} else {
			numeric, ok = parseDecimalValue(value)
			if !ok {
				return errors.New("not valid number")
			}
		}

		// If already seen separator, set high range.
		if options.high {
			options.highRange = numeric
		} else {
			options.lowRange = numeric
			options.highRange = numeric
		}

		// If final character matches what we are looking for.
		if final != ' ' && rune(char) == final {
			break
		}

		if line.isEOL() || unicode.IsSpace(rune(char)) {
			break
		}

		if char == ':' || char == '-' {
			if options.high {
				return errors.New("can't have second high value")
			}
			options.high = true
			char = line.getCurrent()
		}

		value, char = line.parseMemoryValue(char)
		if value == "" {
			break
		}
	}

	// Check if register terminated correctly.
	if final != ' ' && rune(char) != final {
		return errors.New("register must terminate in " + string(final))
	}
	return nil
}

// Dump range of memory to file.
func dumpMemory(options *memoryOpts) {
	// Set up word size based on what options were called for.
	if !options.set {
		if options.char {
			options.wordSize = 16
		} else {
			options.wordSize = 4
		}
	}

	for {
		var str string
		mem := memory.GetBytes(options.lowRange, options.wordSize)
		str = fmt.Sprintf("%06X: ", options.lowRange)

		switch options.wordSize {
		case 4:
			word := uint32(mem[0]) << 24
			word |= uint32(mem[1]) << 16
			word |= uint32(mem[2]) << 8
			word |= uint32(mem[3])
			if options.decimal {
				str += strconv.FormatUint(uint64(word), 10)
			} else {
				str += fmt.Sprintf("%08X ", word)
			}

		case 2:
			word := uint32(mem[0]) << 8
			word |= uint32(mem[1])

			if options.decimal {
				str += strconv.FormatUint(uint64(word), 10)
			} else {
				str += fmt.Sprintf("%04X ", word)
			}

		case 1:
			if options.decimal {
				str += strconv.FormatUint(uint64(mem[0]), 10)
			} else {
				str += fmt.Sprintf("%02X ", mem[0])
			}
		}
		if options.char {
			str += "'"
			for j := range options.wordSize {
				by := xlat.EBCDICToASCII[mem[j]]
				if !unicode.IsPrint(rune(by)) {
					by = '.'
				}
				str += string(by)
			}
			str += "' "
		}

		options.lowRange += uint32(options.wordSize)
		fmt.Fprintln(options.file, str)
		if options.lowRange > options.highRange {
			break
		}
	}
}

// Dump symbolic instructions to file.
func dumpSymbolic(options *memoryOpts) {
	for {
		var str string
		var inst string
		mem := memory.GetBytes(options.lowRange, 6)
		str = fmt.Sprintf("%06X: ", options.lowRange)
		length := 0
		if options.wordSize != 2 {
			inst, length = disassembler.PrintInst(mem)
		} else {
			inst, length = disassembler.Disassemble(mem)
			for i := 0; i < 6; i += 2 {
				if i >= length {
					str += "     "
				} else {
					str += fmt.Sprintf("%02X%02X ", mem[i], mem[i+1])
				}
			}
			str += "  "
		}
		str += inst
		options.lowRange += uint32(length)
		fmt.Fprintln(options.file, str)
		if options.lowRange > options.highRange {
			return
		}
	}
}

// Dump register values.
func dumpRegister(options *memoryOpts) error {
	// Check if registers in range.
	if options.lowRange > 15 || options.highRange > 15 {
		return errors.New("register number too high")
	}

	for {
		var str string
		switch options.regType {
		case Dv.FPRegister:
			value, ok := cpu.GetFPReg(int(options.lowRange), options.long)
			if !ok {
				return fmt.Errorf("invalid register number: %d", options.lowRange)
			}
			if options.long {
				str += fmt.Sprintf("%s[%d] = %016x ", options.prefix, options.lowRange, value)
			} else {
				str += fmt.Sprintf("%s[%d] = %08x ", options.prefix, options.lowRange, (value >> 32))
			}

			e := float64((value>>56)&0x7f) - 64.0
			d := float64(cpu.MMASKL & value)
			d *= math.Exp2(-56.0 + 4.0*e)
			if (cpu.MSIGNL & value) != 0 {
				d *= -1.0
			}
			str += fmt.Sprintf("%f", d)
			options.lowRange += 2

		case Dv.Register, Dv.CtlRegister:
			value, ok := cpu.GetReg(options.regType, uint8(options.lowRange))
			if !ok {
				return errors.New("invalid register number")
			}
			str = fmt.Sprintf("%s[%d] = %08X", options.prefix, options.lowRange, value)
			options.lowRange++
		}
		fmt.Fprintln(options.file, str)
		if options.lowRange > options.highRange {
			return nil
		}
	}
}

// Examine memory/CPU command.
func examine(line *cmdLine, _ *core.Core) (bool, error) {
	var options memoryOpts

	// Get options settings.
	char, err := line.parseMemoryOptions(&options, true)
	if err != nil {
		return false, err
	}

	if options.file == nil {
		options.file = os.Stdout
	} else {
		defer options.file.Close()
	}

	// Get type and range.
	err = line.parseMemoryRange(&options, char)
	if err != nil {
		return false, err
	}

	// Check if we got all of command.
	if !line.isEOL() {
		return false, errors.New("extra arguments to command ")
	}

	// Make sure range is enough.
	if options.high && options.highRange < options.lowRange {
		return false, errors.New("high address below low address")
	}

	// Dump requested values.
	switch options.regType {
	default:
		fallthrough
	case 0: // Access memory.
		dumpMemory(&options)

	case Dv.Symbolic:
		options.lowRange &= ^uint32(1)
		dumpSymbolic(&options)

	case Dv.FPRegister, Dv.Register, Dv.CtlRegister:
		err = dumpRegister(&options)

	case Dv.PSWRegister:
		fmt.Fprintln(options.file, cpu.GetPSW())

	case Dv.PCRegister:
		fmt.Fprintf(options.file, "PC=%06x\n", cpu.GetPC())
	}

	return false, err
}

// Deposit memory/CPU command.
func deposit(line *cmdLine, core *core.Core) (bool, error) {
	if core.IsRunning() {
		return false, errors.New("can't deposit when CPU is running")
	}

	// Collect next characters an process arguments.
	var options memoryOpts

	// Get options settings.
	char, err := line.parseMemoryOptions(&options, false)
	if err != nil {
		return false, err
	}

	if options.file == nil {
		options.file = os.Stdout
	} else {
		defer options.file.Close()
	}

	// Get type and range.
	err = line.parseMemoryRange(&options, char)
	if err != nil {
		return false, err
	}

	// Make sure range is enough.
	if options.high && options.highRange < options.lowRange {
		return false, errors.New("high address below low address")
	}

	// Set up word size based on what options were called for.
	if !options.set {
		options.wordSize = 1
	}
	if options.regType != 0 {
		options.wordSize = 4
	}

	memData := []byte{}
	// Dump requested values.
	switch options.regType {
	default:
		fallthrough
	case 0: // Access memory.
		if options.char {
			memData = line.parseChar()
			break
		}
		if options.decimal {
			memData, err = line.parseDepositDecimal(options.wordSize)
			break
		}
		memData, err = line.parseDepositHex(options.wordSize)

	case Dv.Symbolic:
		memData, err = line.parseSymbolic()

	case Dv.Register, Dv.CtlRegister:
		num := int(options.highRange - options.lowRange)
		regData, regerr := line.parseDepositReg(num)
		if regerr != nil {
			return false, regerr
		}
		for i, value := range regData {
			cpu.SetReg(options.regType, uint8(i+int(options.lowRange)), value)
		}
		return false, nil

	case Dv.FPRegister:
		return false, errors.New("can't deposit to floating point registers yet")

	case Dv.PSWRegister:
		err = errors.New("can't deposit into PSW")

	case Dv.PCRegister:
		regData, regerr := line.parseDepositReg(1)
		if regerr == nil {
			cpu.SetPC(regData[0])
		}
		return false, regerr
	}

	if err != nil {
		return false, err
	}

	if !options.high {
		options.highRange = options.lowRange + uint32(len(memData))
	}

	for options.lowRange < options.highRange {
		if (options.lowRange + uint32(len(memData))) > options.highRange {
			memData = memData[0:int(options.highRange-options.lowRange)]
		}
		memory.SetBytes(options.lowRange, memData)
		options.lowRange += uint32(len(memData))
	}

	return false, err
}
