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
	"math/rand"
	"testing"

	"github.com/rcornwell/S370/emu/memory"
)

const testCycles int = 100
const HDMASK uint64 = 0xffffffff80000000
const FDMASK uint64 = 0x00000000ffffffff

//  CTEST( instruct, fp_conversion) {
// 	ASSERT_EQUAL(0, floatToFpreg(0, 0.0))
// 	ASSERT_EQUAL(0, get_fpreg_s(0))
// 	ASSERT_EQUAL(0, get_fpreg_s(1))

// 	// From Princ Ops page 157
// 	ASSERT_EQUAL(0, floatToFpreg(0, 1.0))
// 	ASSERT_EQUAL_X(0x41100000, get_fpreg_s(0))
// 	ASSERT_EQUAL(0, get_fpreg_s(1))

// 	ASSERT_EQUAL(0, floatToFpreg(0, 0.5))
// 	ASSERT_EQUAL_X(0x40800000, get_fpreg_s(0))
// 	ASSERT_EQUAL(0, get_fpreg_s(1))

// 	ASSERT_EQUAL(0, floatToFpreg(0, 1.0/64.0))
// 	ASSERT_EQUAL_X(0x3f400000, get_fpreg_s(0))
// 	ASSERT_EQUAL(0, get_fpreg_s(1))

// 	ASSERT_EQUAL(0, floatToFpreg(0, -15.0))
// 	ASSERT_EQUAL_X(0xc1f00000, get_fpreg_s(0))
// 	ASSERT_EQUAL(0, get_fpreg_s(1))
// }

// CTEST(instruct, fp_32_conversion) {
// 	int   i

// 	ASSERT_EQUAL(0, floatToFpreg(0, 0.0))

// 	set_fpreg_s(0, 0xff000000)
// 	ASSERT_EQUAL(0.0, cnvt_32_float(0))

// 	set_fpreg_s(0, 0x41100000)
// 	ASSERT_EQUAL(1.0, cnvt_32_float(0))

// 	set_fpreg_s(0, 0x40800000)
// 	ASSERT_EQUAL(0.5, cnvt_32_float(0))

// 	set_fpreg_s(0, 0x3f400000)
// 	ASSERT_EQUAL(1.0/64.0, cnvt_32_float(0))

// 	set_fpreg_s(0, 0xc1f00000)
// 	ASSERT_EQUAL(-15.0, cnvt_32_float(0))

// 	srand(1)
// 	for (i = 0 i < 20 i++) {
// 		double f = rand() / (double)(RAND_MAX)
// 		int p = (rand() / (double)(RAND_MAX) * 400) - 200
// 		double fp, ratio
// 		f = f * pow(2, p)
// 		if (rand() & 1) {
// 			f = -f
// 		}
// 		(void)floatToFpreg(0, f)
// 		fp = cnvt_32_float(0)
// 		// Compare within tolerance
// 		ratio = fabs((fp - f) / f)
// 		ASSERT_TRUE(ratio < .000001)
// 	}
// }

// CTEST(instruct, fp_64_conversion) {
// 	int   i
// 	ASSERT_EQUAL(0, floatToFpreg(0, 0.0))

// 	set_fpreg_s(0, 0xff000000)
// 	set_fpreg_s(1, 0)
// 	ASSERT_EQUAL(0.0, cnvt_64_float(0))

// 	set_fpreg_s(0, 0x41100000)
// 	set_fpreg_s(1, 0)
// 	ASSERT_EQUAL(1.0, cnvt_64_float(0))

// 	set_fpreg_s(0, 0x40800000)
// 	set_fpreg_s(1, 0)
// 	ASSERT_EQUAL(0.5, cnvt_64_float(0))

// 	set_fpreg_s(0, 0x3f400000)
// 	set_fpreg_s(1, 0)
// 	ASSERT_EQUAL(1.0/64.0, cnvt_64_float(0))

// 	set_fpreg_s(0, 0xc1f00000)
// 	set_fpreg_s(1, 0)
// 	ASSERT_EQUAL(-15.0, cnvt_64_float(0))

// 	srand(1)
// 	for (i = 0 i < 20 i++) {
// 		double f = rand() / (double)(RAND_MAX)
// 		int p = (rand() / (double)(RAND_MAX) * 400) - 200
// 		double fp
// 		f = f * pow(2, p)
// 		if (rand() & 1) {
// 			f = -f
// 		}
// 		(void)floatToFpreg(0, f)
// 		fp = cnvt_64_float(0)
// 		ASSERT_EQUAL(f, fp)
// 	}
// }

// // Roughly test characteristics of random number generator
// CTEST(instruct, randfloat) {
// 	int pos = 0, neg = 0
// 	int big = 0, small = 0
// 	int i

// 	srand(5)
// 	for (int i = 0 i < 100 i++) {
// 		double f = randfloat(200)
// 		if (f < 0) {
// 			neg ++
// 		} else {
// 			pos ++
// 		}
// 		if (fabs(f) > pow(2, 100)) {
// 			big++
// 		} else if (fabs(f) < pow(2, -100)) {
// 			small++
// 		}
// 	}
// 	ASSERT_TRUE(pos > 30)
// 	ASSERT_TRUE(neg > 30)
// 	ASSERT_TRUE(big > 15)
// 	ASSERT_TRUE(small > 15)

//		// Test scaling
//		big = 0
//		small = 0
//		for (i = 0 i < 100 i++) {
//			double f = randfloat(10)
//			if (f < 0) {
//				neg ++
//			} else {
//				pos ++
//			}
//			if (fabs(f) > pow(2, 10)) {
//				big++
//			} else if (fabs(f) < pow(2, -10)) {
//				small++
//			}
//		}
//		ASSERT_TRUE(big < 8)
//		ASSERT_TRUE(small < 8)
//	}

var trapFlag bool

func setup() {
	memory.SetSize(64)
	InitializeCPU()
	cpuState.cc = 3
}

func (cpu *cpu) testInst(mask uint8, steps int) {
	cpu.PC = 0x400
	cpu.progMask = mask & 0xf
	memory.SetMemory(0x68, 0)
	memory.SetMemory(0x6c, 0x800)
	trapFlag = false
	for range steps {
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(8, 20)
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
	cpuState.testInst(0, 20)
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
		cpuState.testInst(0, 20)

		if r == 0 { // Zero
			if cpuState.cc != 0 {
				t.Errorf("A rand not correct got: %x wanted: %x", cpuState.cc, 0)
			}
		} else if r > 0 { // Positive
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
		} else { // Negative
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)
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
		cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)

	if cpuState.regs[1] != 0x00045678 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00045678)
	}
	if cpuState.cc != 2 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpuState.regs[1] = 0x8fffffff
	cpuState.testInst(0, 20)

	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpuState.regs[1] = 0xffffffff
	cpuState.testInst(0, 20)

	if cpuState.regs[1] != 0x00000000 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpuState.regs[1], 0x00000000)
	}
	if cpuState.cc != 0 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpuState.regs[1] = 0x80000000
	cpuState.testInst(0, 20)
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
		cpuState.testInst(0, 20)

		if r == 0 { // Zero
			if cpuState.cc != 0 {
				t.Errorf("S rand not correct got: %x wanted: %x", cpuState.cc, 0)
			}
		} else if r > 0 { // Positive
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
		} else { // Negative
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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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

		cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

	v := cpuState.regs[1]
	if v != 0x12345678 {
		t.Errorf("CL Register changed got: %08x wanted: %08x", v, 0x12345678)
	}
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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

	if cpuState.cc != 0 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	// Compare half word high
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0x1234abcd)
	cpuState.regs[3] = 0x00001235
	cpuState.testInst(0, 20)

	if cpuState.cc != 2 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Compare half word sign extended
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0x8234abcd)
	cpuState.regs[3] = 0x00001235
	cpuState.testInst(0, 20)

	if cpuState.cc != 2 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	// Compare half word low
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0x1234abcd)
	cpuState.regs[3] = 0x80001235
	cpuState.testInst(0, 20)

	if cpuState.cc != 1 {
		t.Errorf("CH CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	// Compare half lower extended
	cpuState.cc = 3
	memory.SetMemory(0x400, 0x49300100) // CH 3,100(0,0)
	memory.SetMemory(0x100, 0xfffd0000)
	cpuState.regs[3] = 0xfffffffc
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
		cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	memory.SetMemory(0x500, 0x73456789)
	cpuState.regs[2] = 0x12345678
	cpuState.regs[3] = 0x9abcdef0
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200
	cpuState.testInst(0, 20)
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
	memory.SetMemory(0x500, 0x23456789)
	cpuState.regs[2] = 0x12345678
	cpuState.regs[3] = 0x9abcdef0
	cpuState.regs[5] = 0x00000100
	cpuState.regs[6] = 0x00000200

	cpuState.testInst(0x8, 20)
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
		divisor := rand.Int31()
		q := dividend / int64(divisor)
		r := dividend % int64(divisor)

		cpuState.regs[2] = uint32(dividend >> 32)
		cpuState.regs[3] = uint32(dividend & int64(FMASK))
		memory.SetMemory(0x100, uint32(divisor))
		memory.SetMemory(0x400, 0x5d200100) // D 2,100(0,0)
		cpuState.testInst(0, 20)

		if divisor < 0 {
			r = -r
		}
		if (q & 0x7fffffff) != q {
			if !trapFlag {
				t.Errorf("D rand over did not trap")
			}
		} else {
			if trapFlag {
				t.Errorf("D rand no over did trap")
			}
		}
		if cpuState.regs[2] != uint32(r) {
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

// Test Store Word.
func TestCycleST(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x50123400) // ST 1,400(2,3)
	memory.SetMemory(0x600, 0xffffffff)
	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x100
	cpuState.regs[3] = 0x100
	// Store Half
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0, 20)

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
		cpuState.testInst(0, 20)
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
		cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)

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
	cpuState.testInst(0xa, 20)
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
	cpuState.testInst(0xa, 20)
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
	cpuState.testInst(0xa, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
			cpuState.testInst(0, 20)
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
			cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
	v = cpuState.regs[1]
	mv = uint32(0xffffffff) & uint32(0x00000000)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
	v = cpuState.regs[1]
	mv = uint32(0x00000000)
	if v != mv {
		t.Errorf("O Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("O CC not correct got: %x wanted: %x", cpuState.cc, 0)
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

	cpuState.testInst(0, 20)
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

	cpuState.testInst(0, 20)
	v = cpuState.regs[1]
	mv = 0x00000000
	if v != mv {
		t.Errorf("X Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 0 {
		t.Errorf("X CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}
}

// Shift left arithmetic single register.
func TestCycleSLA(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x12345678
	cpuState.regs[2] = 0x00000001
	memory.SetMemory(0x400, 0x8b1f2001) // SLA 1,1(2)
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
	v := cpuState.regs[1]
	mv := uint32(0x11a2b3c0)
	if v != mv {
		t.Errorf("SLL Register not correct got: %08x wanted: %08x", v, mv)
	}
	if cpuState.cc != 3 {
		t.Errorf("SLL CC not correct got: %x wanted: %x", cpuState.cc, 3)
	}
}

// Shift right logical instruction.
func TestCycleSRL(t *testing.T) {
	setup()

	cpuState.regs[1] = 0x82345678
	cpuState.regs[2] = 0x12340003       // Shift 3 bits
	memory.SetMemory(0x400, 0x881f2100) // SRL 1,100(2)
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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

// // Shift double left arithmatic.
func TestCycleSLDA(t *testing.T) {
	setup()

	cpuState.regs[2] = 0x007f0a72
	cpuState.regs[3] = 0xfedcba98
	memory.SetMemory(0x400, 0x8f2f001f) // SLDA 2,1f(0)
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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

func TestCycleCLM(t *testing.T) {
	setup()
	cpuState.regs[1] = 0xFF00FF00
	cpuState.regs[2] = 0x00FFFF00
	memory.SetMemory(0x500, 0xFFFFFFFF)
	memory.SetMemory(0x400, 0xbd1a0500) // CLM 1,b'1010', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // CLM 2,b'0110', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd130500) // CLM 1,b'0011', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 1 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 1)
	}

	cpuState.regs[1] = 0x01050102
	cpuState.regs[2] = 0x00010203
	memory.SetMemory(0x500, 0x01020304)
	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd190500) // CLM 1,b'1001', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // CLM 2,b'0110', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd150500) // CLM 1,b'0101', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 2 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd230500) // CLM 2,b'0011', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 2 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
}

func TestCycleICM(t *testing.T) {
	setup()
	cpuState.regs[1] = 0x00000000
	memory.SetMemory(0x500, 0x01020304)
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0x00000102 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0x00000102)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x00000000
	memory.SetMemory(0x400, 0xbF190500) // ICM 1,b'1001', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0x01000002 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0x01000002)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0xd0d00102 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0xd0d00102)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x400, 0xbF190500) // ICM 1,b'1001', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}
	if cpuState.regs[1] != 0x01d0d002 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpuState.regs[1], 0x01d0d002)
	}

	cpuState.cc = 3
	cpuState.regs[1] = 0x00000000
	memory.SetMemory(0x400, 0xbF170500) // ICM 1,b'0111', 500
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
	if cpuState.cc != 0 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // ICM 2,b'0110', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 0 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 0)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd150500) // ICM 1,b'0101', 500
	cpuState.testInst(0, 20)
	if cpuState.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpuState.cc, 2)
	}

	cpuState.cc = 3
	memory.SetMemory(0x400, 0xbd230500) // ICM 2,b'0011', 500
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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

	cpuState.regs[3] = 0xf0f1f2f3
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbe390500) // STCM 3,b'1001', 500
	cpuState.testInst(0, 20)
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
	cpuState.regs[1] = 0xf0f1f2f3
	memory.SetMemory(0x500, 0x00000000)
	memory.SetMemory(0x400, 0xbe3f0500) // STCM 3,b'1111', 500
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
	cpuState.testInst(0, 20)
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
