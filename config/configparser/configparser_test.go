/*
 * S370 - Configuration file parser test set.
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
	"fmt"
	"testing"

	D "github.com/rcornwell/S370/emu/device"
)

var testOptions []Option
var testDevNum uint16
var testValue string
var testType string

func resetTest() {
	testOptions = []Option{}
	testDevNum = 0xffff
	testValue = "error"
	testType = ""
}

func cleanUpConfig() {
	models = map[string]modelDef{}
	resetTest()
	fmt.Println("Cleanup")
}

// Create a device.
func modDevice(devNum uint16, value string, options []Option) error {
	testDevNum = devNum
	testValue = value
	testType = "model"
	testOptions = options
	return nil
}

// Create a switch.
func modSwitch(devNum uint16, value string, options []Option) error {
	testDevNum = devNum
	testValue = value
	testType = "switch"
	testOptions = options
	return nil
}

// Create a Option type.
func modOption(devNum uint16, value string, options []Option) error {
	testDevNum = devNum
	testValue = value
	testType = "option"
	testOptions = options
	return nil
}

// Test regisering a model.
func TestRegisterModel(t *testing.T) {
	cleanUpConfig()

	RegisterModel("testdev", TypeModel, modDevice)
	fTest := FirstOption{devNum: 0x100, isAddr: true, value: "test"}
	err := createModel("test", &fTest, nil)
	if err == nil {
		t.Errorf("Create non existent model succeeded")
	}
	err = createModel("testdev", &fTest, nil)
	if err != nil {
		t.Errorf("Unable to create model")
	}
	if testDevNum != 0x100 {
		t.Errorf("Device number not valid: %d", testDevNum)
	}
	if testValue != "" {
		t.Errorf("Device number not valid: %s", testValue)
	}
	err = createSwitch("testdev")
	if err == nil {
		t.Errorf("Create device as switch succeeded")
	}
}

// Test register a switch
func TestRegisterSwitch(t *testing.T) {
	cleanUpConfig()

	RegisterSwitch("testswitch", modSwitch)
	err := createSwitch("test")
	if err == nil {
		t.Errorf("Create non existent switch succeeded")
	}
	err = createSwitch("testswitch")
	if err != nil {
		t.Errorf("Unable to create switch")
	}
	if testDevNum != 0 {
		t.Errorf("Switch number not valid: %d", testDevNum)
	}
	if testValue != "" {
		t.Errorf("Switch value not valid: %s", testValue)
	}
	fTest := FirstOption{devNum: 0x100, isAddr: true, value: "test"}
	err = createModel("testdev", &fTest, nil)
	if err == nil {
		t.Errorf("Create switch as model succeeded")
	}
}

// Test register an option.
func TestRegisterOption(t *testing.T) {
	cleanUpConfig()

	fTest := FirstOption{devNum: 0x100, isAddr: false, value: "test"}
	RegisterOption("testoption", modOption)
	err := createOption("test", &fTest)
	if err == nil {
		t.Errorf("Create non existent option succeeded")
	}
	err = createOption("testoption", &fTest)
	if err != nil {
		t.Errorf("Unable to create option")
	}
	if testDevNum != D.NoDev {
		t.Errorf("Option number not valid: %d", testDevNum)
	}
	if testValue != "test" {
		t.Errorf("Option value not valid: %s", testValue)
	}
	err = createModel("testoption", &fTest, nil)
	if err == nil {
		t.Errorf("Create option as model succeeded")
	}
}

// Test register multiple options.
func TestRegisterMultiple(t *testing.T) {
	cleanUpConfig()

	fTest := FirstOption{devNum: 0x100, isAddr: false, value: "test"}
	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)
	err := createOption("test", &fTest)
	if err == nil {
		t.Errorf("Create non existent option succeeded")
	}
	err = createOption("testoption", &fTest)
	if err != nil {
		t.Errorf("Unable to create option")
	}
	err = createSwitch("testSwitch")
	if err != nil {
		t.Errorf("Unable to create switch")
	}
	err = createModel("testdevice", &fTest, nil)
	if err != nil {
		t.Errorf("Unable to create device")
	}
}

// Test parsing of switch types.
func TestParseLineSwitch(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "testSwitch", pos: 0}
	err := line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse switch")
	}
	if testType != "switch" {
		t.Errorf("ParseLine did not create a switch")
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave switch some options")
	}

	resetTest()
	line = optionLine{line: "testSwitch  # Comment", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse switch and coment")
	}
	if testType != "switch" {
		t.Errorf("ParseLine did not create a switch")
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave switch some options")
	}

	resetTest()
	line = optionLine{line: "testSwitch 0", pos: 0}
	err = line.parseLine()
	if err == nil {
		t.Errorf("ParseLine succeeded in parseing switch with address")
	}
	if testType == "switch" {
		t.Errorf("ParseLine created a switch with argument")
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave switch some options")
	}

	resetTest()
	line = optionLine{line: "testSwitch 0 name", pos: 0}
	err = line.parseLine()
	if err == nil {
		t.Errorf("ParseLine created a switch with argument and options")
	}
	if testType == "switch" {
		t.Errorf("ParseLine created a switch with argument and options")
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave switch some options")
	}
}

// Test parsing of optonal parameter types.
func TestParseLineOption(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "TESTOPTION", pos: 0}
	err := line.parseLine()
	if err == nil {
		t.Errorf("ParseLine created an option with no argument")
	}

	resetTest()
	line = optionLine{line: "testOption enable  # Comment", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse option and coment")
	}
	if testType != "option" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != D.NoDev {
		t.Errorf("Option set address to %04x\n", testDevNum)
	}
	if testValue != "enable" {
		t.Errorf("Option did not set value")
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave option some extra options")
	}

	resetTest()
	line = optionLine{line: "testOption 0100    ", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "option" {
		t.Errorf("ParseLine did not create a option")
	}

	if testDevNum != 0x100 {
		t.Errorf("Option set address to %04x\n", testDevNum)
	}
	if testValue != "0100" {
		t.Errorf("Option did not set value")
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave option some extra options")
	}
}

// Test parsing of model parameter types.
func TestParseLineModel(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "TESTdevice", pos: 0}
	err := line.parseLine()
	if err == nil {
		t.Errorf("ParseLine created model without argument")
	}

	resetTest()
	line = optionLine{line: "testDevice enable  # Comment", pos: 0}
	err = line.parseLine()
	if err == nil {
		t.Errorf("ParseLine created device with invalid address")
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave device some extra options")
	}

	resetTest()
	line = optionLine{line: "testDevice 0100    ", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave device some extra options")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
}

// Test parsing of model with optional flags.
func TestParseLineModelOptions(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "testDevice 0100    ", pos: 0}
	err := line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	if len(testOptions) != 0 {
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}

	resetTest()
	line = optionLine{line: "testDevice 0100   single ", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 1:
		if testOptions[0].Name != "single" {
			t.Errorf("ParseLine did not give correct option")
		}
		if testOptions[0].EqualOpt != "" {
			t.Errorf("ParseLine gave equal value")
		}
		if len(testOptions[0].Value) != 0 {
			t.Errorf("ParseLine comma parameters")
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}

	resetTest()
	line = optionLine{line: "testDevice 0100   single second  ", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 2:
		if testOptions[0].Name != "single" {
			t.Errorf("ParseLine did not give correct option")
		}
		if testOptions[0].EqualOpt != "" {
			t.Errorf("ParseLine gave equal value")
		}
		if len(testOptions[0].Value) != 0 {
			t.Errorf("ParseLine comma parameters")
		}
		if testOptions[1].Name != "second" {
			t.Errorf("ParseLine did not give correct second option")
		}
		if testOptions[1].EqualOpt != "" {
			t.Errorf("ParseLine gave equal value for second")
		}
		if len(testOptions[1].Value) != 0 {
			t.Errorf("ParseLine comma parameters")
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}
}

// Test comma options.
func TestParseLineModelOptionsComma(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "testDevice 0100   single, second", pos: 0}
	err := line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 1:
		if testOptions[0].Name != "single" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "" {
			t.Errorf("ParseLine gave equal value")
		}
		if len(testOptions[0].Value) == 1 {
			if *testOptions[0].Value[0] != "second" {
				t.Errorf("First comma value not correct: %s", *testOptions[0].Value[0])
			}
		} else {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}

	resetTest()
	line = optionLine{line: "testDevice 0101   test, second, third # comment", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x101 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 1:
		if testOptions[0].Name != "test" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "" {
			t.Errorf("ParseLine gave equal value")
		}
		if len(testOptions[0].Value) == 2 {
			if *testOptions[0].Value[0] != "second" {
				t.Errorf("First comma value not correct: %s", *testOptions[0].Value[0])
			}
			if *testOptions[0].Value[1] != "third" {
				t.Errorf("First comma value not correct: %s", *testOptions[0].Value[1])
			}
		} else {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}
}

// Test equal option, with and without comma.
func TestParseLineModelOptionsEqual(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "testDevice 0100   equal=value   ", pos: 0}
	err := line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 1:
		if testOptions[0].Name != "equal" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "value" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[0].EqualOpt)
		}
		if len(testOptions[0].Value) != 0 {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}

	resetTest()
	line = optionLine{line: "testDevice 0100   param=opt second   ", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 2:
		if testOptions[0].Name != "param" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "opt" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[0].EqualOpt)
		}
		if len(testOptions[0].Value) != 0 {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
		if testOptions[1].Name != "second" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[1].Name)
		}
		if testOptions[1].EqualOpt != "" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[1].EqualOpt)
		}
		if len(testOptions[1].Value) != 0 {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[1].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}

	resetTest()
	line = optionLine{line: "testDevice 0100   single=second, third # comment", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 1:
		if testOptions[0].Name != "single" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "second" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[0].EqualOpt)
		}
		if len(testOptions[0].Value) == 1 {
			if *testOptions[0].Value[0] != "third" {
				t.Errorf("First comma value not correct: %s", *testOptions[0].Value[0])
			}
		} else {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}
}

// Test equal option, with and without comma.
func TestParseLineModelOptionsQuote(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "testDevice 0100   equal=\"value\"   ", pos: 0}
	err := line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 1:
		if testOptions[0].Name != "equal" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "value" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[0].EqualOpt)
		}
		if len(testOptions[0].Value) != 0 {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}

	resetTest()
	line = optionLine{line: `testDevice 0100   param="Value Second"  `, pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 1:
		if testOptions[0].Name != "param" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "Value Second" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[0].EqualOpt)
		}
		if len(testOptions[0].Value) != 0 {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}

	resetTest()
	line = optionLine{line: "testDevice 0100   paramx=\"option,third fourth\" ,comma  ", pos: 0}
	err = line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 1:
		if testOptions[0].Name != "paramx" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "option,third fourth" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[0].EqualOpt)
		}
		if len(testOptions[0].Value) == 1 {
			if *testOptions[0].Value[0] != "comma" {
				t.Errorf("First comma value not correct: %s", *testOptions[0].Value[0])
			}
		} else {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}
}

// Test equal option, with and without comma. Part2.
func TestParseLineModelOptionsQuote2(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "testDevice 0100   equal=\"value\"  second=another option", pos: 0}
	err := line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 3:
		if testOptions[0].Name != "equal" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "value" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[0].EqualOpt)
		}
		if len(testOptions[0].Value) != 0 {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[0].Value))
		}
		if testOptions[1].Name != "second" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[1].Name)
		}
		if testOptions[1].EqualOpt != "another" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[1].EqualOpt)
		}
		if len(testOptions[1].Value) != 0 {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[1].Value))
		}
		if testOptions[2].Name != "option" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[2].Name)
		}
		if testOptions[2].EqualOpt != "" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[2].EqualOpt)
		}
		if len(testOptions[2].Value) != 0 {
			t.Errorf("Wrong number of comma options: %d", len(testOptions[2].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}
}

// Test equal quote option, with comma. Part3.
func TestParseLineModelOptionsQuote3(t *testing.T) {
	cleanUpConfig()

	RegisterOption("testoption", modOption)
	RegisterSwitch("testswitch", modSwitch)
	RegisterModel("testDevice", TypeModel, modDevice)

	line := optionLine{line: "testDevice 0100   equal=\"value\",extra  second=another option,extra", pos: 0}
	err := line.parseLine()
	if err != nil {
		t.Errorf("ParseLine failed to parse address")
	}
	if testType != "model" {
		t.Errorf("ParseLine did not create a option")
	}
	if testDevNum != 0x100 {
		t.Errorf("Model set address to %04x\n", testDevNum)
	}
	switch len(testOptions) {
	case 0:
		t.Errorf("ParseLine did not give device any options")
	case 3:
		if testOptions[0].Name != "equal" {
			t.Errorf("ParseLine did not give correct option: %s", testOptions[0].Name)
		}
		if testOptions[0].EqualOpt != "value" {
			t.Errorf("ParseLine did not give = value: '%s'", testOptions[0].EqualOpt)
		}
		if len(testOptions[0].Value) == 1 {
			if *testOptions[0].Value[0] != "extra" {
				t.Errorf("First comma value not correct: %s", *testOptions[0].Value[0])
			}
		} else {
			t.Errorf("Wrong number of comma options (equal): %d", len(testOptions[0].Value))
		}
		if testOptions[1].Name != "second" {
			t.Errorf("ParseLine did not give second correct option: %s", testOptions[1].Name)
		}
		if testOptions[1].EqualOpt != "another" {
			t.Errorf("ParseLine did not give second = value: '%s'", testOptions[1].EqualOpt)
		}
		if len(testOptions[1].Value) != 0 {
			t.Errorf("Wrong number of comma second options: %d", len(testOptions[1].Value))
		}
		if testOptions[2].Name != "option" {
			t.Errorf("ParseLine did not give third correct option: %s", testOptions[2].Name)
		}
		if testOptions[2].EqualOpt != "" {
			t.Errorf("ParseLine did not give third = value: '%s'", testOptions[2].EqualOpt)
		}
		if len(testOptions[0].Value) == 1 {
			if *testOptions[0].Value[0] != "extra" {
				t.Errorf("First third comma value not correct: %s", *testOptions[0].Value[0])
			}
		} else {
			t.Errorf("Wrong number of third comma options (equal): %d", len(testOptions[0].Value))
		}
	default:
		t.Errorf("ParseLine gave device some extra options: %d", len(testOptions))
	}
}
