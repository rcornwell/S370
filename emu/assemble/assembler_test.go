/*
	   IBM 370 Assembler Test routines.

		Copyright (c) 2024, Richard Cornwell

		Permission is hereby granted, free of charge, to any person obtaining a
		copy of this software and associated documentation files (the "Software"),
		to deal in the Software without restriction, including without limitation
		the rights to use, copy, modify, merge, publish, distribute, sublicense,
		and/or sell copies of the Software, and to permit persons to whom the
		Software is furnished to do so, subject to the following conditions:

		The above copyright notice and this permission notice shall be included in
		all copies or substantial portions of the Software.

		THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
		IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
		FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL
		RICHARD CORNWELL BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
		IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
		CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package assembler

import (
	"bytes"
	"fmt"
	"testing"

	op "github.com/rcornwell/S370/emu/opcodemap"
)

func printBytes(b []byte) string {
	text := ""
	for _, by := range b {
		text += fmt.Sprintf("%02x, ", by)
	}
	if text != "" {
		text = text[:len(text)-2]
	}
	return text
}

func TestAssemble(t *testing.T) {
	// Test empty string value.
	test := " "
	inst, err := Assemble(test)
	if err == nil {
		t.Error("Empty instruction did not return error")
	} else if err.Error() != "undefined opcode " {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = ""
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Empty instruction did not return error")
	} else if err.Error() != "undefined opcode " {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = "ABC"
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Empty instruction did not return error")
	} else if err.Error() != "undefined opcode ABC" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

}

func TestAssembleRR(t *testing.T) {
	// Test correct value.
	test := " AR 1,2"
	match := []byte{op.OpAR, 0x12}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	// Test without operands.
	test = " AR"
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for AR" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleSVC(t *testing.T) {
	test := " SVC  10  "
	match := []byte{op.OpSVC, 0x10}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "SVC 5 , 30  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction SVC" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = "SVC 100  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Invalid immediate value for SVC" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	// Test without operands.
	test = " SVC"
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for SVC" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleSPM(t *testing.T) {
	test := " SPM  A  "
	match := []byte{op.OpSPM, 0xA}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "SPM 5 , 30  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction SPM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " SPM  10  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Invalid immediate value for SPM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	// Test without operands.
	test = " SPM   "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for SPM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleRX(t *testing.T) {
	test := " L 8,0100(1,2)"
	match := []byte{op.OpL, 0x81, 0x21, 0x00}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "L 5,0200"
	match = []byte{op.OpL, 0x50, 0x02, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "L 7,0a0(5)"
	match = []byte{op.OpL, 0x70, 0x50, 0xA0}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = " L 20,200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "register values out of range L" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " L 10,1000  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "displacment out of range L" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleS(t *testing.T) {
	test := " ssm 0a00"
	match := []byte{op.OpSSM, 0, 0x0A, 0x00}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}
	test = " SSM 0100 ( 1 ) "
	match = []byte{op.OpSSM, 0x00, 0x11, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = " SSM 20,200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction SSM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " SSM 1000  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "displacment out of range SSM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " SSM 0(1,2)  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for SSM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleRS1(t *testing.T) {
	test := " SRA 8,0100(1)"
	match := []byte{op.OpSRA, 0x80, 0x11, 0x00}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "SRA 5,0200"
	match = []byte{op.OpSRA, 0x50, 0x02, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "SRA 7,0a0(5)"
	match = []byte{op.OpSRA, 0x70, 0x50, 0xA0}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = " SRA 20,200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "register values out of range SRA" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " SRA 10,1000  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "displacment out of range SRA" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " SRA 5,0(1,2)  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for SRA" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleRS2(t *testing.T) {
	test := " LM 8,5,0100(1)"
	match := []byte{op.OpLM, 0x85, 0x11, 0x00}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = " LM 8,5,0100 ( 1 ) "
	match = []byte{op.OpLM, 0x85, 0x11, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "LM 5,10,0200"
	match = []byte{op.OpLM, 0x5A, 0x02, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = " LM 20,200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for LM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " LM 10,1000  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for LM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " LM 5,0(1,2)  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for LM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " LM 5,20,0  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "register values out of range LM" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleSI(t *testing.T) {
	test := " CLI 0100(3), 45"
	match := []byte{op.OpCLI, 0x45, 0x31, 0x00}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "CLI 0200, 50  "
	match = []byte{op.OpCLI, 0x50, 0x02, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = " CLI 200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for CLI" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " CLI 1000,10  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "displacment out of range CLI" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " CLI 0(1,2),50  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for CLI" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " CLI 5,20,0  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction CLI" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleSS1(t *testing.T) {
	test := " MVC 0100(3), 45"
	match := []byte{op.OpMVC, 0x03, 0x01, 0x00, 0x00, 0x45}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "MVC 0200, 50  "
	match = []byte{op.OpMVC, 0x00, 0x02, 0x00, 0x00, 0x50}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "MVC 0200(50), 100  "
	match = []byte{op.OpMVC, 0x32, 0x02, 0x00, 0x01, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "MVC 0200(25, 2), 100  "
	match = []byte{op.OpMVC, 0x19, 0x22, 0x00, 0x01, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "MVC 0200(10, 3), 100(5)  "
	match = []byte{op.OpMVC, 0x0a, 0x32, 0x00, 0x51, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "MVC 0200(256), 100(5)  "
	match = []byte{op.OpMVC, 0x0a, 0x32, 0x00, 0x51, 0x00}
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "length value out of range MVC" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " MVC 200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for MVC" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " MVC 1000,10  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "displacment out of range MVC" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " MVC 5,20,0  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction MVC" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleSS2(t *testing.T) {
	test := " AP 0100(3), 45"
	match := []byte{op.OpAP, 0x30, 0x01, 0x00, 0x00, 0x45}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "AP 0200, 50  "
	match = []byte{op.OpAP, 0x00, 0x02, 0x00, 0x00, 0x50}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "AP 0200(4), 100(10)  "
	match = []byte{op.OpAP, 0x4A, 0x02, 0x00, 0x01, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "AP 0200(6, 2), 100  "
	match = []byte{op.OpAP, 0x60, 0x22, 0x00, 0x01, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "AP 0200(10, 3), 100(5)  "
	match = []byte{op.OpAP, 0xa5, 0x32, 0x00, 0x01, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "AP 0400(6, 3), 100(10,8)  "
	match = []byte{op.OpAP, 0x6a, 0x34, 0x00, 0x81, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = "AP 0200(30), 100(5)  "
	match = []byte{op.OpAP, 0x0a, 0x32, 0x00, 0x51, 0x00}
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "length value out of range AP" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " AP 200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for AP" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " AP 1000,10  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "displacment out of range AP" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " AP 5,20,0  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction AP" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleRRE1(t *testing.T) {
	test := " stpt 0a00"
	match := []byte{0xb2, 0x9, 0x0A, 0x00}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}
	test = " STPT 0100 ( 1 ) "
	match = []byte{0xb2, 0x09, 0x11, 0x00}
	inst, err = Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	test = " STPT 20,200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction STPT" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " STPT 1000  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "displacment out of range STPT" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " STPT 0(1,2)  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "invalid format for STPT" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}
}

func TestAssembleRRE2(t *testing.T) {
	test := " ptlb"
	match := []byte{0xb2, 0xD, 0x00, 0x00}
	inst, err := Assemble(test)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(match, inst) {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
	}
	if len(match) != len(inst) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}
	test = " PTLB 0100 ( 1 ) "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction PTLB" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " PTLB 20,200  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction PTLB" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " PTLB 1000  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction PTLB" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

	test = " PTLB 0(1,2)  "
	inst, err = Assemble(test)
	if err == nil {
		t.Error("Invalid format did not return error")
	} else if err.Error() != "Extra data after instruction PTLB" {
		t.Error("Wrong error message: " + err.Error())
	}
	if len(inst) != 0 {
		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
	}

}
