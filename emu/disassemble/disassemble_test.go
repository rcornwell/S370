/*
	   IBM 370 Disassembler Test routines.

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
package disassembler

import (
	"testing"

	op "github.com/rcornwell/S370/emu/opcodemap"
)

// func TestDisassemble(t *testing.T) {
// 	// Test empty string value.
// 	test := []byte{0xa0, 0x00}
// 	inst, err := Disassemble(test)
// 	if err == nil {
// 		t.Error("Empty instruction did not return error")
// 	} else if err.Error() != "undefined opcode " {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// 	test = ""
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Empty instruction did not return error")
// 	} else if err.Error() != "undefined opcode " {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// 	test = "ABC"
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Empty instruction did not return error")
// 	} else if err.Error() != "undefined opcode ABC" {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// }

func TestDisassembleRR(t *testing.T) {
	// Test correct value.
	match := "AR    1,2"
	test := []byte{op.OpAR, 0x12}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 2 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 2)
	}

	// Test correct value.
	match = "AR    5,6"
	test = []byte{op.OpAR, 0x56, 0x00, 0x10}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 2 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 2)
	}

	// Test correct value.
	match = "1A 56                 AR    5,6"
	test = []byte{op.OpAR, 0x56, 0x00, 0x10}
	inst, length = PrintInst(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 2 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 2)
	}
}

func TestDisassembleSVC(t *testing.T) {
	match := "SVC   10"
	test := []byte{op.OpSVC, 0x10}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst  Got: " + inst + " Expected " + match)
	}
	if length != 2 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 2)
	}

	match = "SVC   10"
	test = []byte{op.OpSVC, 0x10, 0x00, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 2 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 2)
	}
}

func TestDisassembleSPM(t *testing.T) {
	match := "SPM   A"
	test := []byte{op.OpSPM, 0xA0}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 2 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 2)
	}

	match = "SPM   5"
	test = []byte{op.OpSPM, 0x51}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 2 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 2)
	}

	match = "SPM   6"
	test = []byte{op.OpSPM, 0x60, 0x50, 0x40}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 2 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 2)
	}
}

func TestDisassembleRX(t *testing.T) {
	match := "L     8,100(1,2)"
	test := []byte{op.OpL, 0x81, 0x21, 0x00}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != len(test) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	match = "L     5,200"
	test = []byte{op.OpL, 0x50, 0x02, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != len(test) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	match = "L     7,0A0(5)"
	test = []byte{op.OpL, 0x70, 0x50, 0xA0}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != len(test) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}

	match = "58 7 5 0A0            L     7,0A0(4,5)"
	test = []byte{op.OpL, 0x74, 0x50, 0xA0}
	inst, length = PrintInst(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != len(test) {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
	}
}

func TestDisassembleS(t *testing.T) {
	match := "SSM   A00"
	test := []byte{op.OpSSM, 0, 0x0A, 0x00}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "SSM   100(1)"
	test = []byte{op.OpSSM, 0x00, 0x11, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}
}

func TestDisassembleRS1(t *testing.T) {
	match := "SRA   8,100(1)"
	test := []byte{op.OpSRA, 0x80, 0x11, 0x00}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "SRA   5,200"
	test = []byte{op.OpSRA, 0x50, 0x02, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "SRA   10,100(12)"
	test = []byte{op.OpSRA, 0xA0, 0xc1, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "SRA   10,100(12)"
	test = []byte{op.OpSRA, 0xA6, 0xc1, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}
}

func TestDisassembleRS2(t *testing.T) {
	match := "LM    8,5,100(1)"
	test := []byte{op.OpLM, 0x85, 0x11, 0x00}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "LM    8,5,200"
	test = []byte{op.OpLM, 0x85, 0x02, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "LM    5,10,200(12)"
	test = []byte{op.OpLM, 0x5A, 0xC2, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}
}

func TestDisassembleCLI(t *testing.T) {
	match := "CLI   100(1),85"
	test := []byte{op.OpCLI, 0x85, 0x11, 0x00}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "CLI   200,45"
	test = []byte{op.OpCLI, 0x45, 0x02, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "CLI   200(12),5A"
	test = []byte{op.OpCLI, 0x5A, 0xC2, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}
}

func TestDisassembleSS1(t *testing.T) {
	match := "MVC   100(3),045"
	test := []byte{op.OpMVC, 0x03, 0x01, 0x00, 0x00, 0x45}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "MVC   200(0),050"
	test = []byte{op.OpMVC, 0x00, 0x02, 0x00, 0x00, 0x50}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "MVC   200(50),100"
	test = []byte{op.OpMVC, 0x32, 0x02, 0x00, 0x01, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "MVC   200(0,5),050"
	test = []byte{op.OpMVC, 0x00, 0x52, 0x00, 0x00, 0x50}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "MVC   200(50,7),100"
	test = []byte{op.OpMVC, 0x32, 0x72, 0x00, 0x01, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "MVC   200(0),050(5)"
	test = []byte{op.OpMVC, 0x00, 0x02, 0x00, 0x50, 0x50}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "MVC   200(50),100(7)"
	test = []byte{op.OpMVC, 0x32, 0x02, 0x00, 0x71, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "MVC   200(0,10),050(5)"
	test = []byte{op.OpMVC, 0x00, 0xA2, 0x00, 0x50, 0x50}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "MVC   200(50,7),100(9)"
	test = []byte{op.OpMVC, 0x32, 0x72, 0x00, 0x91, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "D2 32  7 200 9 100    MVC   200(50,7),100(9)"
	test = []byte{op.OpMVC, 0x32, 0x72, 0x00, 0x91, 0x00}
	inst, length = PrintInst(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}
}

func TestDisassembleSS2(t *testing.T) {
	match := "AP    100(3),045(0)"
	test := []byte{op.OpAP, 0x30, 0x01, 0x00, 0x00, 0x45}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "AP    200(0),050(0)"
	test = []byte{op.OpAP, 0x00, 0x02, 0x00, 0x00, 0x50}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "AP    200(4),100(10)"
	test = []byte{op.OpAP, 0x4A, 0x02, 0x00, 0x01, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "AP    200(10,3),100(5)"
	test = []byte{op.OpAP, 0xa5, 0x32, 0x00, 0x01, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "AP    400(6,3),100(10,8)"
	test = []byte{op.OpAP, 0x6a, 0x34, 0x00, 0x81, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}

	match = "FA 6A  3 400 8 100    AP    400(6,3),100(10,8)"
	test = []byte{op.OpAP, 0x6a, 0x34, 0x00, 0x81, 0x00}
	inst, length = PrintInst(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 6 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 6)
	}
}

func TestDisassembleRRE(t *testing.T) {
	match := "STPT  A00"
	test := []byte{0xb2, 0x9, 0x0A, 0x00}
	inst, length := Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "STPT  100(1)"
	test = []byte{0xb2, 0x09, 0x11, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "PTLB  "
	test = []byte{0xb2, 0xD, 0x00, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}

	match = "PTLB  "
	test = []byte{0xb2, 0x0D, 0x11, 0x00}
	inst, length = Disassemble(test)
	if match != inst {
		t.Error("Inst Got: " + inst + " Expected " + match)
	}
	if length != 4 {
		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), 4)
	}
}

// func TestDisassembleRRE1(t *testing.T) {
// 	test := " stpt 0a00"
// 	match := []byte{0xb2, 0x9, 0x0A, 0x00}
// 	inst, err := Disassemble(test)
// 	if err != nil {
// 		t.Error(err.Error())
// 	}
// 	if !bytes.Equal(match, inst) {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
// 	}
// 	if len(match) != len(inst) {
// 		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
// 	}
// 	test = " STPT 0100 ( 1 ) "
// 	match = []byte{0xb2, 0x09, 0x11, 0x00}
// 	inst, err = Disassemble(test)
// 	if err != nil {
// 		t.Error(err.Error())
// 	}
// 	if !bytes.Equal(match, inst) {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
// 	}
// 	if len(match) != len(inst) {
// 		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
// 	}

// 	test = " STPT 20,200  "
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Invalid format did not return error")
// 	} else if err.Error() != "Extra data after instruction STPT" {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// 	test = " STPT 1000  "
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Invalid format did not return error")
// 	} else if err.Error() != "displacment out of range STPT" {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// 	test = " STPT 0(1,2)  "
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Invalid format did not return error")
// 	} else if err.Error() != "invalid format for STPT" {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}
// }

// func TestDisassembleRRE2(t *testing.T) {
// 	test := " ptlb"
// 	match := []byte{0xb2, 0xD, 0x00, 0x00}
// 	inst, err := Disassemble(test)
// 	if err != nil {
// 		t.Error(err.Error())
// 	}
// 	if !bytes.Equal(match, inst) {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected " + printBytes(match))
// 	}
// 	if len(match) != len(inst) {
// 		t.Errorf("Returned wrong number of bytes: %d expected: %d", len(inst), len(match))
// 	}
// 	test = " PTLB 0100 ( 1 ) "
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Invalid format did not return error")
// 	} else if err.Error() != "Extra data after instruction PTLB" {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// 	test = " PTLB 20,200  "
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Invalid format did not return error")
// 	} else if err.Error() != "Extra data after instruction PTLB" {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// 	test = " PTLB 1000  "
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Invalid format did not return error")
// 	} else if err.Error() != "Extra data after instruction PTLB" {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// 	test = " PTLB 0(1,2)  "
// 	inst, err = Disassemble(test)
// 	if err == nil {
// 		t.Error("Invalid format did not return error")
// 	} else if err.Error() != "Extra data after instruction PTLB" {
// 		t.Error("Wrong error message: " + err.Error())
// 	}
// 	if len(inst) != 0 {
// 		t.Error("Inst: '" + test + "' Got: " + printBytes(inst) + " Expected empty")
// 	}

// }
