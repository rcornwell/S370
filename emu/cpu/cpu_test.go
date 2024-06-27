/*
 * S370 CPU test cases.
 *
 * Copyright 2024, Richard Cornwell
 *                 Original test cases by Ken Shirriff
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (thLoe "Software"), to deal
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

package cpu

import (
	"encoding/hex"
	"math"
	"math/rand"
	"testing"

	"github.com/rcornwell/S370/emu/memory"
)

const (
	testCycles int    = 1000
	HDMASK     uint64 = 0xffffffff80000000
	FDMASK     uint64 = 0x00000000ffffffff
)

// Get a short floating point register.
func getFloatShort(num int) uint32 {
	v := cpuState.fpregs[num&0x6]
	if (num & 1) == 0 {
		v >>= 32
	}
	return uint32(v & LMASKL)
}

// Set a short floating point register.
func setFloatShort(num int, v uint32) {
	n := num & 6
	if (num & 1) == 0 {
		cpuState.fpregs[n] = (uint64(v) << 32) | (cpuState.fpregs[n] & LMASKL)
	} else {
		cpuState.fpregs[n] = (cpuState.fpregs[n] & HMASKL) | uint64(v)
	}
}

// Get load floating point register.
func getFloatLong(num int) uint64 {
	return cpuState.fpregs[num]
}

// Get load floating point register.
func setFloatLong(num int, v uint64) {
	cpuState.fpregs[num] = v
}

// Convert a floating point value to a 64-bit FP register.
func floatToFpreg(num int, val float64) bool {
	var s uint64
	var char uint8 = 64

	// Quick exit if zero
	if val == 0 {
		setFloatLong(num, 0)
		return true
	}

	// Extract sign
	if val < 0 {
		s = MSIGNL
		val = -val
	}

	// Determine exponent
	for val >= 1 && char < 128 {
		char++
		val /= 16
	}

	for val < 1/16. && char >= 0 {
		char--
		val *= 16
	}

	if char < 0 || char >= 128 {
		return false
	}

	val *= 1 << 24
	f := s | (uint64(char) << 56) | (uint64(val) << 32)
	f |= uint64((val - float64(uint32(val))) * float64((uint64(1) << 32)))
	setFloatLong(num, f)
	return true
}

// load floating point short register as float64
func cnvtShortFloat(num int) float64 {
	t64 := getFloatLong(num)
	e := float64((t64>>56)&0x7f) - 64.0
	d := float64(0x00ffffff00000000 & t64)
	d *= math.Exp2(-56.0 + 4.0*e)
	if (MSIGNL & t64) != 0 {
		d *= -1.0
	}
	return d
}

// load floating point long register as float64
func cnvtLongFloat(num int) float64 {
	t64 := getFloatLong(num)
	e := float64((t64>>56)&0x7f) - 64.0
	d := float64(MMASKL & t64)
	d *= math.Exp2(-56.0 + 4.0*e)
	if (MSIGNL & t64) != 0 {
		d *= -1.0
	}
	return d
}

func TestFloatConv(t *testing.T) {
	err := floatToFpreg(0, 0.0)
	if !err {
		t.Error("Unable to convert 0.0")
	}
	r := getFloatLong(0)
	if r != 0 {
		t.Errorf("Wrong conversion of 0.0 got: %016x", r)
	}
	err = floatToFpreg(0, 1.0)
	if !err {
		t.Error("Unable to convert 1.0")
	}
	r = getFloatLong(0)
	if r != 0x4110000000000000 {
		t.Errorf("Wrong conversion of 1.0 got: %016x", r)
	}
	err = floatToFpreg(0, 0.5)
	if !err {
		t.Error("Unable to convert 0.5")
	}
	r = getFloatLong(0)
	if r != 0x4080000000000000 {
		t.Errorf("Wrong conversion of 0.5 got: %016x", r)
	}
	err = floatToFpreg(0, 1.0/64.0)
	if !err {
		t.Error("Unable to convert 1/64")
	}
	r = getFloatLong(0)
	if r != 0x3f40000000000000 {
		t.Errorf("Wrong conversion of 1/64 got: %016x", r)
	}
	err = floatToFpreg(0, -15.0)
	if !err {
		t.Error("Unable to convert -15.0")
	}
	r = getFloatLong(0)
	if r != 0xc1f0000000000000 {
		t.Errorf("Wrong conversion of -15.0 got: %016x", r)
	}
}

func TestShortConv(t *testing.T) {
	setFloatShort(0, 0xff000000)
	setFloatShort(1, 0)
	r := cnvtShortFloat(0)
	if r != 0.0 {
		t.Errorf("Wrong conversion of 0.0 got: %f", r)
	}
	setFloatShort(0, 0x41100000)
	r = cnvtShortFloat(0)
	setFloatShort(1, 0)
	if r != 1.0 {
		t.Errorf("Wrong conversion of 1.0 got: %f", r)
	}
	setFloatShort(0, 0x40800000)
	r = cnvtShortFloat(0)
	if r != 0.5 {
		t.Errorf("Wrong conversion of 0.5 got: %f", r)
	}
	setFloatShort(0, 0x3f400000)
	r = cnvtShortFloat(0)
	if r != (1.0 / 64.0) {
		t.Errorf("Wrong conversion of 1.0/64.0 got: %f", r)
	}
	setFloatShort(0, 0xc1f00000)
	r = cnvtShortFloat(0)
	if r != -15.0 {
		t.Errorf("Wrong conversion of -15.0 got: %f", r)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f = math.Ldexp(f, scale)
		err := floatToFpreg(0, f)
		if !err {
			t.Errorf("Unable to set register to %f", f)
		}
		r := cnvtShortFloat(0)
		ratio := math.Abs((r - f) / f)
		if ratio > 0.000001 {
			t.Errorf("Conversion short failed got: %f expected: %f", r, f)
		}
	}
}

func TestLongConv(t *testing.T) {
	setFloatShort(0, 0xff000000)
	setFloatShort(1, 0)
	r := cnvtLongFloat(0)
	if r != 0.0 {
		t.Errorf("Wrong conversion of 0.0 got: %f", r)
	}
	setFloatShort(0, 0x41100000)
	r = cnvtLongFloat(0)
	if r != 1.0 {
		t.Errorf("Wrong conversion of 1.0 got: %f", r)
	}
	setFloatShort(0, 0x40800000)
	r = cnvtLongFloat(0)
	if r != 0.5 {
		t.Errorf("Wrong conversion of 0.5 got: %f", r)
	}
	setFloatShort(0, 0x3f400000)
	r = cnvtLongFloat(0)
	if r != (1.0 / 64.0) {
		t.Errorf("Wrong conversion of 1.0/64.0 got: %f", r)
	}
	setFloatShort(0, 0xc1f00000)
	r = cnvtLongFloat(0)
	if r != -15.0 {
		t.Errorf("Wrong conversion of -15.0 got: %f", r)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f = math.Ldexp(f, scale)
		err := floatToFpreg(0, f)
		if !err {
			t.Errorf("Unable to set register to %f", f)
		}
		r := cnvtLongFloat(0)
		ratio := math.Abs((r - f) / f)
		if ratio > 0.000001 {
			t.Errorf("Conversion long failed got: %f expected: %f", r, f)
		}
	}
}

// Roughly test characteristics of random number generator
func TestRandFloat(t *testing.T) {
	pos := 0
	neg := 0
	big := 0
	small := 0

	// Test add logical with random values
	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f = math.Ldexp(f, scale)
		if f < 0.0 {
			neg++
		} else {
			pos++
		}
		if math.Abs(f) > math.Pow(2.0, 20.0) {
			big++
		} else if math.Abs(f) < math.Pow(2.0, -20.0) {
			small++
		}
	}

	if big < 200 {
		t.Errorf("Less then 200 big numbers: %d", big)
	}
	if small < 200 {
		t.Errorf("Less then 200 small numbers: %d", small)
	}
	if pos < 400 {
		t.Errorf("Less then 400 pos numbers: %d", pos)
	}
	if neg < 400 {
		t.Errorf("Less then 400 neg numbers: %d", neg)
	}
}

var trapFlag bool

func setup() {
	memory.SetSize(64)
	InitializeCPU()
	cpuState.cc = 3
}

func (cpu *cpu) testInst(mask uint8) {
	cpu.PC = 0x400
	cpu.progMask = mask & 0xf
	memory.SetMemory(0x68, 0)
	memory.SetMemory(0x6c, 0x800)
	trapFlag = false
	for range 20 {
		_ = CycleCPU()

		if cpu.PC == 0x800 {
			trapFlag = true
		}
		// Stop it next opcode = 0
		w := memory.GetMemory(cpu.PC)
		if (cpu.PC & 2) == 0 {
			w >>= 16
		}
		if (w & 0xffff) == 0 {
			break
		}
	}
}

// Test LR instruction.
func TestCycleLR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x18310000) // LR 3,1
	cpuState.regs[1] = 0x12345678
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x12345678 {
		t.Errorf("LR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x12345678)
	}
	if cpuState.regs[1] != 0x12345678 {
		t.Errorf("LR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x12345678)
	}
	if cpuState.cc != 3 {
		t.Errorf("LR CC changed got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test LTR instruction.
func TestCycleLTR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x12340000) // LTR 3,4
	// Test negative number
	cpuState.regs[4] = 0xcdef1234
	cpuState.testInst(0)
	if cpuState.regs[3] != 0xcdef1234 {
		t.Errorf("LTR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x12345678)
	}
	if cpuState.regs[4] != 0xcdef1234 {
		t.Errorf("LTR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0xcdef1234)
	}
	if cpuState.cc != 1 {
		t.Errorf("LTR CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Test zero
	cpuState.regs[4] = 0x00000000
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x00000000 {
		t.Errorf("LTR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000000)
	}
	if cpuState.regs[4] != 0x00000000 {
		t.Errorf("LTR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("LTR CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Test positive
	cpuState.regs[4] = 0x12345678
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x12345678 {
		t.Errorf("LTR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x12345678)
	}
	if cpuState.regs[4] != 0x12345678 {
		t.Errorf("LTR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x12345678)
	}
	if cpuState.cc != 2 {
		t.Errorf("LTR CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Test LCR instruction.
func TestCycleLCR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x13340000) // LCR 3,4

	// Test positive
	cpuState.regs[4] = 0x00001000
	cpuState.testInst(0)
	if cpuState.regs[3] != 0xfffff000 {
		t.Errorf("LCR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0xfffff000)
	}
	if cpuState.regs[4] != 0x00001000 {
		t.Errorf("LCR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x00001000)
	}
	if cpuState.cc != 1 {
		t.Errorf("LCR CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Test negative number
	cpuState.regs[4] = 0xffffffff
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x00000001 {
		t.Errorf("LCR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000001)
	}
	if cpuState.regs[4] != 0xffffffff {
		t.Errorf("LCR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0xffffffff)
	}
	if cpuState.cc != 2 {
		t.Errorf("LCR CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Test zero
	cpuState.regs[4] = 0x00000000
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x00000000 {
		t.Errorf("LCR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000000)
	}
	if cpuState.regs[4] != 0x00000000 {
		t.Errorf("LCR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("LCR CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Test overflow
	cpuState.regs[4] = 0x80000000
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x80000000 {
		t.Errorf("LCR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x80000000)
	}
	if cpuState.regs[4] != 0x80000000 {
		t.Errorf("LCR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x80000000)
	}
	if cpuState.cc != 3 {
		t.Errorf("LCR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test LPR instruction.
func TestCycleLPR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x10340000) // LPR 3,4

	// Test positive
	cpuState.regs[4] = 0x00000001
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x00000001 {
		t.Errorf("LPR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000001)
	}
	if cpuState.regs[4] != 0x00000001 {
		t.Errorf("LPR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x00000001)
	}
	if cpuState.cc != 2 {
		t.Errorf("LPR CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Test negative number
	cpuState.regs[4] = 0xffffffff
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x00000001 {
		t.Errorf("LPR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000001)
	}
	if cpuState.regs[4] != 0xffffffff {
		t.Errorf("LPR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0xffffffff)
	}
	if cpuState.cc != 2 {
		t.Errorf("LPR CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Test zero
	cpuState.regs[4] = 0x00000000
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x00000000 {
		t.Errorf("LPR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000000)
	}
	if cpuState.regs[4] != 0x00000000 {
		t.Errorf("LPR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("LPR CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Test overflow
	cpuState.regs[4] = 0x80000000
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x80000000 {
		t.Errorf("LPR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x80000000)
	}
	if cpuState.regs[4] != 0x80000000 {
		t.Errorf("LPR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x80000000)
	}
	if cpuState.cc != 3 {
		t.Errorf("LPR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test LNR instruction.
func TestCycleLNR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x11340000) // LNR 3,4

	// Test positive
	cpuState.regs[4] = 0x00000001
	cpuState.testInst(0)
	if cpuState.regs[3] != 0xffffffff {
		t.Errorf("LNR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0xffffffff)
	}
	if cpuState.regs[4] != 0x00000001 {
		t.Errorf("LNR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x00000001)
	}
	if cpuState.cc != 1 {
		t.Errorf("LNR CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Test negative number
	cpuState.regs[4] = 0xffffffff
	cpuState.testInst(0)
	if cpuState.regs[3] != 0xffffffff {
		t.Errorf("LNR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0xffffffff)
	}
	if cpuState.regs[4] != 0xffffffff {
		t.Errorf("LNR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0xffffffff)
	}
	if cpuState.cc != 1 {
		t.Errorf("LNR CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Test zero
	cpuState.regs[4] = 0x00000000
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x00000000 {
		t.Errorf("LNR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000000)
	}
	if cpuState.regs[4] != 0x00000000 {
		t.Errorf("LNR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("LNR CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Test overflow
	cpuState.regs[4] = 0x80000000
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x80000000 {
		t.Errorf("LNR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x80000000)
	}
	if cpuState.regs[4] != 0x80000000 {
		t.Errorf("LNR register 4 was incorrect got: %08x wanted: %08x", cpuState.regs[4], 0x80000000)
	}
	if cpuState.cc != 1 {
		t.Errorf("LNR CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Test L instruction.
func TestCycleL(t *testing.T) {
	setup()
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1000
	cpuState.regs[5] = 0x200
	memory.SetMemory(0x1b84, 0x12345678)
	memory.SetMemory(0x400, 0x58345984) // L 3,984(4,5)
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x12345678 {
		t.Errorf("L register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x12345678)
	}
	if cpuState.cc != 3 {
		t.Errorf("L CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Test negative number
	cpuState.cc = 3
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1000
	cpuState.regs[5] = 0x200
	memory.SetMemory(0x1b84, 0x92345678)
	memory.SetMemory(0x400, 0x58345984) // L 3,984(4,5)
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x92345678 {
		t.Errorf("L register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x92345678)
	}
	if cpuState.cc != 3 {
		t.Errorf("L CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Test zero
	cpuState.cc = 3
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1000
	cpuState.regs[5] = 0x200
	memory.SetMemory(0x1984, 0x00000000)
	memory.SetMemory(0x400, 0x58340984) // L 3,984(4)
	cpuState.testInst(0)
	if cpuState.regs[3] != 0x00000000 {
		t.Errorf("L register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000000)
	}
	if cpuState.cc != 3 {
		t.Errorf("L CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Test load off alignment
	cpuState.cc = 3
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1001
	cpuState.regs[5] = 0x200
	memory.SetMemory(0x1b84, 0xff123456)
	memory.SetMemory(0x1b88, 0x78ffffff)
	memory.SetMemory(0x400, 0x58345984) // L 3,984(4,5)
	cpuState.testInst(0)

	if cpuState.regs[3] != 0x12345678 {
		t.Errorf("L register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x12345678)
	}
	if cpuState.cc != 3 {
		t.Errorf("L CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1002
	cpuState.regs[5] = 0x200
	memory.SetMemory(0x1984, 0xffff1234)
	memory.SetMemory(0x1988, 0x5678ffff)
	memory.SetMemory(0x400, 0x58340984) // L 3,984(4)
	cpuState.testInst(0)

	if cpuState.regs[3] != 0x12345678 {
		t.Errorf("L register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x12345678)
	}
	if cpuState.cc != 3 {
		t.Errorf("L CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1003
	cpuState.regs[5] = 0x200
	memory.SetMemory(0x1984, 0xffffff12)
	memory.SetMemory(0x1988, 0x345678ff)
	memory.SetMemory(0x400, 0x58304984) // L 3,984(4)
	cpuState.testInst(0)

	if cpuState.regs[3] != 0x12345678 {
		t.Errorf("L register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x12345678)
	}
	if cpuState.cc != 3 {
		t.Errorf("L CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test Add register.
func TestCycleA(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1A120000) // AR 1,2

	// Test positive
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x00000005
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x1234567d {
		t.Errorf("AR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x1234567d)
	}
	if cpuState.regs[2] != 0x00000005 {
		t.Errorf("AR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x00000005)
	}
	if cpuState.cc != 2 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	// Test negative number
	cpuState.regs[1] = 0x81234567
	cpuState.regs[2] = 0x00000001
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x81234568 {
		t.Errorf("AR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x81234568)
	}
	if cpuState.regs[2] != 0x00000001 {
		t.Errorf("AR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x00000001)
	}
	if cpuState.cc != 1 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	// Test zero
	cpuState.regs[1] = 0x00000002
	cpuState.regs[2] = 0xfffffffe
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("AR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.regs[2] != 0xfffffffe {
		t.Errorf("AR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0xfffffffe)
	}
	if cpuState.cc != 0 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	// Test overflow
	cpuState.regs[1] = 0x7fffffff
	cpuState.regs[2] = 0x00000001
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x80000000 {
		t.Errorf("AR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x80000000)
	}
	if cpuState.regs[2] != 0x00000001 {
		t.Errorf("AR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x00000001)
	}
	if cpuState.cc != 3 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1A121A31) // AR 1,2 AR 3,1
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x00000001
	cpuState.regs[3] = 0x00000010
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x12345679 {
		t.Errorf("AR 2 register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x12345679)
	}
	if cpuState.regs[2] != 0x00000001 {
		t.Errorf("AR 2 register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x00000001)
	}
	if cpuState.regs[3] != 0x12345689 {
		t.Errorf("AR 2 register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x12345689)
	}
	if cpuState.cc != 2 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1a120000) // AR 1,2
	cpuState.regs[1] = 0x7fffffff
	cpuState.regs[2] = 0x00000001
	cpuState.testInst(8)
	psw1 := memory.GetMemory(0x28)
	psw2 := memory.GetMemory(0x2c)
	if !trapFlag {
		t.Errorf("AR 3 did not trap")
	}
	if cpuState.regs[1] != 0x80000000 {
		t.Errorf("AR 3 register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x80000000)
	}
	if cpuState.regs[2] != 0x00000001 {
		t.Errorf("AR 3 register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x00000001)
	}
	if cpuState.cc != 0 {
		t.Errorf("AR 3 CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}
	if psw1 != 0x00000008 {
		t.Errorf("AR 3 psw1 was incorrect got: %08x wanted: %08x", psw1, 0x00000008)
	}
	if psw2 != 0x78000402 {
		t.Errorf("AR 3 psw2 was incorrect got: %08x wanted: %08x", psw2, 0x78000402)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x5a156200) // AR 1,200(5,6)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0x34567890)
	cpuState.testInst(0)
	s := uint32(0x12345678) + uint32(0x34567890)
	if cpuState.regs[1] != s {
		t.Errorf("A register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], s)
	}
	if cpuState.cc != 2 {
		t.Errorf("A CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Test add with random values
	rnum := rand.New(rand.NewSource(42))

	for range testCycles {
		cpuState.cc = 3
		n1 := rnum.Int31()
		n2 := rnum.Int31()
		r := int64(n1) + int64(n2)
		ur := uint64(r)
		sum := uint32(ur & FDMASK)
		cpuState.regs[1] = uint32(n1)
		memory.SetMemory(0x100, uint32(n2))
		memory.SetMemory(0x400, 0x5a100100) // A 1,100(0,0)
		cpuState.testInst(0)

		switch x := r; {
		case x == 0: // Zero
			if cpuState.cc != 0 {
				t.Errorf("A rand not correct got: %x wanted: %x", cpuState.cc, 0)
			}
		case x > 0: // Positive
			if (ur & HDMASK) != 0 {
				if cpuState.cc != 3 {
					t.Errorf("A rand not correct got: %x wanted: %x", cpuState.cc, 3)
				}
				if cpuState.regs[1] != sum {
					t.Errorf("A rand over register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], sum)
				}
				continue
			}
			if cpuState.cc != 2 {
				t.Errorf("A rand not correct got: %x wanted: %x", cpuState.cc, 2)
			}
		default: // Negative
			if (ur & HDMASK) != HDMASK {
				if cpuState.cc != 3 {
					t.Errorf("A rand not correct got: %x wanted: %x", cpuState.cc, 3)
				}
				if cpuState.regs[1] != sum {
					t.Errorf("A rand over register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], sum)
				}
				continue
			}
			if cpuState.cc != 1 {
				t.Errorf("A rand not correct got: %x wanted: %x", cpuState.cc, 1)
			}
		}
		if cpuState.regs[1] != sum {
			t.Errorf("A rand register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], sum)
		}
	}
}

// Second test of Add Half.
func TestCycleAH1(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x4a156200) // AH 1,200(5,6)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000202
	memory.SetMemory(0x500, 0x34567890)
	cpuState.testInst(0)
	s := uint32(0x12345678) + uint32(0x7890)
	if cpuState.regs[1] != s {
		t.Errorf("AH register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], s)
	}
	if cpuState.cc != 2 {
		t.Errorf("AH CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Sign extend
	memory.SetMemory(0x400, 0x4a156200) // AH 1,200(5,6)
	cpuState.regs[1] = 0x00000001
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0xfffe1234)
	cpuState.testInst(0)
	if cpuState.regs[1] != 0xffffffff {
		t.Errorf("AH register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0xffffffff)
	}
	if cpuState.cc != 1 {
		t.Errorf("AH CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Test Add Logical.
func TestCycleAL(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1e120000) // ALR 1,2
	cpuState.regs[1] = 0x00000000
	cpuState.regs[2] = 0x00000000
	cpuState.testInst(0)

	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1e120000) // ALR 1,2
	cpuState.regs[1] = 0xffff0000
	cpuState.regs[2] = 0x00000002
	cpuState.testInst(0)

	if cpuState.regs[1] != 0xffff0002 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0xffff0002)
	}
	if cpuState.cc != 1 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1e120000) // ALR 1,2
	cpuState.regs[1] = 0xfffffffe
	cpuState.regs[2] = 0x00000002
	cpuState.testInst(0)

	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.cc != 2 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1e120000) // ALR 1,2
	cpuState.regs[1] = 0xfffffffe
	cpuState.regs[2] = 0x00000003
	cpuState.testInst(0)

	if cpuState.regs[1] != 0x00000001 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000001)
	}
	if cpuState.cc != 3 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Sign extend
	memory.SetMemory(0x400, 0x5e156200) // AL 1,200(5,6)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0xf0000000)
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x02345678 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x02345678)
	}
	if cpuState.cc != 3 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Test add logical with random values
	rnum := rand.New(rand.NewSource(125))

	for range testCycles {
		cpuState.cc = 3
		n1 := rnum.Int31()
		n2 := rnum.Int31()
		ur := (uint64(n1) & LMASKL) + (uint64(n2) & LMASKL)
		sum := uint32(ur & LMASKL)
		cpuState.regs[1] = uint32(n1)
		memory.SetMemory(0x100, uint32(n2))
		memory.SetMemory(0x400, 0x5e100100) // AL 1,100(0,0)
		cc := uint8(0)
		if (ur & 0x100000000) != 0 {
			cc = 2
			ur &= 0x0ffffffff
		}
		if ur != 0 {
			cc++
		}
		cpuState.testInst(0)

		if cpuState.cc != cc {
			t.Errorf("AL rand not correct got: %x wanted: %x", cpuState.cc, cc)
		}
		if cpuState.regs[1] != sum {
			t.Errorf("AL rand register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], sum)
		}
	}
}

// Test subtract instruction.
func TestCycleS(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1b120000) // SR 1,2
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x00000001
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x12345677 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x12345677)
	}
	if cpuState.cc != 2 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x5b156200) // S 1,200(5,6)
	memory.SetMemory(0x500, 0x12300000)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	cpuState.testInst(0)

	if cpuState.regs[1] != 0x00045678 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00045678)
	}
	if cpuState.cc != 2 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpuState.regs[1] = 0x8fffffff
	cpuState.testInst(0)

	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpuState.regs[1] = 0xffffffff
	cpuState.testInst(0)

	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpuState.regs[1] = 0x80000000
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Test multiply with random values
	rnum := rand.New(rand.NewSource(42))

	for range testCycles {
		cpuState.cc = 3
		n1 := rnum.Int31()
		n2 := rnum.Int31()
		r := n1 - n2
		ur := uint64(r)
		diff := uint32(n1) - uint32(n2)
		cpuState.regs[1] = uint32(n1)
		memory.SetMemory(0x100, uint32(n2))
		memory.SetMemory(0x400, 0x5b100100) // S 1,100(0,0)
		cpuState.testInst(0)

		switch x := r; {
		case x == 0: // Zero
			if cpuState.cc != 0 {
				t.Errorf("S rand not correct got: %x wanted: %x", cpuState.cc, 0)
			}
		case x > 0: // Positive
			if (ur & HDMASK) != 0 {
				if cpuState.cc != 3 {
					t.Errorf("S rand not correct got: %x wanted: %x", cpuState.cc, 3)
				}
				if cpuState.regs[1] != diff {
					t.Errorf("S rand over register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], diff)
				}
				continue
			}
			if cpuState.cc != 2 {
				t.Errorf("S rand not correct got: %x wanted: %x", cpuState.cc, 2)
			}
		default: // Negative
			if (ur & HDMASK) != HDMASK {
				if cpuState.cc != 3 {
					t.Errorf("S rand not correct got: %x wanted: %x", cpuState.cc, 3)
				}
				if cpuState.regs[1] != diff {
					t.Errorf("S rand over register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], diff)
				}
				continue
			}
			if cpuState.cc != 1 {
				t.Errorf("S rand not correct got: %x wanted: %x", cpuState.cc, 1)
			}
		}
		if cpuState.regs[1] != diff {
			t.Errorf("S rand register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], diff)
		}
	}
}

// Test AH instruction.
func TestCycleAH(t *testing.T) {
	setup()
	// Test add half positive
	memory.SetMemory(0x400, 0x4a300200) // AH 3,200(0,0)
	memory.SetMemory(0x200, 0x1234eeee)
	cpuState.regs[3] = 0x12345678
	cpuState.testInst(0)

	v := cpuState.regs[3]
	r := uint32(0x12345678 + 0x1234)
	if v != r {
		t.Errorf("AH Register changed got: %08x wanted: %08x", v, r)
	}
	if cpuState.cc != 2 {
		t.Errorf("AH CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Test add half sign extend
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x4a300200) // AH 3,200(0,0)
	memory.SetMemory(0x200, 0xfffe9999) // -2
	cpuState.regs[3] = 0x12345678
	cpuState.testInst(0)

	v = cpuState.regs[3]
	r = uint32(0x12345676)
	if v != r {
		t.Errorf("AH Register changed got: %08x wanted: %08x", v, r)
	}
	if cpuState.cc != 2 {
		t.Errorf("AH CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Test Subtract half.
func TestCycleSH(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x4b156200) // SH 1,200(5,6)
	memory.SetMemory(0x500, 0x1230ffff)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	cpuState.testInst(0)
	s := uint32(0x12345678) - uint32(0x1230)
	if cpuState.regs[1] != s {
		t.Errorf("SH register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], s)
	}
	if cpuState.cc != 2 {
		t.Errorf("SH CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Test Subtract logical.
func TestCycleSL(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1f120000) // SLR 1,2
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x12345678
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("SL register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.cc != 2 {
		t.Errorf("SL CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x5f156200) // SL 1,200(5,6)
	cpuState.regs[1] = 0xffffffff
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0x11111111)
	cpuState.testInst(0)
	if cpuState.regs[1] != 0xeeeeeeee {
		t.Errorf("SL register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0xeeeeeeee)
	}
	if cpuState.cc != 3 {
		t.Errorf("SL CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x5f156200) // SL 1,200(5,6)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0x23456789)
	cpuState.testInst(0)
	if cpuState.regs[1] != 0xeeeeeeef {
		t.Errorf("SL register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0xeeeeeeef)
	}
	if cpuState.cc != 1 {
		t.Errorf("SL CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
	// // Test add logical with random values
	rnum := rand.New(rand.NewSource(44))

	for range testCycles {
		cpuState.cc = 3
		n1 := uint32(rnum.Int31())
		n2 := uint32(rnum.Int31())
		n3 := ^n2                          // Negate n2
		sum := uint64(n1) + uint64(n3) + 1 // Sum
		cpuState.regs[1] = n1
		memory.SetMemory(0x100, n2)
		memory.SetMemory(0x400, 0x5f100100) // SL 1,100(0,0)
		// Compute resulting cc flags
		cc := uint8(0)
		if (sum & 0x100000000) != 0 {
			cc = 2
			// r &= 0x0ffffffff
		}
		if sum != 0 {
			cc++
		}

		cpuState.testInst(0)

		if cpuState.cc != cc {
			t.Errorf("SL rand not correct got: %x wanted: %x", cpuState.cc, cc)
		}
		if cpuState.regs[1] != uint32(sum&LMASKL) {
			t.Errorf("SL rand register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], sum)
		}
	}
}

func TestCycleC(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x19120000) // CR 1,2
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x12345678
	cpuState.testInst(0)
	if cpuState.regs[1] != 0x12345678 {
		t.Errorf("CR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x12345678)
	}
	if cpuState.cc != 0 {
		t.Errorf("CR CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x19120000) // CR 1,2
	cpuState.regs[1] = 0xfffffffe       // -2
	cpuState.regs[2] = 0xfffffffd       // -3
	cpuState.testInst(0)
	if cpuState.regs[1] != 0xfffffffe {
		t.Errorf("CR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0xfffffffe)
	}
	if cpuState.cc != 2 {
		t.Errorf("CR CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x19120000) // CR 1,2
	cpuState.regs[1] = 2
	cpuState.regs[2] = 3
	cpuState.testInst(0)
	if cpuState.regs[1] != 2 {
		t.Errorf("CR register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 2)
	}
	if cpuState.cc != 1 {
		t.Errorf("CR CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x59156200) // C 1,200(5,6)
	memory.SetMemory(0x500, 0x12345678)
	cpuState.regs[1] = 0xf0000000
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	cpuState.testInst(0)

	if cpuState.regs[1] != 0xf0000000 {
		t.Errorf("C register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0xf0000000)
	}
	if cpuState.cc != 1 {
		t.Errorf("C CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Test CL instruction.
func TestCycleCL(t *testing.T) {
	setup()
	// Test compare half equal
	memory.SetMemory(0x400, 0x55123400) // CL 1,400(2,3)
	memory.SetMemory(0x404, 0)
	memory.SetMemory(0x900, 0x12345678)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x200
	cpuState.regs[3] = 0x300
	cpuState.testInst(0)

	v := cpuState.regs[1]
	if v != 0x12345678 {
		t.Errorf("CL Register changed got: %08x wanted: %08x", v, 0x12345678)
	}
	if cpuState.cc != 0 {
		t.Errorf("CL CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Compare logical register
	cpuState.cc = 3
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x12345678
	memory.SetMemory(0x400, 0x15120000) // CLR 1,2
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CL CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x12345679
	memory.SetMemory(0x400, 0x15120000) // CLR 1,2
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CL CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x12345679
	cpuState.regs[2] = 0x12345678
	memory.SetMemory(0x400, 0x15120000) // CLR 1,2
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("CL CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x7fffffff
	cpuState.regs[2] = 0x8fffffff
	memory.SetMemory(0x400, 0x15120000) // CLR 1,2
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CL CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Compare logical
	cpuState.cc = 3
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x100
	cpuState.regs[3] = 0x100
	memory.SetMemory(0x300, 0x12345678)
	memory.SetMemory(0x400, 0x55123100) // CL 1,100(2,3)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CL CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

}

// Test CH instruction.
func TestCycleCH(t *testing.T) {
	setup()
	// Test compare half equal
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0x5678abcd)
	cpuState.regs[3] = 0x00005678
	cpuState.testInst(0)

	v := cpuState.regs[3]
	if v != 0x00005678 {
		t.Errorf("CH Register changed got: %08x wanted: %08x", v, 0x00005678)
	}
	if cpuState.cc != 0 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Test compare half with sign extension
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0x9678abcd)
	cpuState.regs[3] = 0xffff9678
	cpuState.testInst(0)

	if cpuState.cc != 0 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Compare half word high
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0x1234abcd)
	cpuState.regs[3] = 0x00001235
	cpuState.testInst(0)

	if cpuState.cc != 2 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Compare half word sign extended
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0x8234abcd)
	cpuState.regs[3] = 0x00001235
	cpuState.testInst(0)

	if cpuState.cc != 2 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Compare half word low
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0x1234abcd)
	cpuState.regs[3] = 0x80001235
	cpuState.testInst(0)

	if cpuState.cc != 1 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Compare half lower extended
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0xfffd0000)
	cpuState.regs[3] = 0xfffffffc
	cpuState.testInst(0)

	if cpuState.cc != 1 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Test Multiply instruction.
func TestCycleM(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpuState.regs[2] = 0
	cpuState.regs[3] = 28
	cpuState.regs[4] = 19
	cpuState.testInst(0)
	if cpuState.regs[2] != 0 {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0)
	}
	if cpuState.regs[3] != (28 * 19) {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 28*16)
	}

	if cpuState.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpuState.regs[2] = 0
	cpuState.regs[3] = 0x12345678
	cpuState.regs[4] = 0x34567890
	cpuState.testInst(0)
	if cpuState.regs[2] != 0x3b8c7b8 {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x3b8c7b8)
	}
	if cpuState.regs[3] != 0x3248e380 {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x3248e380)
	}

	if cpuState.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpuState.regs[2] = 0
	cpuState.regs[3] = 0x7fffffff
	cpuState.regs[4] = 0x7fffffff
	cpuState.testInst(0)
	if cpuState.regs[2] != 0x3fffffff {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x3fffffff)
	}
	if cpuState.regs[3] != 0x00000001 {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000001)
	}

	if cpuState.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpuState.regs[2] = 0
	cpuState.regs[3] = 0xfffffffc // -4
	cpuState.regs[4] = 0xfffffffb // -5
	cpuState.testInst(0)
	if cpuState.regs[2] != 0 {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0)
	}
	if cpuState.regs[3] != 20 {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 20)
	}

	if cpuState.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpuState.regs[2] = 0
	cpuState.regs[3] = 0xfffffffc // -4
	cpuState.regs[4] = 0x0000000a // 10
	cpuState.testInst(0)
	if cpuState.regs[2] != 0xffffffff {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0xffffffff)
	}
	if cpuState.regs[3] != 0xffffffd8 {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0xffffffd8)
	}

	if cpuState.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x5c256200) // M 1,200(5,6)
	memory.SetMemory(0x500, 0x34567890)
	cpuState.regs[2] = 0
	cpuState.regs[3] = 0x12345678
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	cpuState.testInst(0)
	if cpuState.regs[2] != 0x03b8c7b8 {
		t.Errorf("M register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x03b8c7b8)
	}
	if cpuState.regs[3] != 0x3248e380 {
		t.Errorf("M register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x3248e380)
	}
	if cpuState.cc != 3 {
		t.Errorf("M CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Test multiply with random values
	rnum := rand.New(rand.NewSource(1))
	for range testCycles {
		cpuState.cc = 3
		n1 := rnum.Int31()
		n2 := rand.Int31()
		r := int64(n1) * int64(n2)
		h := uint32((uint64(r) >> 32) & uint64(FMASK))
		l := uint32(uint64(r) & uint64(FMASK))
		cpuState.regs[2] = 0
		cpuState.regs[3] = uint32(n1)
		cpuState.regs[4] = uint32(n2)
		memory.SetMemory(0x400, 0x1c240000) // MR 2,4
		cpuState.testInst(0)
		if cpuState.regs[2] != h {
			t.Errorf("MR rand register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], h)
		}
		if cpuState.regs[3] != l {
			t.Errorf("MR rand register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], l)
		}
		if cpuState.cc != 3 {
			t.Errorf("MR rand not correct got: %x wanted: %x", cpuState.cc, 3)
		}
	}
}

func TestCycleMH(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x4c356202) // MH 3,202(5,6)
	memory.SetMemory(0x500, 0x00000003)
	cpuState.regs[2] = 0
	cpuState.regs[3] = 4
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	cpuState.testInst(0)
	if cpuState.regs[2] != 0 {
		t.Errorf("MHregister 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0)
	}
	if cpuState.regs[3] != 12 {
		t.Errorf("MH register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 12)
	}

	if cpuState.cc != 3 {
		t.Errorf("MH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x4c250200) // MH 2,200(5)
	memory.SetMemory(0x500, 0xffd91111) // -39
	cpuState.regs[2] = 0x00000015       // 21
	cpuState.regs[3] = 0x00000005
	cpuState.regs[5] = 0x00000300
	cpuState.regs[6] = 0x00000200
	cpuState.testInst(0)
	if cpuState.regs[2] != 0xfffffccd {
		t.Errorf("MHregister 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0xfffffccd)
	}
	if cpuState.regs[3] != 0x00000005 {
		t.Errorf("MH register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00000005)
	}

	if cpuState.cc != 3 {
		t.Errorf("MH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

func TestCycleD(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1d240000) // DR 2,4
	cpuState.regs[2] = 0x1
	cpuState.regs[3] = 0x12345678
	cpuState.regs[4] = 0x00000234
	// divide R2/R3 by R4
	cpuState.testInst(0)
	if cpuState.regs[2] != (0x112345678 % 0x234) {
		t.Errorf("DR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x112345678%0x234)
	}
	if cpuState.regs[3] != (0x112345678 / 0x234) {
		t.Errorf("DR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x112345678/0x234)
	}

	if cpuState.cc != 3 {
		t.Errorf("DR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1d240000) // DR 2,4
	cpuState.regs[2] = 0x1
	cpuState.regs[3] = 0x12345678
	cpuState.regs[4] = 0xfffffdcc
	// divide R2/R3 by R4
	cpuState.testInst(0)
	if cpuState.regs[2] != (0x112345678 % 0x234) {
		t.Errorf("DR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x112345678%0x234)
	}
	if cpuState.regs[3] != (((0x112345678 / 0x234) ^ FMASK) + 1) {
		t.Errorf("DR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], ((0x112345678/0x234)^FMASK)+1)
	}
	if cpuState.cc != 3 {
		t.Errorf("DR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Divide big value
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1d240000) // DR 2,4
	cpuState.regs[2] = 0x00112233
	cpuState.regs[3] = 0x44556677
	cpuState.regs[4] = 0x12345678 // 0x1122334455667788 / 0x12345678
	// divide R2/R3 by R4
	cpuState.testInst(0)
	if cpuState.regs[2] != (0x11b3d5f7) {
		t.Errorf("DR register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x11b3d5f7)
	}
	if cpuState.regs[3] != 0x00f0f0f0 {
		t.Errorf("DR register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x00f0f0f0)
	}

	if cpuState.cc != 3 {
		t.Errorf("DR CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x5d256200) // D 2,200(5,6)
	memory.SetMemory(0x404, 0x00000000)
	memory.SetMemory(0x500, 0x73456789)
	cpuState.regs[2] = 0x12345678
	cpuState.regs[3] = 0x9abcdef0
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	cpuState.testInst(0)
	if cpuState.regs[2] != 0x50c0186a {
		t.Errorf("D register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x50c0186a)
	}
	if cpuState.regs[3] != 0x286dead6 {
		t.Errorf("D register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x286dead6)
	}
	if cpuState.cc != 3 {
		t.Errorf("D CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Divide overflow
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x5d256200) // D 2,200(5,6)
	memory.SetMemory(0x404, 0x00000000)
	memory.SetMemory(0x800, 0x00000000)
	memory.SetMemory(0x500, 0x23456789)
	cpuState.regs[2] = 0x12345678
	cpuState.regs[3] = 0x9abcdef0
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200

	cpuState.testInst(0x8)
	if cpuState.regs[2] != 0x12345678 {
		t.Errorf("D register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], 0x12345678)
	}
	if cpuState.regs[3] != 0x9abcdef0 {
		t.Errorf("D register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], 0x9abcdef0)
	}
	if !trapFlag {
		t.Errorf("D over did not trap")
	}

	// Test divide with random values
	rnum := rand.New(rand.NewSource(124))
	for range testCycles {
		cpuState.cc = 3
		dividend := rnum.Int63() / 1000
		divisor := rnum.Int31()
		q := dividend / int64(divisor)
		r := dividend % int64(divisor)

		cpuState.regs[2] = uint32(dividend >> 32)
		cpuState.regs[3] = uint32(uint64(dividend) & LMASKL)
		memory.SetMemory(0x100, uint32(divisor))
		memory.SetMemory(0x400, 0x5d200100) // D 2,100(0,0)
		memory.SetMemory(0x404, 0x00000000)
		memory.SetMemory(0x800, 0x00000000)
		cpuState.testInst(0)

		if divisor < 0 {
			r = -r
		}

		// Check if we should overflow.
		if (q & 0x7fffffff) != q {
			if !trapFlag {
				t.Errorf("D rand over did not trap")
			}
		} else {
			if trapFlag {
				t.Errorf("D rand no over trap")
			}
			if cpuState.regs[2] != uint32(r) {
				t.Logf("D %016x / %08x = %08x (%08x) %t", dividend, divisor, q, r, trapFlag)
				t.Errorf("D rand register 2 was incorrect got: %08x wanted: %08x", cpuState.regs[2], uint32(r))
			}
			if cpuState.regs[3] != uint32(q) {
				t.Errorf("D rand register 3 was incorrect got: %08x wanted: %08x", cpuState.regs[3], uint32(q))
			}
			if cpuState.cc != 3 {
				t.Errorf("D rand CC not correct got: %x wanted: %x", cpuState.cc, 3)
			}
		}
	}
}

// Test Store Word.
func TestCycleST(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x50123400) // ST 1,400(2,3)
	memory.SetMemory(0x600, 0xffffffff)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x100
	cpuState.regs[3] = 0x100
	// Store Half
	cpuState.testInst(0)

	v := memory.GetMemory(0x600)
	if v != 0x12345678 {
		t.Errorf("ST Memory not correct got: %08x wanted: %08x", v, 0x12345678)
	}
	if cpuState.cc != 3 {
		t.Errorf("ST CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Check store off align
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x50123400) // ST 1,400(2,3)
	memory.SetMemory(0x600, 0xffffffff)
	memory.SetMemory(0x604, 0xffffffff)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x101
	cpuState.regs[3] = 0x100
	// Store Half
	cpuState.testInst(0)

	v = memory.GetMemory(0x600)
	if v != 0xff123456 {
		t.Errorf("ST Memory not correct got: %08x wanted: %08x", v, 0xff123456)
	}
	v = memory.GetMemory(0x604)
	if v != 0x78ffffff {
		t.Errorf("ST Memory not correct got: %08x wanted: %08x", v, 0x78ffffff)
	}
	if cpuState.cc != 3 {
		t.Errorf("ST CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x50123400) // ST 1,400(2,3)
	memory.SetMemory(0x600, 0xffffffff)
	memory.SetMemory(0x604, 0xffffffff)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x102
	cpuState.regs[3] = 0x100
	// Store Half
	cpuState.testInst(0)

	v = memory.GetMemory(0x600)
	if v != 0xffff1234 {
		t.Errorf("ST Memory not correct got: %08x wanted: %08x", v, 0xffff1234)
	}
	v = memory.GetMemory(0x604)
	if v != 0x5678ffff {
		t.Errorf("ST Memory not correct got: %08x wanted: %08x", v, 0x5678ffff)
	}
	if cpuState.cc != 3 {
		t.Errorf("ST CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x50123400) // ST 1,400(2,3)
	memory.SetMemory(0x600, 0xffffffff)
	memory.SetMemory(0x604, 0xffffffff)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x103
	cpuState.regs[3] = 0x100
	// Store Half
	cpuState.testInst(0)

	v = memory.GetMemory(0x600)
	if v != 0xffffff12 {
		t.Errorf("ST Memory not correct got: %08x wanted: %08x", v, 0xffffff12)
	}
	v = memory.GetMemory(0x604)
	if v != 0x345678ff {
		t.Errorf("ST Memory not correct got: %08x wanted: %08x", v, 0x345678ff)
	}
	if cpuState.cc != 3 {
		t.Errorf("ST CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test Store Half Word.
func TestCycleSTH(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x40345ffe) // STH 2,ffe(4,5)
	memory.SetMemory(0x1000, 0x12345678)
	cpuState.regs[3] = 0xaabbccdd
	cpuState.regs[4] = 1
	cpuState.regs[5] = 1
	// Store Half
	cpuState.testInst(0)

	v := memory.GetMemory(0x1000)
	if v != 0xccdd5678 {
		t.Errorf("STH Memory not correct got: %08x wanted: %08x", v, 0xccdd5678)
	}
	if cpuState.cc != 3 {
		t.Errorf("STH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x40345ffe) // STH 2,ffe(4,5)
	memory.SetMemory(0x1000, 0x12345678)
	cpuState.regs[3] = 0xaabbccdd
	cpuState.regs[4] = 1
	cpuState.regs[5] = 3
	// Store Half
	cpuState.testInst(0)

	v = memory.GetMemory(0x1000)
	if v != 0x1234ccdd {
		t.Errorf("STH Memory not correct got: %08x wanted: %08x", v, 0x1234ccdd)
	}
	if cpuState.cc != 3 {
		t.Errorf("STH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x40345ffe) // STH 2,ffe(4,5)
	memory.SetMemory(0x1000, 0x12345678)
	cpuState.regs[3] = 0xaabbccdd
	cpuState.regs[4] = 1
	cpuState.regs[5] = 2
	// Store Half
	cpuState.testInst(0)

	v = memory.GetMemory(0x1000)
	if v != 0x12ccdd78 {
		t.Errorf("STH Memory not correct got: %08x wanted: %08x", v, 0x12ccdd78)
	}
	if cpuState.cc != 3 {
		t.Errorf("STH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x40345ffe) // STH 2,ffe(4,5)
	memory.SetMemory(0x1000, 0x12345678)
	memory.SetMemory(0x1004, 0x9abcdef0)
	cpuState.regs[3] = 0xaabbccdd
	cpuState.regs[4] = 1
	cpuState.regs[5] = 4
	// Store Half
	cpuState.testInst(0)

	v = memory.GetMemory(0x1000)
	if v != 0x123456cc {
		t.Errorf("STH Memory not correct got: %08x wanted: %08x", v, 0x123456cc)
	}
	v = memory.GetMemory(0x1004)
	if v != 0xddbcdef0 {
		t.Errorf("STH Memory not correct got: %08x wanted: %08x", v, 0xddbcdef0)
	}
	if cpuState.cc != 3 {
		t.Errorf("STH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test Load Half Word.
func TestCycleLH(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x48345986) // LH 3,986(4,5)
	memory.SetMemory(0x1b84, 0x87654321)
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1000
	cpuState.regs[5] = 0x200
	// Store Half
	cpuState.testInst(0)

	v := cpuState.regs[3]
	if v != 0x00004321 {
		t.Errorf("LH Memory not correct got: %08x wanted: %08x", v, 0x00004321)
	}
	if cpuState.cc != 3 {
		t.Errorf("LH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x48345984) // LH 3,984(4,5)
	memory.SetMemory(0x1b84, 0x17654321)
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1000
	cpuState.regs[5] = 0x200
	// Store Half
	cpuState.testInst(0)

	v = cpuState.regs[3]
	if v != 0x00001765 {
		t.Errorf("LH Memory not correct got: %08x wanted: %08x", v, 0x00001765)
	}
	if cpuState.cc != 3 {
		t.Errorf("LH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x48345984) /// LH 3,984(4,5)
	memory.SetMemory(0x1b84, 0x87654321)
	cpuState.regs[3] = 0xaabbccdd
	cpuState.regs[4] = 0x1000
	cpuState.regs[5] = 0x200
	// Store Half
	cpuState.testInst(0)

	v = cpuState.regs[3]
	if v != 0xffff8765 {
		t.Errorf("LH Memory not correct got: %08x wanted: %08x", v, 0xffff8765)
	}
	if cpuState.cc != 3 {
		t.Errorf("LH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x48345984) // LH 3,984(4,5)
	memory.SetMemory(0x1b84, 0x87654321)
	memory.SetMemory(0x1b88, 0xabcdef00)
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1000
	cpuState.regs[5] = 0x201
	// Store Half
	cpuState.testInst(0)

	v = cpuState.regs[3]
	if v != 0x00006543 {
		t.Errorf("LH Memory not correct got: %08x wanted: %08x", v, 0x00006543)
	}
	if cpuState.cc != 3 {
		t.Errorf("LH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x48345984) // LH 3,984(4,5)
	memory.SetMemory(0x1b84, 0x87654321)
	memory.SetMemory(0x1b88, 0xabcdef00)
	cpuState.regs[3] = 0xffffffff
	cpuState.regs[4] = 0x1000
	cpuState.regs[5] = 0x203
	// Store Half
	cpuState.testInst(0)

	v = cpuState.regs[3]
	if v != 0x000021ab {
		t.Errorf("LH Memory not correct got: %08x wanted: %08x", v, 0x000021ab)
	}
	if cpuState.cc != 3 {
		t.Errorf("LH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test LA.
func TestCycleLA(t *testing.T) {
	setup()
	// From Princ Ops p147
	memory.SetMemory(0x400, 0x41100800) // LA 1,800
	cpuState.regs[1] = 0xffffffff
	cpuState.testInst(0)
	v := cpuState.regs[1]
	if v != 2048 {
		t.Errorf("LA Register not correct got: %08x wanted: %08x", v, 2048)
	}
	if cpuState.cc != 3 {
		t.Errorf("LA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	// From Princ Ops p147
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x4150500a) // LA 5, 10(5)
	cpuState.regs[5] = 0x00123456
	cpuState.testInst(0)

	v = cpuState.regs[5]
	if v != 0x00123460 {
		t.Errorf("LA Register not correct got: %08x wanted: %08x", v, 0x00123460)
	}
	if cpuState.cc != 3 {
		t.Errorf("LA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x4156500a) // LA 5, 10(6,5)
	cpuState.regs[5] = 0x00123456
	cpuState.regs[6] = 0x00000010
	cpuState.testInst(0)

	v = cpuState.regs[5]
	if v != 0x00123470 {
		t.Errorf("LA Register not correct got: %08x wanted: %08x", v, 0x00123470)
	}
	if cpuState.cc != 3 {
		t.Errorf("LA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x4155000a) // LA 5, 10(0,5)
	cpuState.regs[5] = 0x00123456
	cpuState.regs[6] = 0x00000010
	cpuState.testInst(0)

	v = cpuState.regs[5]
	if v != 0x00123460 {
		t.Errorf("LA Memory not correct got: %08x wanted: %08x", v, 0x00123460)
	}
	if cpuState.cc != 3 {
		t.Errorf("LA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test STC.
func TestCycleSTC(t *testing.T) {
	setup()
	for i := range 4 { // Test all 4 offsets.
		cpuState.cc = 3
		cpuState.regs[5] = 0xffffff12
		cpuState.regs[1] = uint32(i)
		memory.SetMemory(0x400, 0x42501100) // STC 5,100(0,1)
		memory.SetMemory(0x100, 0xaabbccdd)
		cpuState.testInst(0)
		if cpuState.cc != 3 {
			t.Errorf("STC CC not correct got: %x wanted: %x", cpuState.cc, 3)
		}
		v := memory.GetMemory(0x100)
		shift := (3 - i) * 8
		desired := (uint32(0xaabbccdd) & ^(uint32(0xff) << shift)) | (uint32(0x12) << shift)

		if v != desired {
			t.Errorf("STC Memory not correct got: %08x wanted: %08x", v, desired)
		}
	}
}

// Test IC.
func TestCycleSIC(t *testing.T) {
	setup()
	for i := range 4 { // Test all 4 offsets.
		cpuState.cc = 3
		cpuState.regs[5] = 0xaabbccdd
		cpuState.regs[1] = uint32(i)
		memory.SetMemory(0x400, 0x43501100) // IC 5,100(0,1)
		memory.SetMemory(0x100, 0x00112233)
		cpuState.testInst(0)
		if cpuState.cc != 3 {
			t.Errorf("IC CC not correct got: %x wanted: %x", cpuState.cc, 3)
		}
		v := memory.GetMemory(0x100)
		shift := (3 - i) * 8
		desired := (uint32(0x00112233) >> shift & 0xff) | 0x00112233

		if v != desired {
			t.Errorf("IC Memory not correct got: %08x wanted: %08x", v, desired)
		}
	}
}

// Test EX.
func TestCycleEX(t *testing.T) {
	setup()
	memory.SetMemory(0x100, 0x1a000000) // Target instruction AR 0,0
	cpuState.regs[1] = 0x00000045
	cpuState.regs[4] = 0x100
	cpuState.regs[5] = 0x200
	memory.SetMemory(0x400, 0x44100100) // EX 1,100(0,0)
	memory.SetMemory(0x404, 0x00000000) // Prevent fetch of next instruction
	cpuState.testInst(0)
	v := cpuState.regs[4]
	if v != 0x300 {
		t.Errorf("EX AR Register not correct got: %08x wanted: %08x", v, 0x300)
	}

	memory.SetMemory(0x045, 0x1a000000) // Target instruction AR 0,0
	memory.SetMemory(0x100, 0x44100000) // Target instruction EX 1,100(0,0)
	cpuState.regs[1] = 0x00000045
	cpuState.regs[4] = 0x100
	cpuState.regs[5] = 0x200
	memory.SetMemory(0x400, 0x44100100) // EX 1,100(0,0)
	memory.SetMemory(0x404, 0x00000000) // Prevent fetch of next instruction
	cpuState.testInst(0)

	if !trapFlag {
		t.Errorf("EX of EX did not trap")
	}
}

// Test BAL.
func TestCycleBAL(t *testing.T) {
	setup()
	cpuState.regs[3] = 0x12000000
	cpuState.regs[4] = 0x00005600
	memory.SetMemory(0x400, 0x45134078) // BAL 1,78(3,4)
	cpuState.ilc = 0
	cpuState.cc = 3
	cpuState.testInst(0xa)
	v := cpuState.regs[1]
	if v != 0xba000404 {
		t.Errorf("BAL Register 1 not correct got: %08x wanted: %08x", v, 0xba000404)
	}
	if cpuState.PC != 0x00005678 {
		t.Errorf("BAL PC not correct got: %08x wanted: %08x", cpuState.PC, 0x00005678)
	}
}

// Test BALR.
func TestCycleBALR(t *testing.T) {
	setup()
	cpuState.regs[1] = 0
	cpuState.regs[2] = 0x12005678
	memory.SetMemory(0x400, 0x05120000) // BALR 1,2
	cpuState.ilc = 0
	cpuState.cc = 3
	cpuState.testInst(0xa)
	v := cpuState.regs[1]
	if v != 0x7a000402 {
		t.Errorf("BAL Register 1 not correct got: %08x wanted: %08x", v, 0x7a000402)
	}
	if cpuState.PC != 0x00005678 {
		t.Errorf("BAL PC not correct got: %08x wanted: %08x", cpuState.PC, 0x00005678)
	}

	// Branch and link with no branch
	setup()
	cpuState.regs[1] = 0
	cpuState.regs[2] = 0x12005678
	memory.SetMemory(0x400, 0x05100000) // BALR 1,9
	cpuState.ilc = 0
	cpuState.cc = 3
	cpuState.testInst(0xa)
	v = cpuState.regs[1]
	if v != 0x7a000402 {
		t.Errorf("BAL Register 1 not correct got: %08x wanted: %08x", v, 0x7a000402)
	}
	if cpuState.PC != 0x402 {
		t.Errorf("BAL PC not correct got: %08x wanted: %08x", cpuState.PC, 0x402)
	}
}

// Test BCT.
func TestCycleBCT(t *testing.T) {
	setup()
	cpuState.regs[1] = 3          // Count
	cpuState.regs[2] = 0x00005678 // branch destination
	cpuState.regs[3] = 0x00000010
	memory.SetMemory(0x400, 0x46123100) // BCT 1,100(2,3)
	cpuState.cc = 3
	cpuState.testInst(0)
	v := cpuState.regs[1]
	if v != 2 {
		t.Errorf("BCT Register 1 not correct got: %08x wanted: %08x", v, 2)
	}
	if cpuState.PC != 0x00005788 {
		t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x00005788)
	}
	cpuState.regs[1] = 1          // Count
	cpuState.regs[2] = 0x00005678 // branch destination
	cpuState.regs[3] = 0x00000010
	memory.SetMemory(0x400, 0x46123100) // BCT 1,100(2,3)
	cpuState.cc = 3
	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0 {
		t.Errorf("BCT Register 1 not correct got: %08x wanted: %08x", v, 0)
	}
	if cpuState.PC != 0x404 {
		t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x404)
	}
}

// Test BCTR.
func TestCycleBCTR(t *testing.T) {
	setup()
	cpuState.regs[1] = 3          // Count
	cpuState.regs[2] = 0x00005678 // branch destination
	cpuState.regs[3] = 0x00000010
	memory.SetMemory(0x400, 0x06120000) // BCTR 1,2
	cpuState.cc = 3
	cpuState.testInst(0)
	v := cpuState.regs[1]
	if v != 2 {
		t.Errorf("BCT Register 1 not correct got: %08x wanted: %08x", v, 2)
	}
	if cpuState.PC != 0x00005678 {
		t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x00005578)
	}

	cpuState.regs[1] = 0          // Count
	cpuState.regs[2] = 0x00005678 // branch destination
	cpuState.regs[3] = 0x00000010
	memory.SetMemory(0x400, 0x06120000) // BCTR 1,2
	cpuState.cc = 3
	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0xffffffff {
		t.Errorf("BCT Register 1 not correct got: %08x wanted: %08x", v, 0xffffffff)
	}
	if cpuState.PC != 0x00005678 {
		t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x00005678)
	}

	cpuState.regs[1] = 1          // Count
	cpuState.regs[2] = 0x00005678 // branch destination
	cpuState.regs[3] = 0x00000010
	memory.SetMemory(0x400, 0x06120000) // BCTR 1,2
	cpuState.cc = 3
	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0 {
		t.Errorf("BCT Register 1 not correct got: %08x wanted: %08x", v, 0)
	}
	if cpuState.PC != 0x402 {
		t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x402)
	}
}

// Test BC on all conditions with all values of CC.
func TestCycleBC(t *testing.T) {
	setup()
	memory.SetMemory(0x100, 0)
	for i := range 16 {
		for j := range 4 {
			op := uint32(0x47000100) | (uint32(i) << 20) // BC i,100
			cpuState.cc = uint8(j)
			memory.SetMemory(0x400, op)
			cpuState.testInst(0)
			if ((i&8) != 0 && cpuState.cc == 0) ||
				((i&4) != 0 && cpuState.cc == 1) ||
				((i&2) != 0 && cpuState.cc == 2) ||
				((i&1) != 0 && cpuState.cc == 3) {
				// Taken
				if cpuState.PC != 0x100 {
					t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x100)
				}
			} else {
				if cpuState.PC != 0x404 {
					t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x404)
				}
			}
		}
	}
}

// Test BCR on all conditions with all values of CC.
func TestCycleBCR(t *testing.T) {
	setup()
	cpuState.regs[1] = 0x12005678 // Branch destination
	for i := range 16 {
		for j := range 4 {
			op := uint32(0x07010000) | (uint32(i) << 20) // BCR i,1
			cpuState.cc = uint8(j)
			memory.SetMemory(0x400, op)
			cpuState.testInst(0)
			if ((i&8) != 0 && cpuState.cc == 0) ||
				((i&4) != 0 && cpuState.cc == 1) ||
				((i&2) != 0 && cpuState.cc == 2) ||
				((i&1) != 0 && cpuState.cc == 3) {
				// Taken
				if cpuState.PC != 0x00005678 {
					t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x00005678)
				}
			} else {
				if cpuState.PC != 0x402 {
					t.Errorf("BCT PC not correct got: %08x wanted: %08x", cpuState.PC, 0x402)
				}
			}
		}
	}
}

// Test BXH.
func TestCycleBXH(t *testing.T) {
	setup()

	// Add increment to first operand, compare with odd register after R3
	memory.SetMemory(0x400, 0x86142200) // BXH 1, 4, 200(2)
	memory.SetMemory(0x1200, 0)         // Clear target of branch
	cpuState.regs[1] = 0x12345678       // Value
	cpuState.regs[2] = 0x1000           // Branch target
	cpuState.regs[4] = 1                // Increment
	cpuState.regs[5] = 0x12345678       // Compare value

	cpuState.testInst(0)
	v := cpuState.regs[1]
	if v != 0x12345679 {
		t.Errorf("BXH Register not correct got: %08x wanted: %08x", v, 0x12345679)
	}
	if cpuState.cc != 3 {
		t.Errorf("BXH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	if cpuState.PC != 0x1200 {
		t.Errorf("BXH PC not correct got: %08x wanted: %08x", cpuState.PC, 0x1200)
	}

	// Add increment to first operand, compare with odd register after R3
	memory.SetMemory(0x400, 0x86142200) // BXH 1, 4, 200(2)
	memory.SetMemory(0x1200, 0)         // Clear target of branch
	cpuState.regs[1] = 0x12345678       // Value
	cpuState.regs[2] = 0x1000           // Branch target
	cpuState.regs[4] = 0xffffffff       // Increment -1
	cpuState.regs[5] = 0x12345678       // Compare value

	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0x12345677 {
		t.Errorf("BXH Register not correct got: %08x wanted: %08x", v, 0x12345677)
	}
	if cpuState.cc != 3 {
		t.Errorf("BXH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	if cpuState.PC != 0x404 {
		t.Errorf("BXH PC not correct got: %08x wanted: %08x", cpuState.PC, 0x404)
	}

	// Add increment to first operand, compare with odd register after R3
	memory.SetMemory(0x400, 0x86132200) // BXH 1, 3, 200(2)
	memory.SetMemory(0x1200, 0)         // Clear target of branch
	cpuState.regs[1] = 1                // Value
	cpuState.regs[2] = 0x1000           // Branch target
	cpuState.regs[3] = 0x12345678       // Incrment and Compare value
	cpuState.regs[4] = 0xffffffff       // Increment -1
	cpuState.regs[5] = 0x12345678       // Compare value

	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0x12345679 {
		t.Errorf("BXH Register not correct got: %08x wanted: %08x", v, 0x12345679)
	}
	if cpuState.cc != 3 {
		t.Errorf("BXH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	if cpuState.PC != 0x1200 {
		t.Errorf("BXH PC not correct got: %08x wanted: %08x", cpuState.PC, 0x1200)
	}

	// Add increment to first operand, compare with odd register after R3
	memory.SetMemory(0x400, 0x86132200) // BXH 1, 3, 200(2)
	memory.SetMemory(0x1200, 0)         // Clear target of branch
	cpuState.regs[1] = 0xffffffff       // Value
	cpuState.regs[2] = 0x1000           // Branch target
	cpuState.regs[3] = 0x12345678       // Incrment and Compare value
	cpuState.regs[4] = 0xffffffff       // Increment -1
	cpuState.regs[5] = 0x12345678       // Compare value

	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0x12345677 {
		t.Errorf("BXH Register not correct got: %08x wanted: %08x", v, 0x12345677)
	}
	if cpuState.cc != 3 {
		t.Errorf("BXH CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	if cpuState.PC != 0x404 {
		t.Errorf("BXH PC not correct got: %08x wanted: %08x", cpuState.PC, 0x404)
	}
}

// Test BXLE.
func TestCycleBXLE(t *testing.T) {
	setup()

	// Add increment to first operand, compare with odd register after R3
	memory.SetMemory(0x400, 0x87142200) // BXLE 1, 4, 200(2)
	memory.SetMemory(0x1200, 0)         // Clear target of branch
	cpuState.regs[1] = 0x12345678       // Value
	cpuState.regs[2] = 0x1000           // Branch target
	cpuState.regs[4] = 1                // Increment
	cpuState.regs[5] = 0x12345678       // Compare value

	cpuState.testInst(0)
	v := cpuState.regs[1]
	if v != 0x12345679 {
		t.Errorf("BXLE Register not correct got: %08x wanted: %08x", v, 0x12345679)
	}
	if cpuState.cc != 3 {
		t.Errorf("BXLE CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	if cpuState.PC != 0x404 {
		t.Errorf("BXLE PC not correct got: %08x wanted: %08x", cpuState.PC, 0x40)
	}

	// Add increment to first operand, compare with odd register after R3
	memory.SetMemory(0x400, 0x87142200) // BXLE 1, 4, 200(2)
	memory.SetMemory(0x1200, 0)         // Clear target of branch
	cpuState.regs[1] = 0x12345678       // Value
	cpuState.regs[2] = 0x1000           // Branch target
	cpuState.regs[4] = 0xffffffff       // Increment -1
	cpuState.regs[5] = 0x12345678       // Compare value

	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0x12345677 {
		t.Errorf("BXLE Register not correct got: %08x wanted: %08x", v, 0x12345677)
	}
	if cpuState.cc != 3 {
		t.Errorf("BXLE CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	if cpuState.PC != 0x1200 {
		t.Errorf("BXLE PC not correct got: %08x wanted: %08x", cpuState.PC, 0x1200)
	}

	// Add increment to first operand, compare with odd register after R3
	memory.SetMemory(0x400, 0x87132200) // BXLE 1, 3, 200(2)
	memory.SetMemory(0x1200, 0)         // Clear target of branch
	cpuState.regs[1] = 1                // Value
	cpuState.regs[2] = 0x1000           // Branch target
	cpuState.regs[3] = 0x12345678       // Incrment and Compare value
	cpuState.regs[4] = 0xffffffff       // Increment -1
	cpuState.regs[5] = 0x12345678       // Compare value

	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0x12345679 {
		t.Errorf("BXLE Register not correct got: %08x wanted: %08x", v, 0x12345679)
	}
	if cpuState.cc != 3 {
		t.Errorf("BXLE CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	if cpuState.PC != 0x404 {
		t.Errorf("BXLE PC not correct got: %08x wanted: %08x", cpuState.PC, 0x404)
	}

	// Add increment to first operand, compare with odd register after R3
	memory.SetMemory(0x400, 0x87132200) // BXLE 1, 3, 200(2)
	memory.SetMemory(0x1200, 0)         // Clear target of branch
	cpuState.regs[1] = 0xffffffff       // Value
	cpuState.regs[2] = 0x1000           // Branch target
	cpuState.regs[3] = 0x12345678       // Incrment and Compare value
	cpuState.regs[4] = 0xffffffff       // Increment -1
	cpuState.regs[5] = 0x12345678       // Compare value

	cpuState.testInst(0)
	v = cpuState.regs[1]
	if v != 0x12345677 {
		t.Errorf("BXLE Register not correct got: %08x wanted: %08x", v, 0x12345677)
	}
	if cpuState.cc != 3 {
		t.Errorf("BXLE CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	if cpuState.PC != 0x1200 {
		t.Errorf("BXLE PC not correct got: %08x wanted: %08x", cpuState.PC, 0x1200)
	}
}

// Test and instruction.
func TestCycleN(t *testing.T) {
	setup()

	memory.SetMemory(0x400, 0x54123454) // N 1,454(2,3)
	memory.SetMemory(0x404, 0)
	memory.SetMemory(0x954, 0x12345678)
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x200
	cpuState.regs[3] = 0x300
	cpuState.testInst(0)
	v := cpuState.regs[1]
	mv := uint32(0x11223344) & uint32(0x12345678)
	if v != mv {
		t.Errorf("N Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("N CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x54123454) // N 1,454(2,3)
	memory.SetMemory(0x404, 0)
	memory.SetMemory(0x954, 0x00000000)
	cpuState.regs[1] = 0xffffffff
	cpuState.regs[2] = 0x200
	cpuState.regs[3] = 0x300
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0xffffffff) & uint32(0x00000000)
	if v != mv {
		t.Errorf("N Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("N CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}
	// And register
	cpuState.cc = 3
	cpuState.regs[1] = 0xff00ff00
	cpuState.regs[2] = 0x12345678
	memory.SetMemory(0x400, 0x14120000) // NR 1,2
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x12005600)
	if v != mv {
		t.Errorf("N Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("N CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
	// And register zero result
	cpuState.cc = 3
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0xedcba987
	memory.SetMemory(0x400, 0x14120000) // NR 1,2
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x0)
	if v != mv {
		t.Errorf("N Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("N CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}
}

// Test or instruction.
func TestCycleO(t *testing.T) {
	setup()

	memory.SetMemory(0x400, 0x56123454) // O 1,454(2,3)
	memory.SetMemory(0x404, 0)
	memory.SetMemory(0x954, 0x12345678)
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x200
	cpuState.regs[3] = 0x300

	cpuState.testInst(0)
	v := cpuState.regs[1]
	mv := uint32(0x11223344) | uint32(0x12345678)
	if v != mv {
		t.Errorf("O Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("O CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x56123454) // O 1,454(2,3)
	memory.SetMemory(0x404, 0)
	memory.SetMemory(0x954, 0x00000000)
	cpuState.regs[1] = 0x00000000
	cpuState.regs[2] = 0x200
	cpuState.regs[3] = 0x300

	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("O Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("O CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Or register
	cpuState.cc = 3
	cpuState.regs[1] = 0xff00ff00
	cpuState.regs[2] = 0x12345678
	memory.SetMemory(0x400, 0x16120000) // OR 1,2
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0xff34ff78)
	if v != mv {
		t.Errorf("O Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("O CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Test or instruction.
func TestCycleX(t *testing.T) {
	setup()

	memory.SetMemory(0x400, 0x57123454) // X 1,454(2,3)
	memory.SetMemory(0x404, 0)
	memory.SetMemory(0x954, 0x12345678)
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x200
	cpuState.regs[3] = 0x300

	cpuState.testInst(0)
	v := cpuState.regs[1]
	mv := uint32(0x11223344) ^ uint32(0x12345678)
	if v != mv {
		t.Errorf("X Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("X CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x57123454) // X 1,454(2,3)
	memory.SetMemory(0x404, 0)
	memory.SetMemory(0x954, 0x11223344)
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x200
	cpuState.regs[3] = 0x300

	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = 0x00000000
	if v != mv {
		t.Errorf("X Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("X CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Exclusive or register
	cpuState.cc = 3
	cpuState.regs[1] = 0xff00ff00
	cpuState.regs[2] = 0x12345678
	memory.SetMemory(0x400, 0x17120000) // XR 1,2
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = 0xed34a978
	if v != mv {
		t.Errorf("X Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("X CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Shift left arithmetic single register.
func TestCycleSLA(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x00000001
	memory.SetMemory(0x400, 0x8b1f2001) // SLA 1,1(2)
	cpuState.testInst(0)
	v := cpuState.regs[1]
	mv := uint32(0x12345678) << 2
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SLA C not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Shift left single zero
	cpuState.cc = 3
	cpuState.regs[1] = 0x12345678
	memory.SetMemory(0x400, 0x8b100000) // SLA 1,0(0)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// // Shift left single zero negative
	cpuState.cc = 3
	cpuState.regs[1] = 0x92345678
	memory.SetMemory(0x400, 0x8b100000) // SLA 1,0(0)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x92345678)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Shift left single  zero, zero value
	cpuState.cc = 3
	cpuState.regs[1] = 0
	memory.SetMemory(0x400, 0x8b100000) // SLA 1,0(0)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Shift left single positive overflow
	cpuState.cc = 3
	cpuState.regs[1] = 0x10000000
	cpuState.regs[2] = 2
	memory.SetMemory(0x400, 0x8b1f2000) // SLA 1,0(2)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x40000000)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x10000000
	cpuState.regs[2] = 3
	memory.SetMemory(0x400, 0x8b1f2000) // SLA 1,0(2)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Shift left single
	cpuState.cc = 3
	cpuState.regs[1] = 0x7fffffff
	cpuState.regs[2] = 0x0000001f       // Shift by 31 shifts out entire number
	memory.SetMemory(0x400, 0x8b1f2000) // SLA 1,0(2)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	// Shift left single
	cpuState.cc = 3
	cpuState.regs[1] = 0x7fffffff
	cpuState.regs[2] = 0x00000020       // Shift by 32 shifts out entire number
	memory.SetMemory(0x400, 0x8b1f2000) // SLA 1,0(2)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x80000000
	cpuState.regs[2] = 0x0000001f       // Shift by 31 shifts out entire number
	memory.SetMemory(0x400, 0x8b1f2000) // SLA 1,0(2)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x80000000)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x80000000
	cpuState.regs[2] = 2                // Shift by 2 should overflow
	memory.SetMemory(0x400, 0x8b1f2000) // SLA 1,0(2)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x80000000)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x80000001
	cpuState.regs[2] = 2                // Shift by 2 should overflow
	memory.SetMemory(0x400, 0x8b1f2000) // SLA 1,0(2)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0x80000004)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0xf0000001
	cpuState.regs[2] = 0x00000001
	memory.SetMemory(0x400, 0x8b1f2001) // SLA 1,1(2)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0xc0000004)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// From Princ Ops p143
	cpuState.cc = 3
	cpuState.regs[2] = 0x007f0a72
	memory.SetMemory(0x400, 0x8b2f0008) // SLA 2,8(0) // Shift left 8
	cpuState.testInst(0)
	v = cpuState.regs[2]
	mv = uint32(0x7f0a7200)
	if v != mv {
		t.Errorf("SLA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SLA CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Shift left logical instruction.
func TestCycleSLL(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x82345678
	cpuState.regs[2] = 0x12340003       // Shift 3 bits
	memory.SetMemory(0x400, 0x891f2100) // SLL 1,100(2)
	cpuState.testInst(0)
	v := cpuState.regs[1]
	mv := uint32(0x11a2b3c0)
	if v != mv {
		t.Errorf("SLL Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLL CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	for i := range uint32(31) {
		cpuState.regs[1] = 1
		cpuState.regs[2] = 0x12340000 + i   // Shift i bits
		memory.SetMemory(0x400, 0x891f2100) // SLL 1,100(2)
		cpuState.testInst(0)
		v := cpuState.regs[1]
		mv := uint32(1 << i)
		if v != mv {
			t.Errorf("SLL Register not correct got: %08x wanted: %08x", v, mv)
		}
	}
}

// Shift right logical instruction.
func TestCycleSRL(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x82345678
	cpuState.regs[2] = 0x12340003       // Shift 3 bits
	memory.SetMemory(0x400, 0x881f2100) // SRL 1,100(2)
	cpuState.testInst(0)
	v := cpuState.regs[1]
	mv := uint32(0x82345678 >> 3)
	if v != mv {
		t.Errorf("SRL Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SRL CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Shift right arithmatic instruction.
func TestCycleSRA(t *testing.T) {
	setup()

	cpuState.regs[2] = 0x11223344
	memory.SetMemory(0x400, 0x8a2f0105) // SRA 2,105(0) // Shift right 5
	cpuState.testInst(0)
	v := cpuState.regs[2]
	mv := uint32(0x0089119a)
	if v != mv {
		t.Errorf("SRA Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SRA CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Shift right double logical.
func TestCycleSRDL(t *testing.T) {
	setup()

	cpuState.regs[4] = 0x12345678
	cpuState.regs[5] = 0xaabbccdd
	memory.SetMemory(0x400, 0x8c4f0118) // SRDL 4,118(0) // Shift right 24 (x18)
	cpuState.testInst(0)
	v := cpuState.regs[4]
	mv := uint32(0x00000012)
	if v != mv {
		t.Errorf("SRDL Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[5]
	mv = uint32(0x345678aa)
	if v != mv {
		t.Errorf("SRDL Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SRDL CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Shift left double logical.
func TestCycleSLDL(t *testing.T) {
	setup()

	cpuState.regs[4] = 0x12345678
	cpuState.regs[5] = 0xaabbccdd
	cpuState.regs[6] = 8
	memory.SetMemory(0x400, 0x8d4f6100) // SLDL 4,100(6)  // Shift left 8
	cpuState.testInst(0)
	v := cpuState.regs[4]
	mv := uint32(0x345678aa)
	if v != mv {
		t.Errorf("SLDL Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[5]
	mv = uint32(0xbbccdd00)
	if v != mv {
		t.Errorf("SLDL Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLDL CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[4] = 0x12345678
	cpuState.regs[5] = 0x00010001
	memory.SetMemory(0x400, 0x8d4f051b) // SLDL 4,51b(0) // Shift left 27
	cpuState.testInst(0)
	v = cpuState.regs[4]
	mv = uint32(0xc0000800)
	if v != mv {
		t.Errorf("SLDL Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[5]
	mv = uint32(0x08000000)
	if v != mv {
		t.Errorf("SLDL Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLDL CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[4] = 0x12345678
	cpuState.regs[5] = 0x00010001
	memory.SetMemory(0x400, 0x8d1f2100) // SLDL 1,100(2)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error("SLDL did not trap")
	}
	v = cpuState.regs[4]
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("SLDL Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[5]
	mv = uint32(0x00010001)
	if v != mv {
		t.Errorf("SLDL Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Shift double right arithmatic.
func TestCycleSRDA(t *testing.T) {
	setup()

	cpuState.regs[4] = 0x12345678
	cpuState.regs[5] = 0xaabbccdd
	cpuState.regs[6] = 8
	memory.SetMemory(0x400, 0x8e4f0118) // SRDA 4,118(0) // Shift right 24 (x18)
	cpuState.testInst(0)
	v := cpuState.regs[4]
	mv := uint32(0x00000012)
	if v != mv {
		t.Errorf("SRDA Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[5]
	mv = uint32(0x345678aa)
	if v != mv {
		t.Errorf("SRDA Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SRDA CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	cpuState.regs[4] = 0x02345678
	cpuState.regs[5] = 0xaabbccdd
	memory.SetMemory(0x400, 0x8e4f013c) // SRDA 4,13c(0) //  Shift right 60 (x3c)
	cpuState.testInst(0)
	v = cpuState.regs[4]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SRDA Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[5]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SRDA Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("SRDA CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	cpuState.regs[4] = 0x92345678
	cpuState.regs[5] = 0xaabbccdd
	memory.SetMemory(0x400, 0x8e4f0118) // SRDA 4,118(0) // Shift right 24 (x18)
	cpuState.testInst(0)
	v = cpuState.regs[4]
	mv = uint32(0xffffff92)
	if v != mv {
		t.Errorf("SRDA Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[5]
	mv = uint32(0x345678aa)
	if v != mv {
		t.Errorf("SRDA Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("SRDA CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Shift double left arithmatic.
func TestCycleSLDA(t *testing.T) {
	setup()

	cpuState.regs[2] = 0x007f0a72
	cpuState.regs[3] = 0xfedcba98
	memory.SetMemory(0x400, 0x8f2f001f) // SLDA 2,1f(0)
	cpuState.testInst(0)
	v := cpuState.regs[2]
	mv := uint32(0x7f6e5d4c)
	if v != mv {
		t.Errorf("SLDA Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[3]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SLDA Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLDA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[2] = 0xffffffff
	cpuState.regs[3] = 0xffffe070
	memory.SetMemory(0x400, 0x8f2f0030) // SLDA 2,30(0)
	cpuState.testInst(0)
	v = cpuState.regs[2]
	mv = uint32(0xe0700000)
	if v != mv {
		t.Errorf("SLDA Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[3]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SLDA Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("SLDA CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	cpuState.regs[2] = 0x92345678
	cpuState.regs[3] = 0xc0506070
	memory.SetMemory(0x400, 0x8f2f0020) // SLDA 2,20(0)
	cpuState.testInst(0)
	v = cpuState.regs[2]
	mv = uint32(0xc0506070)
	if v != mv {
		t.Errorf("SLDA Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[3]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SLDA Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLDA CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	cpuState.regs[2] = 0xff902030
	cpuState.regs[3] = 0x40506070
	memory.SetMemory(0x400, 0x8f2f0008) // SLDA 2,8(0)
	cpuState.testInst(0)
	v = cpuState.regs[2]
	mv = uint32(0x90203040)
	if v != mv {
		t.Errorf("SLDA Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[3]
	mv = uint32(0x50607000)
	if v != mv {
		t.Errorf("SLDA Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("SLDA CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	cpuState.regs[2] = 0x00000000
	cpuState.regs[3] = 0x000076f7
	memory.SetMemory(0x400, 0x8f2f0030) // SLDA 2,30(0)
	cpuState.testInst(0)
	v = cpuState.regs[2]
	mv = uint32(0x76f70000)
	if v != mv {
		t.Errorf("SLDA Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[3]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("SLDA Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SLDA CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Load multiple registers.
func TestCycleLM(t *testing.T) {
	setup()

	cpuState.regs[2] = 0xffffffff
	cpuState.regs[3] = 0x00000010
	cpuState.regs[4] = 0xffffffff
	cpuState.regs[5] = 0x00000000
	memory.SetMemory(0x110, 0x12345678)
	memory.SetMemory(0x114, 0x11223344)
	memory.SetMemory(0x118, 0x55667788)
	memory.SetMemory(0x11c, 0x99aabbcc)
	memory.SetMemory(0x400, 0x98253100) // LM 2,5,100(3)
	cpuState.testInst(0)
	v := cpuState.regs[2]
	mv := uint32(0x12345678)
	if v != mv {
		t.Errorf("LM Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[3]
	mv = uint32(0x11223344)
	if v != mv {
		t.Errorf("LM Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[4]
	mv = uint32(0x55667788)
	if v != mv {
		t.Errorf("LM Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[5]
	mv = uint32(0x99aabbcc)
	if v != mv {
		t.Errorf("LM Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("LM CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Store multiple registers.
func TestCycleSTM(t *testing.T) {
	setup()

	// From Princ Ops p143
	cpuState.regs[14] = 0x00002563
	cpuState.regs[15] = 0x00012736
	cpuState.regs[0] = 0x12430062
	cpuState.regs[1] = 0x73261257
	cpuState.regs[6] = 0x00004000
	memory.SetMemory(0x400, 0x90e16050) // STM 14,1,50(6)
	cpuState.testInst(0)
	v := memory.GetMemory(0x4050)
	mv := uint32(0x00002563)
	if v != mv {
		t.Errorf("STM Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x4054)
	mv = uint32(0x00012736)
	if v != mv {
		t.Errorf("STM Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x4058)
	mv = uint32(0x12430062)
	if v != mv {
		t.Errorf("STM Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x405c)
	mv = uint32(0x73261257)
	if v != mv {
		t.Errorf("STM Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("STM CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Test under mask.
func TestCycleTM(t *testing.T) {
	setup()

	// From Princ Ops p147
	memory.SetMemory(0x9998, 0xaafbaaaa)
	cpuState.regs[9] = 0x00009990
	memory.SetMemory(0x400, 0x91c39009) // TM 9(9),c3
	cpuState.testInst(0)
	if cpuState.cc != 3 {
		t.Errorf("TM CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x9998, 0xaa3caaaa)
	cpuState.regs[9] = 0x00009990
	memory.SetMemory(0x400, 0x91c39009) // TM 9(9),c3
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("TM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x9998, 0xaa3caaaa)
	cpuState.regs[9] = 0x00009990
	memory.SetMemory(0x400, 0x91c39009) // TM 9(9),c3
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("TM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x9998, 0xaa3caaaa)
	cpuState.regs[9] = 0x00009990
	memory.SetMemory(0x400, 0x91009008) // TM 9(9),c3
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("TM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x9998, 0xf03caaaa)
	cpuState.regs[9] = 0x00009990
	memory.SetMemory(0x400, 0x91f09008) // TM 9(9),c3
	cpuState.testInst(0)
	if cpuState.cc != 3 {
		t.Errorf("TM CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x9998, 0xa0f8aaaa)
	cpuState.regs[9] = 0x00009990
	memory.SetMemory(0x400, 0x910c9009) // TM 9(9),c3
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("TM CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Test to convert to binary
func TestCycleCVB(t *testing.T) {
	setup()

	// Example from Principles of Operation p122
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x00000000)
	memory.SetMemory(1004, 0x0025594f)
	memory.SetMemory(0x400, 0x4f756032) //  CVB 7,32(5,6)
	cpuState.testInst(0)
	v := cpuState.regs[7]
	mv := uint32(25594)
	if v != mv {
		t.Errorf("CVB 1 Register 7 not correct got: %08x wanted: %08x", v, mv)
	}

	// Test convert to binary with bad sign
	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x00000000)
	memory.SetMemory(1004, 0x00255941)  // 1 is not a valid sign
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error(("CVB 2 Should have trapped"))
	}

	// Test convert to binary with bad digit.
	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x00000000)
	memory.SetMemory(1004, 0x002a594f)
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error(("CVB 3 Should have trapped"))
	}

	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x00000214)
	memory.SetMemory(1004, 0x8000000f)
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error(("CVB 4 Should have trapped"))
	}
	v = cpuState.regs[7]
	mv = uint32(2148000000)
	if v != mv {
		t.Errorf("CVB 4 Register 7 not correct got: %08x wanted: %08x", v, mv)
	}

	// Test for overflow
	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x00000284)
	memory.SetMemory(1004, 0x4242842c)
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error(("CVB 5 Should have trapped"))
	}
	v = cpuState.regs[7]
	mv = uint32(0xa987b39a)
	if v != mv {
		t.Errorf("CVB 5 Register 7 not correct got: %08x wanted: %08x", v, mv)
	}

	// Test for larger overflow
	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x12345678)
	memory.SetMemory(1004, 0x4800000f)
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error(("CVB 6 Should have trapped"))
	}

	// Test big overflow
	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x12345678)
	memory.SetMemory(1004, 0x4800000f)
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error(("CVB 7 Should have trapped"))
	}

	// Test with large number
	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x00000021)
	memory.SetMemory(1004, 0x2345678f)
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	v = cpuState.regs[7]
	mv = uint32(212345678)
	if v != mv {
		t.Errorf("CVB 8 Register 7 not correct got: %08x wanted: %08x", v, mv)
	}

	//  Test negative
	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x00000000)
	memory.SetMemory(1004, 0x0025594d)  // d is negative
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	v = cpuState.regs[7]
	mv = uint32(0xffff9c06)
	if v != mv {
		t.Errorf("CVB 9 Register 7 not correct got: %08x wanted: %08x", v, mv)
	}

	// test model 50 case QE900/073C, CLF 112
	cpuState.cc = 3
	cpuState.regs[5] = 0x100
	cpuState.regs[6] = 0x200
	memory.SetMemory(0x500, 0)
	memory.SetMemory(0x504, 0x1234567f) // Decimal 1234567+
	memory.SetMemory(0x400, 0x4f156200) // CVB 1,200(5,6)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(1234567)
	if v != mv {
		t.Errorf("CVB 10 Register ` not correct got: %08x wanted: %08x", v, mv)
	}

	// Second test with negative
	cpuState.cc = 3
	cpuState.regs[5] = 0x100
	cpuState.regs[6] = 0x200
	memory.SetMemory(0x500, 0)
	memory.SetMemory(0x504, 0x1234567b) // Decimal 1234567-
	memory.SetMemory(0x400, 0x4f156200) // CVB 1,200(5,6)
	cpuState.testInst(0)
	v = cpuState.regs[1]
	mv = uint32(0xffed2979)
	if v != mv {
		t.Errorf("CVB 11 Register 1 not correct got: %08x wanted: %08x", v, mv)
	}

	cpuState.cc = 3
	cpuState.regs[5] = 50 // Example seems to have addresses in decimal?
	cpuState.regs[6] = 900
	memory.SetMemory(1000, 0x00000214)
	memory.SetMemory(1004, 0x8000000f)
	memory.SetMemory(0x400, 0x4f756032) // CVB 7,32(5,6)
	cpuState.testInst(0)
	v = cpuState.regs[7]
	mv = uint32(2148000000)
	if v != mv {
		t.Errorf("CVB 12 Register 7 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Test convert to decimal.
func TestCycleCVD(t *testing.T) {
	setup()

	// Example from Principles of Operation p122
	cpuState.regs[1] = 0x00000f0f // 3855 dec
	cpuState.regs[13] = 0x00007600
	memory.SetMemory(0x400, 0x4e10d008) // CVD 1,8(0,13)
	cpuState.testInst(0)
	v := memory.GetMemory(0x7608)
	mv := uint32(0x00000000)
	if v != mv {
		t.Errorf("CVD Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x760C)
	mv = uint32(0x0003855c)
	if v != mv {
		t.Errorf("CVD Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}

	cpuState.regs[1] = 0xfffff0f1 // -3855 dec
	cpuState.regs[13] = 0x00007600
	memory.SetMemory(0x400, 0x4e10d008) // CVD 1,8(0,13)
	cpuState.testInst(0)
	v = memory.GetMemory(0x7608)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("CVD Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x760C)
	mv = uint32(0x0003855d)
	if v != mv {
		t.Errorf("CVD Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Move immeditate.
func TestCycleMVI(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x3456
	memory.SetMemory(0x3464, 0x12345678)
	memory.SetMemory(0x400, 0x92421010) // MVI 10(1),42
	cpuState.testInst(0)
	v := memory.GetMemory(0x3464)
	mv := uint32(0x12344278)
	if v != mv {
		t.Errorf("MVI Memory not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("MVI CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}

	cpuState.cc = 3
	memory.SetMemory(0x100, 0x11223344)
	cpuState.regs[1] = 1
	memory.SetMemory(0x400, 0x92551100) // MVI 100(1),55 // Move byte 55 to location 101
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x11553344)
	if v != mv {
		t.Errorf("MVI Memory not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("MVI CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// And immediate.
func TestCycleNI(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x3456
	memory.SetMemory(0x3464, 0x12345678)
	memory.SetMemory(0x400, 0x94f01010) // NI 10(1),f0
	cpuState.testInst(0)
	v := memory.GetMemory(0x3464)
	mv := uint32(0x12345078)
	if v != mv {
		t.Errorf("NI Memory not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("NI CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x3456
	memory.SetMemory(0x3464, 0x12345678)
	memory.SetMemory(0x400, 0x940f1010) // NI 10(1),f0
	cpuState.testInst(0)
	v = memory.GetMemory(0x3464)
	mv = uint32(0x12340678)
	if v != mv {
		t.Errorf("NI Memory not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("NI CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x3456
	memory.SetMemory(0x3464, 0x12345678)
	memory.SetMemory(0x400, 0x94001010) // NI 10(1),0
	cpuState.testInst(0)
	v = memory.GetMemory(0x3464)
	mv = uint32(0x12340078)
	if v != mv {
		t.Errorf("NII Memory not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("NI CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}
}

// Compare logical immediate.
func TestCycleCLI(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x3456
	memory.SetMemory(0x3464, 0x12345678)
	memory.SetMemory(0x400, 0x95561010) // CLI 10(1),56
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLI CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x3456
	memory.SetMemory(0x3464, 0x12345678)
	memory.SetMemory(0x400, 0x95ff1010) // CLI 10(1),ff
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CLI CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	for i := range uint32(255) {
		cpuState.regs[1] = 0x3442
		memory.SetMemory(0x3450, 0x12345678)
		memory.SetMemory(0x400, 0x95001010|(i<<16)) // CLI 10(1),i
		cpuState.testInst(0)
		var cc uint8
		switch x := i; {
		case x == 0x56: // Equal
			cc = 0
		case x < 0x56: // Greater
			cc = 2
		default: // Less
			cc = 1
		}
		if cpuState.cc != cc {
			t.Errorf("CLI CC not correct got: %x wanted: %x", cpuState.cc, cc)
		}
	}
}

// Or immediate.
func TestCycleOI(t *testing.T) {
	setup()

	cpuState.regs[1] = 2
	memory.SetMemory(0x1000, 0x12345678)
	memory.SetMemory(0x400, 0x96421fff) // OI fff(1),42
	cpuState.testInst(0)
	v := memory.GetMemory(0x1000)
	mv := uint32(0x12765678)
	if v != mv {
		t.Errorf("OI Memory not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("OI CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Xor immediate.
func TestCycleXI(t *testing.T) {
	setup()

	cpuState.regs[0] = 0x100
	memory.SetMemory(0x120, 0x12345678)
	memory.SetMemory(0x400, 0x970f0123) // XI 123(0),f
	cpuState.testInst(0)
	v := memory.GetMemory(0x120)
	mv := uint32(0x12345677)
	if v != mv {
		t.Errorf("XI Memory not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("XI CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Move numeric.
func TestCycleMVN(t *testing.T) {
	setup()

	// From Princ Ops p144
	memory.SetMemory(0x7090, 0xc1c2c3c4)
	memory.SetMemory(0x7094, 0xc5c6c7c8)
	memory.SetMemory(0x7040, 0xaaf0f1f2)
	memory.SetMemory(0x7044, 0xf3f4f5f6)
	memory.SetMemory(0x7048, 0xf7f8aaaa)
	cpuState.regs[14] = 0x00007090
	cpuState.regs[15] = 0x00007040
	memory.SetMemory(0x400, 0xd103f001)
	memory.SetMemory(0x404, 0xe0000000) // MVN 1(4,15),0(14)
	cpuState.testInst(0)
	v := memory.GetMemory(0x7090)
	mv := uint32(0xc1c2c3c4)
	if v != mv {
		t.Errorf("MVN Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x7040)
	mv = uint32(0xaaf1f2f3)
	if v != mv {
		t.Errorf("MVN Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x7044)
	mv = uint32(0xf4f4f5f6)
	if v != mv {
		t.Errorf("MVN Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x7048)
	mv = uint32(0xf7f8aaaa)
	if v != mv {
		t.Errorf("MVN Memory 4 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Move character.
func TestCycleMVC(t *testing.T) {
	setup()

	memory.SetMemory(0x100, 0x12345678)
	memory.SetMemory(0x200, 0x11223344)
	memory.SetMemory(0x400, 0xd2030100)
	memory.SetMemory(0x404, 0x02000000) // MVC 100(4,0),200(0) // Move 4 bytes from 200 to 100
	cpuState.testInst(0)
	v := memory.GetMemory(0x100)
	mv := uint32(0x11223344)
	if v != mv {
		t.Errorf("MVC Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x200)
	mv = uint32(0x11223344)
	if v != mv {
		t.Errorf("MVC Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}

	memory.SetMemory(0x100, 0x12345678)
	memory.SetMemory(0x104, 0xabcdef01)
	cpuState.regs[1] = 2
	cpuState.regs[2] = 0
	memory.SetMemory(0x400, 0xd2011100)
	memory.SetMemory(0x404, 0x01050000) // MVC 100(2,1),105(0) // Move 2 bytes from 105 to 102
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x1234cdef)
	if v != mv {
		t.Errorf("MVC Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x104)
	mv = uint32(0xabcdef01)
	if v != mv {
		t.Errorf("MVC Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Move zones.
func TestCycleMVZ(t *testing.T) {
	setup()

	// From Princ Ops page 144
	memory.SetMemory(0x800, 0xf1c2f3c4)
	memory.SetMemory(0x804, 0xf5c6aabb)
	cpuState.regs[15] = 0x00000800
	memory.SetMemory(0x400, 0xd304f001)
	memory.SetMemory(0x404, 0xf0000000) // MVZ 1(5,15),0(15)
	cpuState.testInst(0)
	v := memory.GetMemory(0x800)
	mv := uint32(0xf1f2f3f4)
	if v != mv {
		t.Errorf("MVZ Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x804)
	mv = uint32(0xf5f6aabb)
	if v != mv {
		t.Errorf("MVZ Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Move offset.
func TestCycleMVO(t *testing.T) {
	setup()

	// Princ Ops 152
	cpuState.regs[12] = 0x00005600
	cpuState.regs[15] = 0x00004500
	memory.SetMemory(0x5600, 0x7788990c)
	memory.SetMemory(0x4500, 0x123456ff)
	memory.SetMemory(0x400, 0xf132c000)
	memory.SetMemory(0x404, 0xf0000000) // MVO 0(4, 12), 0(3, 15)
	cpuState.testInst(0)
	v := memory.GetMemory(0x5600)
	mv := uint32(0x0123456c)
	if v != mv {
		t.Errorf("MVO Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Move inverse.
func TestCycleMVIN(t *testing.T) {
	setup()

	memory.SetMemory(0x200, 0xC1C2C3C4)
	memory.SetMemory(0x204, 0xC5C6C7C8)
	memory.SetMemory(0x208, 0xC9CACB00)
	memory.SetMemory(0x300, 0xF1F2F3F4)
	memory.SetMemory(0x304, 0xF5F6F7F8)
	memory.SetMemory(0x308, 0xF9000000)
	memory.SetMemory(0x400, 0xe8070200) // MVINV 200(7),300
	memory.SetMemory(0x404, 0x03070000)
	cpuState.testInst(0)
	v := memory.GetMemory(0x200)
	mv := uint32(0xF8F7F6F5)
	if v != mv {
		t.Errorf("MVIN Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x204)
	mv = uint32(0xF4F3F2F1)
	if v != mv {
		t.Errorf("MVIN Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x208)
	mv = uint32(0xC9CACB00)
	if v != mv {
		t.Errorf("MVIN Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Pack instruction.
func TestCyclePACK(t *testing.T) {
	setup()

	// Princ Ops p151
	cpuState.regs[12] = 0x00001000
	memory.SetMemory(0x1000, 0xf1f2f3f4)
	memory.SetMemory(0x1004, 0xc5000000)
	memory.SetMemory(0x400, 0xf244c000)
	memory.SetMemory(0x404, 0xc0000000) // PACK 0(5, 12), 0(5, 12)
	cpuState.testInst(0)
	v := memory.GetMemory(0x1000)
	mv := uint32(0x00001234)
	if v != mv {
		t.Errorf("PACK Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1004)
	mv = uint32(0x5c000000)
	if v != mv {
		t.Errorf("PACK Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Unpack.
func TestCycleUNPK(t *testing.T) {
	setup()

	// Princ Ops p151
	cpuState.regs[12] = 0x00001000
	cpuState.regs[13] = 0x00002500
	memory.SetMemory(0x2500, 0xaa12345d)
	memory.SetMemory(0x1000, 0xffffffff)
	memory.SetMemory(0x1004, 0xffffffff)
	memory.SetMemory(0x400, 0xf342c000)
	memory.SetMemory(0x404, 0xd0010000) // UNPK 0(5, 12), 1(3, 13)
	cpuState.testInst(0)
	v := memory.GetMemory(0x1000)
	mv := uint32(0xf1f2f3f4)
	if v != mv {
		t.Errorf("UNPK Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1004)
	mv = uint32(0xd5ffffff)
	if v != mv {
		t.Errorf("UNPK Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// And characters.
func TestCycleNC(t *testing.T) {
	setup()

	memory.SetMemory(0x358, 0x00001790)
	memory.SetMemory(0x360, 0x00001401)
	cpuState.regs[7] = 0x00000358
	memory.SetMemory(0x400, 0xd4037000)
	memory.SetMemory(0x404, 0x70080000) // NC 0(4,7),8(7)
	cpuState.testInst(0)
	v := memory.GetMemory(0x358)
	mv := uint32(0x00001400)
	if v != mv {
		t.Errorf("NC Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Compare logical character.
func TestCycleCLC(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x100
	memory.SetMemory(0x200, 0x12345633)
	memory.SetMemory(0x300, 0x12345644)
	memory.SetMemory(0x400, 0xd5021100)
	memory.SetMemory(0x404, 0x22000000) // CLC 100(3,1),200(2)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLI CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x100
	memory.SetMemory(0x200, 0x12345678)
	memory.SetMemory(0x300, 0x12345678)
	// 123456 vs 345678 because of offset
	memory.SetMemory(0x400, 0xd5021100)
	memory.SetMemory(0x404, 0x22010000) // CLC 100(3,1),201(2)
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CLI CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Or character.
func TestCycleOC(t *testing.T) {
	setup()

	memory.SetMemory(0x358, 0x00001790)
	memory.SetMemory(0x360, 0x00001401)
	cpuState.regs[7] = 0x00000358
	memory.SetMemory(0x400, 0xd6037000)
	memory.SetMemory(0x404, 0x7008aaaa) // OC 0(4,7),8(7)
	cpuState.testInst(0)
	v := memory.GetMemory(0x358)
	mv := uint32(0x00001791)
	if v != mv {
		t.Errorf("OC Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
}

// exclusive or character.
func TestCycleXC(t *testing.T) {
	setup()

	memory.SetMemory(0x358, 0x00001790)
	memory.SetMemory(0x360, 0x00001401)
	cpuState.regs[7] = 0x00000358
	memory.SetMemory(0x400, 0xd7037000)
	memory.SetMemory(0x404, 0x70080000) // XC 0(4,7),8(7)
	cpuState.testInst(0)
	v := memory.GetMemory(0x358)
	mv := uint32(0x00000391)
	if v != mv {
		t.Errorf("XC Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}

	memory.SetMemory(0x400, 0xd7037008)
	memory.SetMemory(0x404, 0x70000000) // XC 8(4,7),0(7)
	cpuState.testInst(0)
	v = memory.GetMemory(0x360)
	mv = uint32(0x00001790)
	if v != mv {
		t.Errorf("XC Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}

	memory.SetMemory(0x400, 0xd7037000)
	memory.SetMemory(0x404, 0x70080000) // XC 0(4,7),8(7)
	cpuState.testInst(0)
	v = memory.GetMemory(0x358)
	mv = uint32(0x00001401)
	if v != mv {
		t.Errorf("XC Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
}

// translate.
func TestCycleTR(t *testing.T) {
	setup()

	// Based on Princ Ops p147
	for i := uint32(0); i < 256; i += 4 {
		// Table increments each char by 3. Don't worry about wrapping.
		memory.SetMemory(0x1000+i, (((i + 3) << 24) |
			((i + 4) << 16) |
			((i + 5) << 8) |
			(i + 6)))
	}
	memory.SetMemory(0x2100, 0x12345678)
	memory.SetMemory(0x2104, 0xabcdef01)
	memory.SetMemory(0x2108, 0x11223344)
	memory.SetMemory(0x210c, 0x55667788)
	memory.SetMemory(0x2110, 0x99aabbcc)
	cpuState.regs[12] = 0x00002100
	cpuState.regs[15] = 0x00001000
	memory.SetMemory(0x400, 0xdc13c000)
	memory.SetMemory(0x404, 0xf0000000) // TR 0(20,12),0(15)
	cpuState.testInst(0)
	v := memory.GetMemory(0x2100)
	mv := uint32(0x1537597b)
	if v != mv {
		t.Errorf("TR Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x2104)
	mv = uint32(0xaed0f204)
	if v != mv {
		t.Errorf("TR Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x2108)
	mv = uint32(0x14253647)
	if v != mv {
		t.Errorf("TR Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x210c)
	mv = uint32(0x58697a8b)
	if v != mv {
		t.Errorf("TR Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x2110)
	mv = uint32(0x9cadbecf)
	if v != mv {
		t.Errorf("TR Memory 4 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Translate and test.
func TestCycleTRT(t *testing.T) {
	setup()

	// Based on Princ Ops p147
	for i := uint32(0); i < 256; i += 4 {
		memory.SetMemory(0x2000+i, 0)
	}
	memory.SetMemory(0x204c, 0x10202500)
	memory.SetMemory(0x2050, 0x90000000)
	memory.SetMemory(0x2058, 0x00000030)
	memory.SetMemory(0x205c, 0x35404500)
	memory.SetMemory(0x2060, 0x80850000)
	memory.SetMemory(0x2068, 0x00000050)
	memory.SetMemory(0x206c, 0x55000000)
	memory.SetMemory(0x2078, 0x00000060)
	memory.SetMemory(0x207c, 0x65707500)

	memory.SetMemory(0x3000, 0x40404040)
	memory.SetMemory(0x3004, 0x40e4d5d7) //  UNP
	memory.SetMemory(0x3008, 0xd2404040) // K
	memory.SetMemory(0x300c, 0x4040d7d9) //   PR
	memory.SetMemory(0x3010, 0xd6e4e34d) // OUT(
	memory.SetMemory(0x3014, 0xf95d6be6) // 9),W
	memory.SetMemory(0x3018, 0xd6d9c44d) // ORD(
	memory.SetMemory(0x301C, 0xf55d0000) // 5)

	cpuState.regs[1] = 0x3000
	cpuState.regs[2] = 0
	cpuState.regs[15] = 0x2000

	memory.SetMemory(0x400, 0xdd1d1000) // TRT 0(30,1),0(15)
	memory.SetMemory(0x404, 0xf0000000)
	cpuState.testInst(0)
	v := cpuState.regs[1] // Match at 3013
	mv := uint32(0x00003013)
	if v != mv {
		t.Errorf("TRT Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[2] // Function value from table
	mv = uint32(0x00000020)
	if v != mv {
		t.Errorf("TRT Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 { // not complete
		t.Errorf("TRT CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Based on Princ Ops p147
	for i := uint32(0); i < 256; i += 4 {
		memory.SetMemory(0x1000+i, 0)
	}
	memory.SetMemory(0x2020, 0x10203040)
	memory.SetMemory(0x3000, 0x12345621) // 21 will match table entry 20
	memory.SetMemory(0x3004, 0x11223344)
	memory.SetMemory(0x3008, 0x55667788)
	memory.SetMemory(0x300c, 0x99aabbcc)
	memory.SetMemory(0x400, 0xdd0f1000)
	memory.SetMemory(0x404, 0xf0000000) // TRT 0(16,1),0(15)
	cpuState.regs[1] = 0x3000
	cpuState.regs[2] = 0
	cpuState.regs[15] = 0x2000
	cpuState.testInst(0)
	v = cpuState.regs[1] // Match at 3013
	mv = uint32(0x00003003)
	if v != mv {
		t.Errorf("TRT Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[2] // Function value from table
	mv = uint32(0x00000020)
	if v != mv {
		t.Errorf("TRT Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 { // not complete
		t.Errorf("TRT CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Test SPM instruction.
func TestCycleSPM(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x12345678       // Mask 2
	memory.SetMemory(0x400, 0x041f0000) // SPM 1
	cpuState.testInst(0)
	v := cpuState.progMask
	mv := uint8(0x2)
	if v != mv {
		t.Errorf("SPM Mask not correct got: %02x wanted: %02x", v, mv)
	}
	if cpuState.cc != 1 { // not complete
		t.Errorf("SPM CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Test SSM instruction.
func TestCycleSSM(t *testing.T) {
	setup()

	cpuState.sysMask = 0xff00
	cpuState.stKey = 0x30
	cpuState.flags = 0x0 // privileged
	cpuState.cc = 1
	cpuState.regs[3] = 0x11
	memory.SetMemory(0x110, 0xaabbccdd) // Access byte 1
	memory.SetMemory(0x400, 0x80ee3100) // "SSM 100(3)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	v := cpuState.sysMask
	mv := uint16(0xBBFF)
	if v != mv {
		t.Errorf("SSM Mask not correct got: %04x wanted: %04x", v, mv)
	}
	v1 := cpuState.stKey
	mv1 := uint8(0x30)
	if v1 != mv1 {
		t.Errorf("SSM Key not correct got: %02x wanted: %02x", v1, mv1)
	}
	v1 = cpuState.progMask
	mv1 = uint8(0x0)
	if v1 != mv1 {
		t.Errorf("SSM Prog Mask not correct got: %02x wanted: %02x", v1, mv1)
	}
	if cpuState.cc != 1 {
		t.Errorf("SPM CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
	v2 := cpuState.PC
	mv2 := uint32(0x404)
	if v2 != mv2 {
		t.Errorf("SSM PC not correct got: %04x wanted: %04x", v2, mv2)
	}
	if !cpuState.extEnb {
		t.Error("SSM External mask not set")
	}
	cpuState.stKey = 0x00

	cpuState.sysMask = 0xff00
	cpuState.flags = 0x1 // problem state
	cpuState.cc = 1
	cpuState.regs[3] = 0x11
	memory.SetMemory(0x110, 0xaabbccdd) // Access byte 1
	memory.SetMemory(0x400, 0x80ee3100) // "SSM 100(3)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error("SSM in problem state did not trap")
	}
	cpuState.flags = 0
}

// Test lpsw instruction.
func TestCycleLPSW(t *testing.T) {
	setup()

	cpuState.stKey = 0
	cpuState.flags = 0x0 // privileged
	cpuState.regs[3] = 0x10
	memory.SetMemory(0x110, 0xE1345678)
	memory.SetMemory(0x114, 0x9a003450)  // Branch to 123450
	memory.SetMemory(0x400, 0x82003100)  // LPSW 100(3)
	memory.SetMemory(0x3450, 0x00000000) // Nop in case things are executed
	cpuState.testInst(0)
	v := cpuState.sysMask
	mv := uint16(0xe000)
	if v != mv {
		t.Errorf("LPSW Mask not correct got: %04x wanted: %04x", v, mv)
	}
	v1 := cpuState.stKey
	mv1 := uint8(0x30)
	if v1 != mv1 {
		t.Errorf("LPSW Key not correct got: %02x wanted: %02x", v1, mv1)
	}
	v1 = cpuState.progMask
	mv1 = uint8(0xa)
	if v1 != mv1 {
		t.Errorf("LPSW Prog Mask not correct got: %02x wanted: %02x", v1, mv1)
	}
	if cpuState.cc != 1 {
		t.Errorf("LPSW CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
	v2 := cpuState.PC
	mv2 := uint32(0x003450)
	if v2 != mv2 {
		t.Errorf("LPSW PC not correct got: %04x wanted: %04x", v2, mv2)
	}
	if !cpuState.extEnb {
		t.Error("LPSW External mask not set")
	}
	cpuState.stKey = 0x00
}

// Supervisory call.
func TestCycleSVC(t *testing.T) {
	setup()

	cpuState.stKey = 0
	cpuState.flags = 0x1 // privileged
	cpuState.sysMask = 0xe000
	cpuState.extEnb = true
	cpuState.cc = 1
	cpuState.regs[3] = 0x10
	memory.SetMemory(0x60, 0xE1345678)
	memory.SetMemory(0x64, 0x9a003450)   // Branch to 3450
	memory.SetMemory(0x400, 0x0a120000)  // SVC 12
	memory.SetMemory(0x3450, 0x00000000) // Nop in case things are executed
	cpuState.testInst(0x4)
	v := cpuState.sysMask
	mv := uint16(0xe000)
	if v != mv {
		t.Errorf("SVC Mask not correct got: %04x wanted: %04x", v, mv)
	}
	v1 := cpuState.stKey
	mv1 := uint8(0x30)
	if v1 != mv1 {
		t.Errorf("SVC Key not correct got: %02x wanted: %02x", v1, mv1)
	}
	v1 = cpuState.progMask
	mv1 = uint8(0xa)
	if v1 != mv1 {
		t.Errorf("SVC Prog Mask not correct got: %02x wanted: %02x", v1, mv1)
	}
	if cpuState.cc != 1 {
		t.Errorf("SVC CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
	v2 := cpuState.PC
	mv2 := uint32(0x003450)
	if v2 != mv2 {
		t.Errorf("SVC PC not correct got: %04x wanted: %04x", v2, mv2)
	}
	if !cpuState.extEnb {
		t.Error("SVC External mask not set")
	}
	v2 = memory.GetMemory(0x20)
	mv2 = uint32(0xE1010012)
	if v2 != mv2 {
		t.Errorf("TR Memory 1 not correct got: %08x wanted: %08x", v2, mv2)
	}
	v2 = memory.GetMemory(0x24)
	mv2 = uint32(0x54000402)
	if v2 != mv2 {
		t.Errorf("SVC Memory 2 not correct got: %08x wanted: %08x", v2, mv2)
	}
	cpuState.stKey = 0x00
}

// Set storage key.
func TestCycleSSK(t *testing.T) {
	setup()

	cpuState.flags = 0x1          // unprivileged
	cpuState.regs[1] = 0x11223344 // Key
	cpuState.regs[2] = 0x00005600 // Address: last 4 bits must be 0
	memory.PutKey(0x5600, 0x40)
	memory.SetMemory(0x400, 0x08120000) // SSK 1,2
	cpuState.testInst(0)
	if !trapFlag {
		t.Error("SSK should have trapped")
	}
	if memory.GetKey(0x5600) != 0x40 {
		t.Errorf("SSK unprivileged changed key got: %02x expected: %02x", memory.GetKey(0x5600), 0x40)
	}

	cpuState.flags = 0x0          // privileged
	cpuState.regs[1] = 0x11223344 // Key
	cpuState.regs[2] = 0x00005600 // Address: last 4 bits must be 0
	memory.PutKey(0x5600, 0)
	memory.SetMemory(0x400, 0x08120000) // SSK 1,2
	cpuState.testInst(0)
	if memory.GetKey(0x5600) != 0x40 {
		t.Errorf("SSK privileged did not changed key got: %02x expected: %02x", memory.GetKey(0x5600), 0x40)
	}

	cpuState.flags = 0x0          // privileged
	cpuState.regs[1] = 0x11223344 // Key
	cpuState.regs[2] = 0x12345674 // Address: last 4 bits must be 0
	memory.PutKey(0x5600, 0x70)
	memory.SetMemory(0x400, 0x08120000) // SSK 1,2
	cpuState.testInst(0)
	if !trapFlag {
		t.Error("SSK should have trapped")
	}
	if memory.GetKey(0x5600) != 0x70 {
		t.Errorf("SSK unaligned changed key got: %02x expected: %02x", memory.GetKey(0x5600), 0x70)
	}
}

// ISK reads the storage key
func TestCycleISK(t *testing.T) {
	setup()

	cpuState.flags = 0x1          // unprivileged
	cpuState.regs[1] = 0x11223344 // Key
	cpuState.regs[2] = 0x00005600 // Address: last 4 bits must be 0
	memory.PutKey(0x5600, 0x40)
	memory.SetMemory(0x400, 0x09120000) // ISK 1,2
	cpuState.testInst(0)
	if !trapFlag {
		t.Error("ISK should have trapped")
	}

	cpuState.flags = 0x0          // privileged
	cpuState.regs[1] = 0x89abcdef // Key
	cpuState.regs[2] = 0x00005600 // Address: last 4 bits must be 0
	memory.PutKey(0x5600, 0x20)
	memory.SetMemory(0x400, 0x09120000) // ISK 1,2
	cpuState.testInst(0)
	v := cpuState.regs[1]
	mv := uint32(0x89abcd20)
	if v != mv {
		t.Errorf("ISK Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if memory.GetKey(0x5600) != 0x20 {
		t.Errorf("ISK privileged changed key got: %02x expected: %02x", memory.GetKey(0x5600), 0x20)
	}

	cpuState.flags = 0x0          // privileged
	cpuState.regs[1] = 0x11223344 // Key
	cpuState.regs[2] = 0x12345674 // Address: last 4 bits must be 0
	memory.PutKey(0x5600, 0x70)
	memory.SetMemory(0x400, 0x09120000) // ISK 1,2
	cpuState.testInst(0)
	if !trapFlag {
		t.Error("ISK should have trapped")
	}
	if memory.GetKey(0x5600) != 0x70 {
		t.Errorf("ISK unaligned changed key got: %02x expected: %02x", memory.GetKey(0x5600), 0x70)
	}
}

// Protection check. unmatched key
func TestCycleProt(t *testing.T) {
	setup()

	cpuState.flags = 0x1 // unprivileged
	cpuState.stKey = 0x20
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x00005670
	memory.SetMemory(0x5678, 0xff)
	memory.PutKey(0x5600, 0x40)
	memory.SetMemory(0x400, 0x50102008) // st 1,0(2)
	cpuState.testInst(0)
	if !trapFlag {
		t.Error("Store to wrong key did not trap")
	}
	v := memory.GetMemory(0x5678)
	mv := uint32(0xff)
	if v != mv {
		t.Errorf("Store to wrong key changed memory correct got: %08x wanted: %08x", v, mv)
	}

	cpuState.flags = 0x1 // unprivileged
	cpuState.stKey = 0x40
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x00005670
	memory.SetMemory(0x5678, 0xff)
	memory.PutKey(0x5600, 0x40)
	memory.SetMemory(0x400, 0x50102008) // st 1,0(2)
	cpuState.testInst(0)
	if trapFlag {
		t.Error("Store to correct key traped")
	}
	v = memory.GetMemory(0x5678)
	mv = uint32(0x11223344)
	if v != mv {
		t.Errorf("Store to wrong key changed memory correct got: %08x wanted: %08x", v, mv)
	}

	cpuState.flags = 0x1 // unprivileged
	cpuState.stKey = 0x20
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x00005670
	memory.SetMemory(0x5678, 0xff)
	memory.PutKey(0x5600, 0x40)
	memory.SetMemory(0x400, 0x58102008) // l 1,0(2)
	memory.SetMemory(0x5678, 0x12345678)
	cpuState.testInst(0)
	if trapFlag {
		t.Error("Load to wrong key traped")
	}
	v = memory.GetMemory(0x5678)
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("Load to wrong key did not load register correct got: %08x wanted: %08x", v, mv)
	}

	cpuState.flags = 0x1 // unprivileged
	cpuState.stKey = 0x40
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x00005670
	memory.SetMemory(0x5678, 0xff)
	memory.PutKey(0x5600, 0x40)
	memory.SetMemory(0x400, 0x58102008) // l 1,0(2)
	memory.SetMemory(0x5678, 0x12345678)
	cpuState.testInst(0)
	if trapFlag {
		t.Error("Load to correct key traped")
	}
	v = memory.GetMemory(0x5678)
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("Load to correct key did not change register correct got: %08x wanted: %08x", v, mv)
	}

	cpuState.flags = 0x1 // unprivileged
	cpuState.stKey = 0x00
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x00005670
	memory.SetMemory(0x5678, 0xff)
	memory.PutKey(0x5600, 0x40)
	memory.SetMemory(0x400, 0x50102008) // st 1,0(2)
	cpuState.testInst(0)
	if trapFlag {
		t.Error("Store zero key did trap")
	}
	v = memory.GetMemory(0x5678)
	mv = uint32(0x11223344)
	if v != mv {
		t.Errorf("Store to zero did not update memory correct got: %08x wanted: %08x", v, mv)
	}

	// Test fetch protection
	cpuState.flags = 0x1 // unprivileged
	cpuState.stKey = 0x20
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x00005670
	memory.SetMemory(0x5678, 0xff)
	memory.PutKey(0x5600, 0x41)
	memory.SetMemory(0x400, 0x58102008) // l 1,0(2)
	memory.SetMemory(0x5678, 0x12345678)
	cpuState.testInst(0)
	if trapFlag {
		t.Error("Load to fetch protected key traped")
	}
	v = memory.GetMemory(0x5678)
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("Load to fetch protected key did load register correct got: %08x wanted: %08x", v, mv)
	}

	cpuState.flags = 0x1 // unprivileged
	cpuState.stKey = 0x40
	cpuState.regs[1] = 0x11223344
	cpuState.regs[2] = 0x00005670
	memory.SetMemory(0x5678, 0xff)
	memory.PutKey(0x5600, 0x41)
	memory.SetMemory(0x400, 0x58102008) // l 1,0(2)
	memory.SetMemory(0x5678, 0x12345678)
	cpuState.testInst(0)
	if trapFlag {
		t.Error("Load to  fetch correct key traped")
	}
	v = memory.GetMemory(0x5678)
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("Load to correct key did change register correct got: %08x wanted: %08x", v, mv)
	}
	cpuState.flags = 0
	cpuState.stKey = 0x00
	memory.PutKey(0x5600, 0)
}

// Test and set.
func TestCycleTS(t *testing.T) {
	setup()

	cpuState.regs[2] = 2                // Index
	memory.SetMemory(0x100, 0x83857789) // 102 top bit not set
	memory.SetMemory(0x400, 0x93002100) // TS 100(2)
	cpuState.testInst(0)
	v := memory.GetMemory(0x100)
	mv := uint32(0x8385ff89)
	if v != mv {
		t.Errorf("TS Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 { // not complete
		t.Errorf("TS CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	cpuState.regs[2] = 2                // Index
	memory.SetMemory(0x100, 0x8385c789) // 102 top bit not set
	memory.SetMemory(0x400, 0x93002100) // TS 100(2)
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x8385ff89)
	if v != mv {
		t.Errorf("TS Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 { // not complete
		t.Errorf("TS CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Edit test.
func TestCycleED(t *testing.T) {
	setup()

	cpuState.regs[12] = 0x1000
	memory.SetMemory(0x1200, 0x0257426c)
	memory.SetMemory(0x1000, 0x4020206b)
	memory.SetMemory(0x1004, 0x2020214b)
	memory.SetMemory(0x1008, 0x202040c3)
	memory.SetMemory(0x100c, 0xd9ffffff)
	memory.SetMemory(0x400, 0xde0cc000)
	memory.SetMemory(0x404, 0xc2000000) // ED 0(13,12),200(12)
	cpuState.testInst(0)
	v := memory.GetMemory(0x1000)
	mv := uint32(0x4040f26b)
	if v != mv {
		t.Errorf("ED Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1004)
	mv = uint32(0xf5f7f44b)
	if v != mv {
		t.Errorf("ED Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1008)
	mv = uint32(0xf2f64040)
	if v != mv {
		t.Errorf("ED Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x100c)
	mv = uint32(0x40ffffff)
	if v != mv {
		t.Errorf("ED Memory 4 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 { // not complete
		t.Errorf("ED CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.regs[12] = 0x1000
	memory.SetMemory(0x1200, 0x0000026d)
	memory.SetMemory(0x1000, 0x4020206b)
	memory.SetMemory(0x1004, 0x2020214b)
	memory.SetMemory(0x1008, 0x202040c3)
	memory.SetMemory(0x100c, 0xd9ffffff)
	memory.SetMemory(0x400, 0xde0cc000)
	memory.SetMemory(0x404, 0xc2000000) // ED 0(13,12),200(12)
	cpuState.testInst(0)
	v = memory.GetMemory(0x1000)
	mv = uint32(0x40404040)
	if v != mv {
		t.Errorf("ED Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1004)
	mv = uint32(0x4040404b)
	if v != mv {
		t.Errorf("ED Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1008)
	mv = uint32(0xf2f640c3)
	if v != mv {
		t.Errorf("ED Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x100c)
	mv = uint32(0xd9ffffff)
	if v != mv {
		t.Errorf("ED Memory 4 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 { // not complete
		t.Errorf("ED CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Edit and mark.
func TestCycleEDMK(t *testing.T) {
	setup()

	cpuState.regs[1] = 0xaabbccdd
	cpuState.regs[12] = 0x1000
	memory.SetMemory(0x1200, 0x0000026d)
	memory.SetMemory(0x1000, 0x4020206b)
	memory.SetMemory(0x1004, 0x2020214b)
	memory.SetMemory(0x1008, 0x202040c3)
	memory.SetMemory(0x100c, 0xd9ffffff)
	memory.SetMemory(0x400, 0xdf0cc000)
	memory.SetMemory(0x404, 0xc2000000) // EDMK 0(13,12),200(12)
	cpuState.testInst(0)
	v := memory.GetMemory(0x1000)
	mv := uint32(0x40404040)
	if v != mv {
		t.Errorf("EDMK Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1004)
	mv = uint32(0x4040404b)
	if v != mv {
		t.Errorf("EDMK Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1008)
	mv = uint32(0xf2f640c3)
	if v != mv {
		t.Errorf("EDMK Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x100c)
	mv = uint32(0xd9ffffff)
	if v != mv {
		t.Errorf("EDMK Memory 4 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[1]
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("EDMK Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 { // not complete
		t.Errorf("EDMK CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.regs[1] = 0xaabbccdd
	cpuState.regs[12] = 0x1000
	memory.SetMemory(0x1200, 0x0000026d)
	memory.SetMemory(0x1000, 0x4020206b)
	memory.SetMemory(0x1004, 0x2020204b)
	memory.SetMemory(0x1008, 0x202040c3)
	memory.SetMemory(0x100c, 0xd9ffffff)
	memory.SetMemory(0x400, 0xdf0cc000)
	memory.SetMemory(0x404, 0xc2000000) // EDMK 0(13,12),200(12)
	cpuState.testInst(0)
	v = memory.GetMemory(0x1000)
	mv = uint32(0x40404040)
	if v != mv {
		t.Errorf("EDMK Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1004)
	mv = uint32(0x40404040)
	if v != mv {
		t.Errorf("EDMK Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1008)
	mv = uint32(0xf2f640c3)
	if v != mv {
		t.Errorf("EDMK Memory 3 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x100c)
	mv = uint32(0xd9ffffff)
	if v != mv {
		t.Errorf("EDMK Memory 4 not correct got: %08x wanted: %08x", v, mv)
	}
	v = cpuState.regs[1]
	mv = uint32(0xaa001008)
	if v != mv {
		t.Errorf("EDMK Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 { // not complete
		t.Errorf("EDMK CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

func TestCycleCLM(t *testing.T) {
	setup()
	cpuState.regs[1] = 0xFF00FF00
	cpuState.regs[2] = 0x00FFFF00
	memory.SetMemory(0x500, 0xFFFFFFFF)
	memory.SetMemory(0x400, 0xbd1a0500) // CLM 1,b'1010', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // CLM 2,b'0110', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd130500) // CLM 1,b'0011', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.regs[1] = 0x01050102
	cpuState.regs[2] = 0x00010203
	memory.SetMemory(0x500, 0x01020304)
	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd190500) // CLM 1,b'1001', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // CLM 2,b'0110', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd150500) // CLM 1,b'0101', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd230500) // CLM 2,b'0011', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

func TestCycleICM(t *testing.T) {
	setup()
	cpuState.regs[1] = 0x00000000
	memory.SetMemory(0x500, 0x01020304)
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0x00000102 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0x00000102)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x00000000
	memory.SetMemory(0x400, 0xbF190500) // ICM 1,b'1001', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0x01000002 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0x01000002)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0xd0d00102 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0xd0d00102)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x400, 0xbF190500) // ICM 1,b'1001', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0x01d0d002 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0x01d0d002)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x00000000
	memory.SetMemory(0x400, 0xbF170500) // ICM 1,b'0111', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0x00010203 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0x00010203)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x00000000
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
	if cpuState.regs[1] != 0x0000F0F1 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0x0000F0F1)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbF190500) // ICM 1,b'1001', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
	if cpuState.regs[1] != 0xf0d0d0f1 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0xf0d0d0f1)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x500, 0x00000000)
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}
	if cpuState.regs[1] != 0xD0D00000 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0xD0D00000)
	}

	cpuState.regs[1] = 0x01050102
	cpuState.regs[2] = 0x00010203
	memory.SetMemory(0x500, 0x01020304)
	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd190500) // ICM 1,b'1001', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // ICM 2,b'0110', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd150500) // ICM 1,b'0101', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd230500) // ICM 2,b'0011', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

func TestCycleSTCM(t *testing.T) {
	setup()
	cpuState.cc = 3
	cpuState.regs[3] = 0xf0f1f2f3
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbe330500) // STCM 3,b'0011', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 3 {
		t.Errorf("STCM CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	v := memory.GetMemory(0x500)
	if v != 0xf2f3f2f3 {
		t.Errorf("STCM memory not correct got: %x wanted: %x", v, 0xf2f3f2f3)
	}
	if cpuState.regs[3] != 0xf0f1f2f3 {
		t.Errorf("STCM R3 not correct got: %x wanted: %x", cpuState.regs[3], 0xf0f1f2f3)
	}

	cpuState.cc = 3
	cpuState.regs[3] = 0xf0f1f2f3
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbe390500) // STCM 3,b'1001', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 3 {
		t.Errorf("STCM CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	v = memory.GetMemory(0x500)
	if v != 0xf0f3f2f3 {
		t.Errorf("STCM memory not correct got: %x wanted: %x", v, 0xf0f3f2f3)
	}
	if cpuState.regs[3] != 0xf0f1f2f3 {
		t.Errorf("STCM R3 not correct got: %x wanted: %x", cpuState.regs[3], 0xf0f1f2f3)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0xf0f1f2f3
	memory.SetMemory(0x500, 0x00000000)
	memory.SetMemory(0x400, 0xbe3f0500) // STCM 3,b'1111', 500
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 3 {
		t.Errorf("STCM CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
	v = memory.GetMemory(0x500)
	if v != 0xf0f1f2f3 {
		t.Errorf("STCM memory not correct got: %x wanted: %x", v, 0xf0f1f2f3)
	}
	if cpuState.regs[1] != 0xf0f1f2f3 {
		t.Errorf("STCM R3 not correct got: %x wanted: %x", cpuState.regs[1], 0xf0f1f2f3)
	}
}

func TestCycleCLCL(t *testing.T) {
	setup()

	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x504, 0xf4f5f6f7)
	memory.SetMemory(0x508, 0xf8f9f0f0)
	memory.SetMemory(0x50c, 0xf0f0f0f0)
	memory.SetMemory(0x510, 0xf0f0f0f0)
	memory.SetMemory(0x600, 0xf0f1f2f3)
	memory.SetMemory(0x604, 0xf4f5f6f7)
	memory.SetMemory(0x608, 0xf8f9f0f0)
	memory.SetMemory(0x60c, 0xf0f0f0f0)
	memory.SetMemory(0x610, 0xf0f0f0f0)
	cpuState.regs[2] = 0x500
	cpuState.regs[3] = 20
	cpuState.regs[4] = 0x600
	cpuState.regs[5] = 20
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	if cpuState.regs[2] != 0x500+20 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpuState.regs[2], 0x500+20)
	}

	if cpuState.regs[3] != 0 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpuState.regs[3], 0)
	}

	if cpuState.regs[4] != 0x600+20 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpuState.regs[4], 0x600+20)
	}

	if cpuState.regs[5] != 0 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpuState.regs[5], 0)
	}

	cpuState.regs[2] = 0x500
	cpuState.regs[3] = 20
	cpuState.regs[4] = 0x600
	cpuState.regs[5] = 0xf0000000 + 10
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	if cpuState.regs[2] != 0x500+20 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpuState.regs[2], 0x500+20)
	}

	if cpuState.regs[3] != 0 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpuState.regs[3], 0)
	}

	if cpuState.regs[4] != 0x600+10 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpuState.regs[4], 0x600+10)
	}

	if cpuState.regs[5] != 0xf0000000 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpuState.regs[5], 0xf0000000)
	}

	cpuState.regs[2] = 0x500
	cpuState.regs[3] = 10
	cpuState.regs[4] = 0x600
	cpuState.regs[5] = 0xf0000000 + 20
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	if cpuState.regs[2] != 0x500+10 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpuState.regs[2], 0x500+10)
	}

	if cpuState.regs[3] != 0 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpuState.regs[3], 0)
	}

	if cpuState.regs[4] != 0x600+20 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpuState.regs[4], 0x600+20)
	}

	if cpuState.regs[5] != 0xf0000000 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpuState.regs[5], 0xf0000000)
	}

	memory.SetMemory(0x600, 0xf0f1f2f3)
	memory.SetMemory(0x604, 0xf4f5f6f7)
	memory.SetMemory(0x608, 0xf8f9f9f9)
	memory.SetMemory(0x60c, 0xf9f9f9f9)
	memory.SetMemory(0x610, 0xf9f9f9f9)
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4

	cpuState.regs[2] = 0x500
	cpuState.regs[3] = 20
	cpuState.regs[4] = 0x600
	cpuState.regs[5] = 20
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	if cpuState.regs[2] != 0x500+10 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpuState.regs[2], 0x500+10)
	}

	if cpuState.regs[3] != 10 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpuState.regs[3], 10)
	}

	if cpuState.regs[4] != 0x600+10 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpuState.regs[4], 0x600+10)
	}

	if cpuState.regs[5] != 10 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpuState.regs[5], 10)
	}

	memory.SetMemory(0x400, 0x0f420000) // CLCL 4,2

	cpuState.regs[2] = 0x500
	cpuState.regs[3] = 20
	cpuState.regs[4] = 0x600
	cpuState.regs[5] = 20
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	if cpuState.regs[2] != 0x500+10 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpuState.regs[2], 0x500+10)
	}

	if cpuState.regs[3] != 10 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpuState.regs[3], 10)
	}

	if cpuState.regs[4] != 0x600+10 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpuState.regs[4], 0x600+10)
	}

	if cpuState.regs[5] != 10 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpuState.regs[5], 10)
	}

	cpuState.regs[2] = 0x500
	cpuState.regs[3] = 5
	cpuState.regs[4] = 0x600
	cpuState.regs[5] = 0xf5000000 + 20
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	if cpuState.regs[2] != 0x500+5 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpuState.regs[2], 0x500+5)
	}

	if cpuState.regs[3] != 0 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpuState.regs[3], 0)
	}

	if cpuState.regs[4] != 0x600+6 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpuState.regs[4], 0x600+6)
	}

	if cpuState.regs[5] != 0xf500000e {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpuState.regs[5], 0xf500000e)
	}
}

// Basic Add Packed Decimal tests.
func TestCycleAP(t *testing.T) {
	setup()

	// Short field
	memory.SetMemory(0x100, 0x0000002c) // 2+
	memory.SetMemory(0x200, 0x00003c00) // 3+
	memory.SetMemory(0x400, 0xfa000103) // AP 103(1,0),202(1,0)
	memory.SetMemory(0x404, 0x02020000)
	cpuState.testInst(0)
	v := memory.GetMemory(0x100)
	mv := uint32(0x0000005c)
	if v != mv {
		t.Errorf("AP Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Add one
	memory.SetMemory(0x100, 0x2888011c) // 2888011+
	memory.SetMemory(0x200, 0x1112292c) // 1112292+
	memory.SetMemory(0x400, 0xfa330100) // AP 100(4,0),200(4,0)
	memory.SetMemory(0x404, 0x02000000)
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x4000303c)
	if v != mv {
		t.Errorf("AP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Add one
	memory.SetMemory(0x100, 0x0000002c) // 2+
	memory.SetMemory(0x200, 0x0000003c) // 3+
	memory.SetMemory(0x400, 0xfa330100) // AP 100(4,0),200(4,0)
	memory.SetMemory(0x404, 0x02000000)
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x0000005c)
	if v != mv {
		t.Errorf("AP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Add packed with offset
	memory.SetMemory(0x100, 0x0043212c) // 2+
	memory.SetMemory(0x200, 0x0023413c) // 3+
	memory.SetMemory(0x400, 0xfa220101) // AP 101(3,0),201(3,0)
	memory.SetMemory(0x404, 0x02010000)
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x0066625c)
	if v != mv {
		t.Errorf("AP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Add packed no offset
	memory.SetMemory(0x100, 0x0043212c) // 2+
	memory.SetMemory(0x200, 0x0023413c) // 3+
	memory.SetMemory(0x400, 0xfa330100) // AP 100(4,0),200(4,0)
	memory.SetMemory(0x404, 0x02000000)
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x0066625c)
	if v != mv {
		t.Errorf("AP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Add packed offset
	// Example from Princ Ops p136.2
	cpuState.regs[12] = 0x00002000
	cpuState.regs[13] = 0x000004fd
	memory.SetMemory(0x2000, 0x38460d00) // 38460-
	memory.SetMemory(0x500, 0x0112345c)  // 112345+
	memory.SetMemory(0x400, 0xfa23c000)  // AP 0(3,12),3(4,13)
	memory.SetMemory(0x404, 0xd0030000)
	cpuState.testInst(0)
	v = memory.GetMemory(0x2000)
	mv = uint32(0x73885c00)
	if v != mv {
		t.Errorf("AP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Add packed
	// Example from Princ Ops p136.2
	cpuState.regs[12] = 0x00002000
	cpuState.regs[13] = 0x000004fd
	memory.SetMemory(0x2000, 0x0038460d)
	memory.SetMemory(0x500, 0x0112345c)
	memory.SetMemory(0x400, 0xfa33c000)
	memory.SetMemory(0x404, 0xd0030000) // AP 0(4, 12), 3(4, 13)
	cpuState.testInst(0)
	v = memory.GetMemory(0x2000)
	mv = uint32(0x0073885c)
	if v != mv {
		t.Errorf("AP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Basic Zero Add Packed Decimal tests.
func TestCycleZAP(t *testing.T) {
	setup()

	cpuState.regs[9] = 0x00004000
	memory.SetMemory(0x4000, 0x12345678)
	memory.SetMemory(0x4004, 0x90aaaaaa)
	memory.SetMemory(0x4500, 0x38460dff)
	memory.SetMemory(0x400, 0xf8429000)
	memory.SetMemory(0x404, 0x95000000) // ZAP 0(5, 9), 500(3, 9)
	cpuState.testInst(0)
	v := memory.GetMemory(0x4000)
	mv := uint32(0x00003846)
	if v != mv {
		t.Errorf("ZAP Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x4004)
	mv = uint32(0x0daaaaaa)
	if v != mv {
		t.Errorf("ZAP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("ZAP CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Zap short field
	memory.SetMemory(0x100, 0x2a000000) // 2+
	memory.SetMemory(0x200, 0x3a000000) // 3+
	memory.SetMemory(0x400, 0xf8000100)
	memory.SetMemory(0x404, 0x02000000) // ZAP 100(1, 0), 200(1, 0)
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x3c000000)
	if v != mv {
		t.Errorf("ZAP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("ZAP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Zap with offset
	memory.SetMemory(0x100, 0x002a0000) // 2+
	memory.SetMemory(0x200, 0x00003a00) // 3+
	memory.SetMemory(0x400, 0xf8000101)
	memory.SetMemory(0x404, 0x02020000) // ZAP 101(1, 0), 202(1, 0)
	cpuState.testInst(0)
	v = memory.GetMemory(0x100)
	mv = uint32(0x003c0000)
	if v != mv {
		t.Errorf("ZAP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("ZAP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Compare packed.
func TestCycleCP(t *testing.T) {
	setup()

	// Princ Op page 150
	cpuState.regs[12] = 0x00000600
	cpuState.regs[13] = 0x00000400
	memory.SetMemory(0x700, 0x1725356d)
	memory.SetMemory(0x500, 0x0672142d)
	memory.SetMemory(0x400, 0xf933c100)
	memory.SetMemory(0x404, 0xd1000000) // CP 100(4, 12), 100(4, 13)
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CP CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Compare packed  equal
	cpuState.regs[12] = 0x00000600
	cpuState.regs[13] = 0x00000400
	memory.SetMemory(0x700, 0x1725356d)
	memory.SetMemory(0x500, 0x00172535)
	memory.SetMemory(0x504, 0x6d000000)
	memory.SetMemory(0x400, 0xf933c100)
	memory.SetMemory(0x404, 0xd1010000) // CP 100(4, 12), 101(4, 13)
	cpuState.testInst(0)

	if cpuState.cc != 0 {
		t.Errorf("CP CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Compare packed first higher
	cpuState.regs[12] = 0x00000600
	cpuState.regs[13] = 0x00000400
	memory.SetMemory(0x700, 0x1725346d)
	memory.SetMemory(0x500, 0x00172535)
	memory.SetMemory(0x504, 0x6d000000)
	memory.SetMemory(0x400, 0xf933c100)
	memory.SetMemory(0x404, 0xd1010000) // CP 100(4, 12), 101(4, 13)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("CP CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

// Subtract packed.
func TestCycleSP(t *testing.T) {
	setup()

	cpuState.regs[12] = 0x00002000
	cpuState.regs[13] = 0x000004fc
	memory.SetMemory(0x2000, 0x0038460c)
	memory.SetMemory(0x500, 0x0112345c)
	memory.SetMemory(0x400, 0xfb33c000)
	memory.SetMemory(0x404, 0xd0040000) // SP 0(4, 12), 3(4, 13)
	cpuState.testInst(0)
	v := memory.GetMemory(0x2000)
	mv := uint32(0x0073885d)
	if v != mv {
		t.Errorf("SP Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("SP CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}
}

// Multiply packed.
func TestCycleMP(t *testing.T) {
	setup()

	cpuState.regs[4] = 0x00001200
	cpuState.regs[6] = 0x00000500
	memory.SetMemory(0x1300, 0x00003846)
	memory.SetMemory(0x1304, 0x0cffffff)
	memory.SetMemory(0x500, 0x321dffff)
	memory.SetMemory(0x400, 0xfc414100)
	memory.SetMemory(0x404, 0x60000000) // MP 100(5, 4), 0(2, 6)
	cpuState.testInst(0)
	v := memory.GetMemory(0x1300)
	mv := uint32(0x01234566)
	if v != mv {
		t.Errorf("MP Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x1304)
	mv = uint32(0x0dffffff)
	if v != mv {
		t.Errorf("MP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Divide packed.
func TestCycleDP(t *testing.T) {
	setup()

	cpuState.regs[12] = 0x00002000
	cpuState.regs[13] = 0x00003000
	memory.SetMemory(0x2000, 0x01234567)
	memory.SetMemory(0x2004, 0x8cffffff)
	memory.SetMemory(0x3000, 0x321dffff)
	memory.SetMemory(0x400, 0xfd41c000)
	memory.SetMemory(0x404, 0xd0000000) // DP 0(5, 12), 0(2, 13)
	cpuState.testInst(0)
	v := memory.GetMemory(0x2000)
	mv := uint32(0x38460d01)
	if v != mv {
		t.Errorf("DP Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x2004)
	mv = uint32(0x8cffffff)
	if v != mv {
		t.Errorf("DP Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Do a bunch of canned tests with Packed Decimal instructions.
var hexDigits = "0123456789abcdef"

type decCase struct {
	op  uint8
	i1  string
	i2  string
	out string
	cc  uint8
	ex  uint8
}

var cases = []decCase{
	{OpAP, "2c", "3c", "5c", 2, 0},
	{OpSP, "1c", "7c", "6d", 1, 0},
	{OpAP, "1c", "7c", "8c", 2, 0},
	{OpSP, "9c", "5c", "4c", 2, 0},
	{OpAP, "9c", "5c", "4c", 3, 10},
	{OpSP, "009c", "5d", "014c", 2, 0},
	{OpSP, "1d", "1d", "0c", 0, 0},
	{OpAP, "12345c", "54321c", "66666c", 2, 0},
	{OpSP, "12345c", "54321c", "41976d", 1, 0},
	{OpSP, "54321c", "12345c", "41976c", 2, 0},
	{OpSP, "54321c", "01234d", "55555c", 2, 0},
	{OpSP, "12345c", "54321d", "66666c", 2, 0},
	{OpAP, "12345d", "54321d", "66666d", 1, 0},
	{OpAP, "012c", "052c", "064c", 2, 0},
	{OpAP, "072c", "012c", "084c", 2, 0},
	{OpAP, "095c", "023c", "118c", 2, 0},
	{OpSP, "095c", "023d", "118c", 2, 0},
	{OpSP, "012c", "532c", "520d", 1, 0},
	{OpAP, "171c", "053c", "224c", 2, 0},
	{OpSP, "171d", "053c", "224d", 1, 0},
	{OpAP, "053d", "171d", "224d", 1, 0},
	{OpAP, "1c", "2c", "3c", 2, 0},
	{OpAP, "072c", "025d", "047c", 2, 0},
	{OpAP, "072d", "080c", "008c", 2, 0},
	{OpSP, "77532c", "12345c", "65187c", 2, 0},
	{OpAP, "9c", "018d", "9d", 1, 0},
	{OpSP, "6c", "014c", "8d", 1, 0},
	{OpSP, "8d", "019d", "1c", 3, 10},
	{OpAP, "7d", "016c", "9c", 2, 0},
	{OpMP, "0000125c", "752c", "0094000c", 2, 0},
	{OpMP, "012345", "654321", "012345", 0, 7},
	{OpMP, "5c", "5c", "5c", 0, 6},
	{OpMP, "005c", "5c", "025c", 0, 0},
	{OpMP, "005c", "005c", "025c", 0, 6},
	{OpMP, "005c", "012c", "005c", 0, 6},
	{OpMP, "006c", "013c", "006c", 0, 6},
	{OpMP, "00004c", "017c", "00068c", 0, 0},
	{OpMP, "005c", "215c", "005c", 0, 6},
	{OpMP, "00006c", "135c", "00810c", 0, 0},
	{OpMP, "00004c", "023c", "00092c", 0, 0},
	{OpMP, "007c", "9c", "063c", 0, 0},
	{OpMP, "009d", "8c", "072d", 0, 0},
	{OpMP, "018c", "2c", "036c", 0, 7},
	{OpMP, "008d", "3d", "024c", 0, 0},
	{OpMP, "001d", "0c", "000d", 0, 0},
	{OpMP, "000c", "052d", "000c", 0, 6},
	{OpMP, "00000014142c", "14142c", "00199996164c", 0, 0},
	{OpMP, "00000017320c", "17320c", "00299982400c", 0, 0},
	{OpMP, "0000000223607d", "0223607c", "0000000223607d", 0, 7},
	{OpMP, "002236067977499c", "3d", "006708203932497d", 0, 0},
	{OpMP, "001414213562373d", "2d", "002828427124746c", 0, 0},
	{OpMP, "022360679774997c", "3d", "022360679774997c", 0, 7},
	{OpMP, "014142135623730d", "2d", "014142135623730d", 0, 7},
	{OpMP, "002236067977499c", "029d", "002236067977499c", 0, 7},
	{OpMP, "001414213562373d", "021d", "001414213562373d", 0, 7},
	{OpMP, "000223606797749c", "029d", "000223606797749c", 0, 7},
	{OpMP, "000141421356237d", "021d", "000141421356237d", 0, 7},
	{OpMP, "022360697774997c", "9d", "022360697774997c", 0, 7},
	{OpMP, "074142315623730d", "8d", "074142315623730d", 0, 7},
	{OpMP, "000000000000005c", "0123456c", "000000000617280c", 0, 0},
	{OpMP, "000000000000005c", "1234567c", "000000006172835c", 0, 0},
	{OpMP, "000000000000003c", "012345678c", "000000037037034c", 0, 0},
	{OpMP, "000000000000015c", "0123456c", "000000001851840c", 0, 0},
	{OpMP, "000000000000025c", "1234567c", "000000030864175c", 0, 0},
	{OpMP, "000000000000093c", "012345678c", "000001148148054c", 0, 0},
	{OpMP, "000000001234567c", "1234567c", "001524155677489c", 0, 0},
	{OpMP, "000000001234567c", "012345678c", "000000001234567c", 0, 7},
	{OpMP, "000000001234567c", "123456789c", "000000001234567c", 0, 7},
	{OpMP, "0001234c", "025c", "0001234c", 0, 7},
	{OpMP, "0001243d", "017c", "0001243d", 0, 7},
	{OpMP, "0005432c", "071d", "0005432c", 0, 7},
	{OpMP, "0000123d", "176d", "0021648c", 0, 0},
	{OpMP, "0000512c", "01068c", "0000512c", 0, 7},
	{OpMP, "002c", "2c", "004c", 0, 0},
	{OpMP, "004c", "4c", "016c", 0, 0},
	{OpMP, "008c", "8c", "064c", 0, 0},
	{OpMP, "00016c", "016c", "00016c", 0, 7},
	{OpMP, "0000032c", "032c", "0001024c", 0, 0},
	{OpMP, "0000064c", "064c", "0004096c", 0, 0},
	{OpMP, "0000128c", "128c", "0016384c", 0, 0},
	{OpMP, "0000256c", "256c", "0065536c", 0, 0},
	{OpMP, "0000512c", "512c", "0262144c", 0, 0},
	{OpMP, "00000001024c", "01024c", "00001048576c", 0, 0},
	{OpMP, "00000002048c", "02048c", "00004194304c", 0, 0},
	{OpMP, "00000004096c", "04096c", "00016777216c", 0, 0},
	{OpMP, "00000008192c", "08192c", "00067108864c", 0, 0},
	{OpMP, "00000016384c", "16384c", "00268435456c", 0, 0},
	{OpMP, "00000032768c", "32768c", "01073741824c", 0, 0},
	{OpMP, "00000065536c", "65536c", "04294967296c", 0, 0},
	{OpMP, "000000000131072c", "0131072c", "000017179869184c", 0, 0},
	{OpMP, "000000000524288c", "0524288c", "000274877906944c", 0, 0},
	{OpMP, "000000002097152c", "0131072c", "000274877906944c", 0, 0},
	{OpMP, "000000002097152c", "65536c", "000137438953472c", 0, 0},
	{OpMP, "000000002097152c", "2097152c", "004398046511104c", 0, 0},
	{OpMP, "000002147483646c", "512c", "001099511626752c", 0, 0},
	{OpMP, "000002147483646c", "08192c", "000002147483646c", 0, 7},
	{OpMP, "000002147483646c", "16384c", "000002147483646c", 0, 7},
	{OpMP, "000002147483646c", "65536c", "000002147483646c", 0, 7},
	{OpMP, "004398046511104c", "8c", "035184372088832c", 0, 0},
	{OpMP, "004398046511104c", "064c", "004398046511104c", 0, 7},
	{OpMP, "000549755813888c", "08192c", "000549755813888c", 0, 7},
	{OpMP, "000549755813888c", "512c", "000549755813888c", 0, 7},
	{OpMP, "000549755813888c", "064c", "000549755813888c", 0, 7},
	{OpMP, "000549755813888c", "8c", "004398046511104c", 0, 0},
	{OpMP, "000068719476736c", "16384c", "000068719476736c", 0, 7},
	{OpMP, "000068719476736c", "04096c", "000068719476736c", 0, 7},
	{OpMP, "000068719476736c", "512c", "035184372088832c", 0, 0},
	{OpMP, "7c", "7d", "7c", 0, 6},
	{OpMP, "025c", "3d", "025c", 0, 7},
	{OpMP, "7d", "8d", "7d", 0, 6},
	{OpDP, "77325c", "025c", "77325c", 0, 11},
	{OpDP, "066c", "1c", "066c", 0, 11},
	{OpDP, "072c", "3d", "072c", 0, 11},
	{OpDP, "066d", "2c", "066d", 0, 11},
	{OpDP, "072c", "1c", "072c", 0, 11},
	{OpDP, "072c", "0c", "072c", 0, 11},
	{OpDP, "000077325c", "025c", "03093c000c", 0, 0},
	{OpDP, "0000066c", "2c", "00033c0c", 0, 0},
	{OpDP, "00066c", "2c", "033c0c", 0, 0},
	{OpDP, "00066c", "2c", "033c0c", 0, 0},
	{OpDP, "066c", "2c", "066c", 0, 11},
	{OpDP, "0123456c", "072c", "0123456c", 0, 11},
	{OpDP, "0123456c", "072c", "0123456c", 0, 11},
	{OpDP, "000123456c", "072c", "01714c048c", 0, 0},
	{OpDP, "000123456c", "072c", "01714c048c", 0, 0},
	{OpDP, "00000123456c", "072c", "0001714c048c", 0, 0},
	{OpDP, "00004398046511104c", "064c", "0068719476736c000c", 0, 0},
	{OpDP, "00004398046511104c", "064c", "0068719476736c000c", 0, 0},
	{OpDP, "004398046511104c", "064c", "68719476736c000c", 0, 0},
	{OpDP, "004398046511104c", "064c", "68719476736c000c", 0, 0},
	{OpDP, "00000043980465111c", "653c", "0000067351401c258c", 0, 0},
	{OpDP, "00000439804651110c", "653c", "0000673514013c621c", 0, 0},
	{OpDP, "00004398046511104c", "653c", "0006735140139c337c", 0, 0},
	{OpDP, "00004398046511104c", "653c", "0006735140139c337c", 0, 0},
	{OpDP, "004398046511104c", "653c", "06735140139c337c", 0, 0},
	{OpDP, "043980465111040c", "653c", "67351401395c105c", 0, 0},
	{OpDP, "439804651110400c", "653c", "439804651110400c", 0, 11},
	{OpDP, "0000435d", "7c", "00062d1d", 0, 0},
	{OpDP, "0000435c", "7d", "00062d1c", 0, 0},
	{OpDP, "0000435d", "7d", "00062c1d", 0, 0},
	{OpDP, "0000251d", "7d", "00035c6d", 0, 0},
	{OpDP, "0000252d", "7d", "00036c0d", 0, 0},
	{OpDP, "0000253d", "7d", "00036c1d", 0, 0},
	{OpDP, "00000d", "1c", "000d0d", 0, 0},
	{OpDP, "00001d", "1c", "001d0d", 0, 0},
	{OpDP, "00001c", "1c", "001c0c", 0, 0},
	{OpDP, "00000c", "1d", "000d0c", 0, 0},
	{OpDP, "00000c", "1c", "000c0c", 0, 0},
	{OpDP, "00000c", "0c", "00000c", 0, 11},
	{OpDP, "0000000000725c", "1234567c", "00000c0000725c", 0, 0},
	{OpDP, "0000000000725c", "012345678c", "000c000000725c", 0, 0},
	{OpDP, "1234567c", "1234567c", "1234567c", 0, 6},
	{OpDP, "012345678c", "1234567c", "012345678c", 0, 11},
	{OpDP, "000000008c", "1234567c", "0c0000008c", 0, 0},
	{OpDP, "000000008c", "0123456c", "0c0000008c", 0, 0},
	{OpDP, "000000008c", "12345c", "000c00008c", 0, 0},
	{OpDP, "0000000000000006543210987654321c", "123456789012345c", "000000000000053c000001170000036c", 0, 0},
	{OpDP, "0000000000006543210987654321000c", "123456789012345c", "000000000053000c001170000036000c", 0, 0},
	{OpDP, "0000000006543210987654321000111c", "123456789012345c", "000000053000009c058888934889006c", 0, 0},
	{OpDP, "0000006543210987654321000111222c", "123456789012345c", "000053000009477c000046530117657c", 0, 0},
	{OpDP, "0000043210987654321000111222333c", "123456789012345c", "000350009003150c010253617335583c", 0, 0},
	{OpDP, "0000543210987654321000111222333c", "123456789012345c", "004400009039600c013044117360333c", 0, 0},
	{OpDP, "0006543210987654321000111222333c", "123456789012345c", "053000009477000c046530117657333c", 0, 0},
	{OpDP, "0076543210987654321000111222333c", "123456789012345c", "620000014580003c066829754085298c", 0, 0},
	{OpDP, "0876543210987654321000111222333c", "123456789012345c", "0876543210987654321000111222333c", 0, 11},
	{OpDP, "6543210987654321000111222333444c", "123456789012345c", "6543210987654321000111222333444c", 0, 11},
	{OpDP, "0000000000000000000000000000000c", "123456789012345c", "000000000000000c000000000000000c", 0, 0},
	{OpDP, "0000000000000000000000000000000c", "01234567890123456c", "0000000000000000000000000000000c", 0, 6},
	{OpMVO, "512c", "001068", "068c", 0, 0},
	{OpMVO, "7788990c", "123456", "0123456c", 0, 0},
	{OpMVO, "0001234c", "025c", "000025cc", 0, 0},
	{OpMVO, "0001243d", "017c", "000017cd", 0, 0},
	{OpMVO, "0005432c", "071d", "000071dc", 0, 0},
	{OpMVO, "0000123d", "176d", "000176dd", 0, 0},
	{OpMVO, "0000512c", "01068c", "001068cc", 0, 0},
	{OpMVO, "002c", "2c", "02cc", 0, 0},
	{OpMVO, "004c", "4c", "04cc", 0, 0},
	{OpMVO, "008c", "8c", "08cc", 0, 0},
	{OpMVO, "512c", "00068c", "68cc", 0, 0},
	{OpZAP, "0001234c", "025c", "0000025c", 2, 0},
	{OpZAP, "0001243d", "017c", "0000017c", 2, 0},
	{OpZAP, "0005432c", "071d", "0000071d", 1, 0},
	{OpZAP, "0000123d", "176d", "0000176d", 1, 0},
	{OpZAP, "0000512c", "01068c", "0001068c", 2, 0},
	{OpZAP, "002c", "2c", "002c", 2, 0},
	{OpZAP, "004c", "4c", "004c", 2, 0},
	{OpZAP, "008c", "8c", "008c", 2, 0},
	{OpZAP, "512c", "01068c", "068c", 3, 10},
	{OpZAP, "512c", "00068c", "068c", 2, 0},
	{OpCP, "0c", "000d", "0c", 0, 0},
	{OpCP, "1c", "5c", "1c", 1, 0},
	{OpCP, "9c", "9c", "9c", 0, 0},
	{OpCP, "9c", "9d", "9c", 2, 0},
	{OpCP, "017c", "4d", "017c", 2, 0},
	{OpCP, "1c", "034d", "1c", 2, 0},
	{OpCP, "027c", "000000235d", "027c", 2, 0},
	{OpCP, "5c", "000000235d", "5c", 2, 0},
	{OpCP, "12345c", "54321c", "12345c", 1, 0},
	{OpED, "20204021", "a0", "fa204021", 0, 7},
	{OpED, "ee2020202120", "00023c", "eeeeeeeef2f3", 2, 0},
	{OpED, "ee2020202120", "0c1c012c", "eeeef1eef1f2", 2, 0},
	{OpED, "ee2020202120", "0d1d012d", "eeeef1f0f1f2", 1, 0},
	{OpED, "ee202022202120", "0c1c012e", "eeeef1eeeef1f2", 2, 0},
	{OpED, "ee202020", "00b0", "eeeeee20", 0, 7},
	{OpED, "ee202020", "00c0", "eeeeee20", 0, 7},
	{OpED, "ee212020", "000f", "eeeef0f0", 0, 0},
	{OpED, "ee2020202020202020202020202020", "013b026c00129c789a", "eeeef1f3f0f2f6eeeef1f2f9f7f8f9", 2, 0},
	{OpED, "402020402120", "X1", "40f4f040f2f0", 1, 0},
	{OpAP, "3c", "5c", "8c", 2, 0},
}

// Run group of decimal test cases.
func TestCycleDecimalTest(t *testing.T) {
	setup()

	for i, test := range cases {
		var res [256]byte
		addr := uint32(0x1000)
		cpuState.regs[10] = addr
		arg, _ := hex.DecodeString(test.i1)
		for i, v := range arg {
			setMemByte(addr+uint32(i), uint32(v))
		}
		l1 := len(arg)
		l2 := 0
		// Overlap data fields
		if test.i2[0] == 'X' {
			o := test.i2[1] - '0'
			cpuState.regs[12] = addr
			cpuState.regs[10] = addr + uint32(o)
		} else {
			addr2 := uint32(0x2000)
			cpuState.regs[12] = addr2
			arg, _ := hex.DecodeString(test.i2)
			for i, v := range arg {
				setMemByte(addr2+uint32(i), uint32(v))
			}
			l2 = len(arg)
		}
		inst := (uint32(test.op) << 24) | 0xa000
		if test.op == OpED {
			inst |= uint32(l1-1) << 16
		} else {
			inst |= uint32(l1-1) << 20
			inst |= uint32(l2-1) << 16
		}
		memory.SetMemory(0x400, inst)
		memory.SetMemory(0x404, 0xc0000000)
		memory.SetMemory(0x800, 0)
		memory.SetMemory(0x28, 0)
		memory.SetMemory(0x2c, 0)
		cpuState.testInst(0x4)
		addr = 0x1000
		var data uint8
		// Convert result to hex string for compare.
		for j := range len(test.out) {
			if (j & 1) != 0 {
				addr++
				res[j] = hexDigits[data&0xf]
			} else {
				data = getMemByte(addr)
				res[j] = hexDigits[(data>>4)&0xf]
			}
		}
		result := string(res[:len(test.out)])

		if test.ex != 0 {
			if !trapFlag {
				t.Errorf("Test %d did not trap", i)
			}
			v := memory.GetMemory(0x28) & 0xffff
			if v != uint32(test.ex) {
				t.Errorf("Test %d did not trap correctly got: %04x expected: %04x", i, v, test.ex)
			}
		} else {
			if result != test.out {
				t.Errorf("Test %d did get correct result got: %s expected: %s", i, result, test.out)
			}
			if cpuState.cc != test.cc {
				t.Errorf("Test %d did not get correct CC got: %x expected: %x", i, cpuState.cc, test.cc)
			}
			if trapFlag {
				t.Errorf("Test %d traped", i)
			}
			v := memory.GetMemory(0x28) & 0xffff
			if v != uint32(test.ex) {
				t.Errorf("Test %d did reported incorrect trap got: %04x expected: %04x", i, v, test.ex)
			}
		}
	}
}

// Test floating point store double.
func TestCycleSTD(t *testing.T) {
	setup()

	setFloatShort(0, 0x12345678)
	setFloatShort(1, 0xaabbccdd)
	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x300
	memory.SetMemory(0x400, 0x60012100) // STD 0,100(1,2)
	memory.SetMemory(0x404, 0x0)
	cpuState.testInst(0)
	v := memory.GetMemory(0x500)
	mv := uint32(0x12345678)
	if v != mv {
		t.Errorf("STD Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x504)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("STD Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Test floating point load double.
func TestCycleLD(t *testing.T) {
	setup()

	memory.SetMemory(0x100, 0x12345678)
	memory.SetMemory(0x104, 0xaabbccdd)
	memory.SetMemory(0x400, 0x68000100) //  LD 0,100(0,0)
	memory.SetMemory(0x404, 0x0)
	setFloatShort(0, 0xffffffff)
	setFloatShort(1, 0xffffffff)
	cpuState.testInst(0)
	v := getFloatShort(0)
	mv := uint32(0x12345678)
	if v != mv {
		t.Errorf("LD Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(1)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("LD Register 2 not correct got: %08x wanted: %08x", v, mv)
	}

	memory.SetMemory(0x100, 0x44000000)
	memory.SetMemory(0x104, 0xaabbccdd)
	memory.SetMemory(0x400, 0x68000100) //  LD 0,100(0,0)
	setFloatShort(0, 0xffffffff)
	setFloatShort(1, 0xffffffff)
	cpuState.testInst(0)
	v = getFloatShort(0)
	mv = uint32(0x44000000) // Stays unnormalized
	if v != mv {
		t.Errorf("LD Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(1)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("LD Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Load complement LCDR - LCDR 2,4.
func TestCycleLCDR(t *testing.T) {
	setup()

	memory.SetMemory(0x400, 0x23240000) // LCDR 2,4
	memory.SetMemory(0x404, 0x0)

	// Test positive number
	setFloatShort(4, 0x12345678)
	setFloatShort(5, 0xaabbccdd)
	cpuState.testInst(0)
	v := getFloatShort(2)
	mv := uint32(0x92345678)
	if v != mv {
		t.Errorf("LCDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("LCDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("LCDR CC not set correctly got: %d wanted: %d", cpuState.cc, 1)
	}

	// Test negative number
	setFloatShort(4, 0x92345678)
	setFloatShort(5, 0xaabbccdd)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("LCDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("LCDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("LCDR CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	// Test zero
	setFloatShort(4, 0x00000000)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x80000000)
	if v != mv {
		t.Errorf("LCDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LCDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("LCDR CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}

	// Test overflow
	setFloatShort(4, 0x80000000)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LCDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LCDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("LCDR CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}
}

// Load Positive LPDR - LPDR 3,4.
func TestCycleLPDR(t *testing.T) {
	setup()

	memory.SetMemory(0x400, 0x20240000) // LPDR 2,4
	memory.SetMemory(0x404, 0x0)

	setFloatShort(4, 0xffffffff)
	setFloatShort(5, 0xffffffff)
	cpuState.testInst(0)
	v := getFloatShort(2)
	mv := uint32(0x7fffffff)
	if v != mv {
		t.Errorf("LPDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0xffffffff)
	if v != mv {
		t.Errorf("LPDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("LPDR CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	// Test positive
	setFloatShort(4, 0x12345678)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("LPDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LPDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("LPDR CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	// Test zero
	setFloatShort(4, 0x00000000)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LPDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LPDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("LPDR CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}

	// Test negative number
	setFloatShort(4, 0x92345678)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x12345678)
	if v != mv {
		t.Errorf("LPDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LPDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("LPDR CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	// Test overflow
	setFloatShort(4, 0x80000000)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LPDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LPDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("LPDR CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}
}

// Load negative LNDR - LNDR 3,4.
func TestCycleLNDR(t *testing.T) {
	setup()

	memory.SetMemory(0x400, 0x21240000) // LNDR 2,4
	memory.SetMemory(0x404, 0x0)
	setFloatShort(4, 0xffffffff)
	setFloatShort(5, 0xffffffff)

	cpuState.testInst(0)
	v := getFloatShort(2)
	mv := uint32(0xffffffff)
	if v != mv {
		t.Errorf("LNDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0xffffffff)
	if v != mv {
		t.Errorf("LNDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("LNDR CC not set correctly got: %d wanted: %d", cpuState.cc, 1)
	}

	// Test positive
	setFloatShort(4, 0x12345678)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x92345678)
	if v != mv {
		t.Errorf("LNDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LNDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("LNDR CC not set correctly got: %d wanted: %d", cpuState.cc, 1)
	}

	// Test zero
	setFloatShort(4, 0x00000000)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x80000000)
	if v != mv {
		t.Errorf("LNDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LNDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("LNDR CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}

	// Test negative number
	setFloatShort(4, 0x92345678)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x92345678)
	if v != mv {
		t.Errorf("LNDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LNDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 1 {
		t.Errorf("LNDR CC not set correctly got: %d wanted: %d", cpuState.cc, 1)
	}

	// Test overflow
	setFloatShort(4, 0x80000000)
	setFloatShort(5, 0x00000000)
	cpuState.testInst(0)
	v = getFloatShort(2)
	mv = uint32(0x80000000)
	if v != mv {
		t.Errorf("LNDR Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(3)
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("LNDR Register 3 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("LNDR CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}
}

// Test compare double.
func TestCycleCD(t *testing.T) {
	setup()

	// Check results
	setFloatShort(0, 0x43000000)
	setFloatShort(1, 0x00000000)
	memory.SetMemory(0x100, 0x32123456)
	memory.SetMemory(0x104, 0x789ABCDE)
	memory.SetMemory(0x400, 0x69000100) // CD 0,100(0,0)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CD CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}

	setFloatShort(0, 0x12345678)
	setFloatShort(1, 0xaabbccdd)
	memory.SetMemory(0x100, 0x44000000)
	memory.SetMemory(0x104, 0xaabbccdd)
	memory.SetMemory(0x400, 0x69000100) // CD 0,100(0,0)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CD CC not set correctly got: %d wanted: %d", cpuState.cc, 1)
	}

	setFloatShort(0, 0x43082100)
	setFloatShort(1, 0xaabbccdd)
	memory.SetMemory(0x100, 0x43082100)
	memory.SetMemory(0x104, 0xaabbccdd)
	memory.SetMemory(0x400, 0x69000100) // CD 0,100(0,0)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CD CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}
}

// Half instruct rand.
func TestCycleHD(t *testing.T) {
	setup()
	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f = math.Ldexp(f, scale)
		err := floatToFpreg(2, f)
		if !err {
			t.Errorf("HDR Unable to set register to %f", f)
		}
		mb := f / 2.0
		memory.SetMemory(0x400, 0x24020000) // HDR 0,2
		cpuState.testInst(0)
		v := cnvtLongFloat(0)
		ratio := math.Abs((v - mb) / mb)
		if ratio > 0.000001 {
			t.Errorf("HDR difference too large got: %f expected: %f", v, mb)
		}
	}
}

// Add double.
func TestCycleAD(t *testing.T) {
	setup()

	setFloatShort(6, 0x43082100)
	setFloatShort(7, 0x00000000)
	memory.SetMemory(0x2000, 0x41123456)
	memory.SetMemory(0x2004, 0x00000000)
	cpuState.regs[13] = 0x00002000
	memory.SetMemory(0x400, 0x6a60d000) // AD 6,0(0, 13)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)

	v := getFloatShort(6)
	mv := uint32(0x42833345)
	if v != mv {
		t.Errorf("AD Register 6 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(7)
	mv = uint32(0x60000000)
	if v != mv {
		t.Errorf("AD Register 7 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AD CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f1 := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f1 = math.Ldexp(f1, scale)
		f2 := rnum.NormFloat64()
		scale = rnum.Intn(100) - 50
		f2 = math.Ldexp(f2, scale)
		err := floatToFpreg(0, f1)
		if !err {
			t.Errorf("Unable to set register to %f", f1)
		}
		err = floatToFpreg(2, f2)
		if !err {
			t.Errorf("Unable to set register to %f", f2)
		}
		mb := f1 + f2
		memory.SetMemory(0x400, 0x2a020000) // ADR 0,2
		cpuState.testInst(0)
		v := cnvtLongFloat(0)
		ratio := math.Abs((v - mb) / mb)
		if ratio > 0.000001 {
			t.Errorf("AD difference too large got: %f expected: %f", v, mb)
		}
		cc := uint8(0)
		if mb != 0.0 {
			if mb < 0.0 {
				cc = 1
			} else {
				cc = 2
			}
		}
		if cpuState.cc != cc {
			t.Errorf("AD CC not set correctly got: %d wanted: %d", cpuState.cc, cc)
		}
	}
}

// Subtract double.
func TestCycleSD(t *testing.T) {
	setup()

	setFloatShort(6, 0x43082100)
	setFloatShort(7, 0x00000000)
	memory.SetMemory(0x2000, 0x41123456)
	memory.SetMemory(0x2004, 0x00000000)
	cpuState.regs[13] = 0x00002000
	memory.SetMemory(0x400, 0x6b60d000) // SD 6,0(0, 13)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)

	v := getFloatShort(6)
	mv := uint32(0x4280ECBA)
	if v != mv {
		t.Errorf("SD Register 6 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(7)
	mv = uint32(0xA0000000)
	if v != mv {
		t.Errorf("SD Register 7 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SD CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f1 := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f1 = math.Ldexp(f1, scale)
		f2 := rnum.NormFloat64()
		scale = rnum.Intn(100) - 50
		f2 = math.Ldexp(f2, scale)
		err := floatToFpreg(0, f1)
		if !err {
			continue
		}
		err = floatToFpreg(2, f2)
		if !err {
			continue
		}
		mb := f1 - f2
		memory.SetMemory(0x400, 0x2b020000) // SDR 0,2
		cpuState.testInst(0)
		v := cnvtLongFloat(0)
		ratio := math.Abs((v - mb) / mb)
		if ratio > 0.000001 {
			t.Errorf("SD difference too large got: %f expected: %f", v, mb)
		}
		cc := uint8(0)
		if mb != 0.0 {
			if mb < 0.0 {
				cc = 1
			} else {
				cc = 2
			}
		}
		if cpuState.cc != cc {
			t.Errorf("SD CC not set correctly got: %d wanted: %d", cpuState.cc, cc)
		}
	}
}

// Multiply double.
func TestCyclMD(t *testing.T) {
	setup()

	setFloatShort(6, 0x43082100)
	setFloatShort(7, 0x00000000)
	memory.SetMemory(0x2000, 0x41123456)
	memory.SetMemory(0x2004, 0x00000000)
	cpuState.regs[13] = 0x00002000
	memory.SetMemory(0x400, 0x6c60d000) // MD 6,0(0, 13)
	cpuState.testInst(0)

	v := getFloatShort(6)
	mv := uint32(0x4293fb6f)
	if v != mv {
		t.Errorf("MD Register 6 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(7)
	mv = uint32(0x16000000)
	if v != mv {
		t.Errorf("MD Register 7 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("MD CC not set correctly got: %d wanted: %d", cpuState.cc, 3)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f1 := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f1 = math.Ldexp(f1, scale)
		f2 := rnum.NormFloat64()
		scale = rnum.Intn(100) - 50
		f2 = math.Ldexp(f2, scale)
		if !floatToFpreg(0, f1) {
			continue
		}
		if !floatToFpreg(2, f2) {
			continue
		}
		mb := f1 * f2
		memory.SetMemory(0x400, 0x2c020000) // MDR 0,2
		cpuState.testInst(0)
		if math.Abs(mb) < 5.4e-79 || math.Abs(mb) > 7.2e75 {
			if !trapFlag {
				t.Error("MD did not trap")
			}
		} else {
			if trapFlag {
				t.Error("MD should not have trapped")
			}
			v := cnvtLongFloat(0)
			ratio := math.Abs((v - mb) / mb)
			if ratio > 0.000001 {
				t.Errorf("MD difference too large got: %f expected: %f", v, mb)
			}
		}
	}
}

// Divide double.
func TestCyclDD(t *testing.T) {
	setup()

	setFloatShort(6, 0x43082100)
	setFloatShort(7, 0x00000000)
	memory.SetMemory(0x2000, 0x41123456)
	memory.SetMemory(0x2004, 0x00000000)
	cpuState.regs[13] = 0x00002000
	memory.SetMemory(0x400, 0x6d60d000) // DD 6,0(0, 13)
	// 	cpuState.testInst(0,20)
	// 	ASSERT_EQUAL_X(0x42725012, get_fpreg_s(6))
	// 	ASSERT_EQUAL_X(0xf5527d99, get_fpreg_s(7))
	cpuState.testInst(0)

	v := getFloatShort(6)
	mv := uint32(0x42725012)
	if v != mv {
		t.Errorf("DD Register 6 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(7)
	mv = uint32(0xf5527d99)
	if v != mv {
		t.Errorf("DD Register 7 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("DD CC not set correctly got: %d wanted: %d", cpuState.cc, 3)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f1 := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f1 = math.Ldexp(f1, scale)
		f2 := rnum.NormFloat64()
		scale = rnum.Intn(100) - 50
		f2 = math.Ldexp(f2, scale)
		if !floatToFpreg(0, f1) {
			continue
		}
		if !floatToFpreg(2, f2) {
			continue
		}
		mb := f1 / f2
		memory.SetMemory(0x400, 0x2d020000) // DDR 0,2
		cpuState.testInst(0)
		if math.Abs(mb) < 5.4e-79 || math.Abs(mb) > 7.2e75 {
			if !trapFlag {
				t.Error("DD did not trap")
			}
		} else {
			if trapFlag {
				t.Error("DD should not have trapped")
			}
			v := cnvtLongFloat(0)
			ratio := math.Abs((v - mb) / mb)
			if ratio > 0.000001 {
				t.Errorf("DD difference too large got: %f expected: %f", v, mb)
			}
		}
	}
}

// Add double unnormalized.
func TestCycleAW(t *testing.T) {
	setup()

	setFloatShort(6, 0x43082100)
	setFloatShort(7, 0x00000000)
	memory.SetMemory(0x2000, 0x41123456)
	memory.SetMemory(0x2004, 0x00000000)
	cpuState.regs[13] = 0x00002000
	memory.SetMemory(0x400, 0x6e60d000) // AU 6,0(0, 13)
	cpuState.testInst(0)
	v := getFloatShort(6)
	mv := uint32(0x43083334)
	if v != mv {
		t.Errorf("AW Register 6 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(7)
	mv = uint32(0x56000000)
	if v != mv {
		t.Errorf("AW Register 7 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AW CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}
}

// Subtract double unnormalized.
func TestCycleSW(t *testing.T) {
	setup()

	setFloatShort(6, 0x43082100)
	setFloatShort(7, 0x00000000)
	memory.SetMemory(0x2000, 0x41123456)
	memory.SetMemory(0x2004, 0x00000000)
	cpuState.regs[13] = 0x00002000
	memory.SetMemory(0x400, 0x6f60d000) // SU 6,0(0, 13)
	cpuState.testInst(0)
	v := getFloatShort(6)
	mv := uint32(0x43080ecb)
	if v != mv {
		t.Errorf("SW Register 6 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(7)
	mv = uint32(0xaa000000)
	if v != mv {
		t.Errorf("SW Register 7 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SW CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}
}

// Store float point
func TestCycleSTE(t *testing.T) {
	setup()

	setFloatShort(0, 0x12345678)
	setFloatShort(1, 0xaabbccdd)
	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x300
	memory.SetMemory(0x404, 0x11223344)
	memory.SetMemory(0x400, 0x70012100) // STE 0,100(1,2)
	memory.SetMemory(0x500, 0xaabbccdd)
	memory.SetMemory(0x505, 0x11223344)
	cpuState.testInst(0)
	v := memory.GetMemory(0x500)
	mv := uint32(0x12345678)
	if v != mv {
		t.Errorf("STE Memory 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = memory.GetMemory(0x504)
	mv = uint32(0x11223344)
	if v != mv {
		t.Errorf("STE Memory 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Test floating point load short.
func TestCycleLE(t *testing.T) {
	setup()

	setFloatShort(0, 0x12345678)
	setFloatShort(1, 0xaabbccdd)
	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x300
	memory.SetMemory(0x500, 0x11223344)
	memory.SetMemory(0x505, 0x11223344)
	memory.SetMemory(0x400, 0x78012100) // LE 0,100(1,2)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	v := getFloatShort(0)
	mv := uint32(0x11223344)
	if v != mv {
		t.Errorf("LE Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(1)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("LE Register 2 not correct got: %08x wanted: %08x", v, mv)
	}

	memory.SetMemory(0x100, 0x44000000)
	memory.SetMemory(0x104, 0xaabbccdd)
	memory.SetMemory(0x400, 0x78000100) //  LE 0,100(0,0)
	memory.SetMemory(0x404, 0x00000000)
	setFloatShort(0, 0xffffffff)
	setFloatShort(1, 0xffffffff)
	cpuState.testInst(0)
	v = getFloatShort(0)
	mv = uint32(0x44000000) // Stays unnormalized
	if v != mv {
		t.Errorf("LE Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(1)
	mv = uint32(0xffffffff)
	if v != mv {
		t.Errorf("LE Register 2 not correct got: %08x wanted: %08x", v, mv)
	}
}

// Test compare short.
func TestCycleCE(t *testing.T) {
	setup()

	// Check results
	setFloatShort(0, 0x12345678)
	setFloatShort(1, 0xaabbccdd)
	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x300
	memory.SetMemory(0x500, 0x11223344)
	memory.SetMemory(0x400, 0x79012100) // CE 0,100(1,2)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("CE CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	setFloatShort(0, 0x34100000)
	setFloatShort(1, 0x00000000)
	memory.SetMemory(0x100, 0x32123456)
	memory.SetMemory(0x104, 0x789ABCDE)
	memory.SetMemory(0x400, 0x79000100) // CE 0,100(0,0)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 2 {
		t.Errorf("CE CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	setFloatShort(0, 0x12345678)
	setFloatShort(1, 0xaabbccdd)
	memory.SetMemory(0x100, 0x14100000)
	memory.SetMemory(0x104, 0xaabbccdd)
	memory.SetMemory(0x400, 0x79000100) // CE 0,100(0,0)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 1 {
		t.Errorf("CE CC not set correctly got: %d wanted: %d", cpuState.cc, 1)
	}

	setFloatShort(0, 0x43082100)
	setFloatShort(1, 0xaabbccdd)
	memory.SetMemory(0x100, 0x43082100)
	memory.SetMemory(0x104, 0xaabbccdd)
	memory.SetMemory(0x400, 0x79000100) // CE 0,100(0,0)
	memory.SetMemory(0x404, 0x00000000)
	cpuState.testInst(0)
	if cpuState.cc != 0 {
		t.Errorf("CE CC not set correctly got: %d wanted: %d", cpuState.cc, 0)
	}
}

// Half instruct rand.
func TestCycleHE(t *testing.T) {
	setup()
	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f = math.Ldexp(f, scale)
		low := rnum.Uint32()
		if !floatToFpreg(2, f) {
			continue
		}
		setFloatShort(1, low)
		mb := f / 2.0
		memory.SetMemory(0x400, 0x34020000) // HER 0,2
		cpuState.testInst(0)
		v := cnvtShortFloat(0)
		ratio := math.Abs((v - mb) / mb)
		if ratio > 0.000001 {
			t.Errorf("HER difference too large got: %f expected: %f", v, mb)
		}
		if low != getFloatShort(1) {
			t.Errorf("HER modified lower regiser got: %08x expected: %08x", getFloatShort(1), low)
		}
	}
}

// Add short.
func TestCycleAE(t *testing.T) {
	setup()

	setFloatShort(0, 0x12345678)
	setFloatShort(1, 0xaabbccdd)
	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x300
	memory.SetMemory(0x500, 0x11223344)
	memory.SetMemory(0x400, 0x7a012100) // AE 0,100(1,2)
	cpuState.testInst(0)

	v := getFloatShort(0)
	mv := uint32(0x123679ac)
	if v != mv {
		t.Errorf("AE Register 0 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(1)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("AE Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AE CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f1 := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f1 = math.Ldexp(f1, scale)
		f2 := rnum.NormFloat64()
		scale = rnum.Intn(100) - 50
		f2 = math.Ldexp(f2, scale)
		low := rnum.Uint32()
		if floatToFpreg(0, f1) {
			continue
		}
		if floatToFpreg(2, f2) {
			continue
		}
		mb := f1 + f2
		setFloatShort(1, low)
		setFloatShort(3, ^low)
		memory.SetMemory(0x400, 0x3a020000) // AER 0,2
		cpuState.testInst(0)
		v := cnvtShortFloat(0)
		ratio := math.Abs((v - mb) / mb)
		if ratio > 0.000001 {
			t.Errorf("AE difference too large got: %f expected: %f", v, mb)
		}
		cc := uint8(0)
		if mb != 0.0 {
			if mb < 0.0 {
				cc = 1
			} else {
				cc = 2
			}
		}
		if cpuState.cc != cc {
			t.Errorf("AE CC not set correctly got: %d wanted: %d", cpuState.cc, cc)
		}
		if low != getFloatShort(1) {
			t.Errorf("AE modified lower regiser got: %08x expected: %08x", getFloatShort(1), low)
		}
	}
}

// Subtract short.
func TestCycleSE(t *testing.T) {
	setup()

	setFloatShort(0, 0x12345678)
	setFloatShort(1, 0xaabbccdd)
	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x300
	memory.SetMemory(0x500, 0x11223344)
	memory.SetMemory(0x400, 0x7b012100) // SE 0,100(1,2)
	cpuState.testInst(0)
	v := getFloatShort(0)
	mv := uint32(0x12323343)
	if v != mv {
		t.Errorf("SE Register 0 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(1)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("SE Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SE CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f1 := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f1 = math.Ldexp(f1, scale)
		f2 := rnum.NormFloat64()
		scale = rnum.Intn(100) - 50
		f2 = math.Ldexp(f2, scale)
		low := rnum.Uint32()
		if floatToFpreg(0, f1) {
			continue
		}
		if floatToFpreg(2, f2) {
			continue
		}
		mb := f1 - f2
		setFloatShort(1, low)
		setFloatShort(3, ^low)
		memory.SetMemory(0x400, 0x3b020000) // SER 0,2
		cpuState.testInst(0)
		v := cnvtShortFloat(0)
		ratio := math.Abs((v - mb) / mb)
		if ratio > 0.000001 {
			t.Errorf("SE difference too large got: %f expected: %f", v, mb)
		}
		cc := uint8(0)
		if mb != 0.0 {
			if mb < 0.0 {
				cc = 1
			} else {
				cc = 2
			}
		}
		if cpuState.cc != cc {
			t.Errorf("SE CC not set correctly got: %d wanted: %d", cpuState.cc, cc)
		}
		if low != getFloatShort(1) {
			t.Errorf("SE modified lower regiser got: %08x expected: %08x", getFloatShort(1), low)
		}
	}
}

// Multiply short.
func TestCyclME(t *testing.T) {
	setup()

	setFloatShort(0, 0x43082100)
	setFloatShort(1, 0xaabbccdd)
	memory.SetMemory(0x500, 0x41123456)
	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x300
	memory.SetMemory(0x400, 0x7c012100) // ME 0,100(1,2)
	cpuState.testInst(0)

	v := getFloatLong(0)
	mv := uint64(0x4293fb6f16000000)
	if v != mv {
		t.Errorf("ME Register 0 not correct got: %016x wanted: %016x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("ME CC not set correctly got: %d wanted: %d", cpuState.cc, 3)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f1 := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f1 = math.Ldexp(f1, scale)
		f2 := rnum.NormFloat64()
		scale = rnum.Intn(100) - 50
		f2 = math.Ldexp(f2, scale)
		low := rnum.Uint32()
		if floatToFpreg(0, f1) {
			continue
		}
		if floatToFpreg(2, f2) {
			continue
		}
		setFloatShort(1, low)
		setFloatShort(3, ^low)
		mb := f1 * f2
		memory.SetMemory(0x400, 0x3c020000) // MER 0,2
		cpuState.testInst(0)
		if math.Abs(mb) < 5.4e-79 || math.Abs(mb) > 7.2e75 {
			if !trapFlag {
				t.Error("ME did not trap")
			}
		} else {
			if trapFlag {
				t.Error("ME should not have trapped")
			}
			v := cnvtLongFloat(0)
			ratio := math.Abs((v - mb) / mb)
			if ratio > 0.000001 {
				t.Errorf("ME difference too large got: %f expected: %f", v, mb)
			}
		}
	}
}

// Divide short.
func TestCyclDE(t *testing.T) {
	setup()

	setFloatShort(0, 0x43082100)
	setFloatShort(1, 0xaabbccdd)
	memory.SetMemory(0x500, 0x41123456)
	cpuState.regs[1] = 0x100
	cpuState.regs[2] = 0x300
	memory.SetMemory(0x400, 0x7d012100) // DE 0,100(1,2)
	// 	cpuState.testInst(0,20)
	// 	ASSERT_EQUAL_X(0x42725012, get_fpreg_s(0))
	cpuState.testInst(0)

	v := getFloatShort(0)
	mv := uint32(0x42725012)
	if v != mv {
		t.Errorf("DE Register 0 not correct got: %08x wanted: %08x", v, mv)
	}
	v = getFloatShort(1)
	mv = uint32(0xaabbccdd)
	if v != mv {
		t.Errorf("DE Register 1 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("DE CC not set correctly got: %d wanted: %d", cpuState.cc, 3)
	}

	rnum := rand.New(rand.NewSource(125))
	for range testCycles {
		f1 := rnum.NormFloat64()
		scale := rnum.Intn(100) - 50
		f1 = math.Ldexp(f1, scale)
		f2 := rnum.NormFloat64()
		scale = rnum.Intn(100) - 50
		f2 = math.Ldexp(f2, scale)
		low := rnum.Uint32()
		if floatToFpreg(0, f1) {
			continue
		}
		if floatToFpreg(2, f2) {
			continue
		}
		mb := f1 / f2
		setFloatShort(1, low)
		setFloatShort(3, ^low)
		memory.SetMemory(0x400, 0x3d020000) // DER 0,2
		cpuState.testInst(0)
		v := cnvtShortFloat(0)
		ratio := math.Abs((v - mb) / mb)
		if ratio > 0.000001 {
			t.Errorf("DE difference too large got: %f expected: %f", v, mb)
		}
		cc := uint8(0)
		if mb != 0.0 {
			if mb < 0.0 {
				cc = 1
			} else {
				cc = 2
			}
		}
		if cpuState.cc != cc {
			t.Errorf("DE CC not set correctly got: %d wanted: %d", cpuState.cc, cc)
		}
		if low != getFloatShort(1) {
			t.Errorf("DE modified lower regiser got: %08x expected: %08x", getFloatShort(1), low)
		}
	}
}

// Add short unnormalized.
func TestCycleAU(t *testing.T) {
	setup()
	// Princ Ops 153

	setFloatShort(6, 0x43082100)
	setFloatShort(7, 0x00000000)
	memory.SetMemory(0x2000, 0x41123456)
	memory.SetMemory(0x2004, 0x00000000)
	cpuState.regs[13] = 0x00002000
	memory.SetMemory(0x400, 0x7e60d000) // AU 6,0(0, 13)
	cpuState.testInst(0)
	v := getFloatShort(6)
	mv := uint32(0x43083334)
	if v != mv {
		t.Errorf("AW Register 6 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("AW CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}
}

// Subtract short unnormalized.
func TestCycleSU(t *testing.T) {
	setup()

	setFloatShort(6, 0x43082100)
	setFloatShort(7, 0x00000000)
	memory.SetMemory(0x2000, 0x41123456)
	memory.SetMemory(0x2004, 0x00000000)
	cpuState.regs[13] = 0x00002000
	memory.SetMemory(0x400, 0x7f60d000) // SU 6,0(0, 13)
	cpuState.testInst(0)
	v := getFloatShort(6)
	mv := uint32(0x43080ecb)
	if v != mv {
		t.Errorf("SE Register 6 not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 2 {
		t.Errorf("SW CC not set correctly got: %d wanted: %d", cpuState.cc, 2)
	}
}
