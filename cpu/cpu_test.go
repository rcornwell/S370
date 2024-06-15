package cpu

/*
 * S370 CPU test cases.
 *
 * Copyright 2024, Richard Cornwell
 *                 Original test cases by Ken Shirriff
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

import (
	"math/rand"
	"testing"

	"github.com/rcornwell/S370/memory"
)

const testCycles int = 100
const HDMASK uint64 = 0xffffffff80000000
const FDMASK uint64 = 0x00000000ffffffff

//  CTEST( instruct, fp_conversion) {
// 	ASSERT_EQUAL(0, floatToFpreg(0, 0.0));
// 	ASSERT_EQUAL(0, get_fpreg_s(0));
// 	ASSERT_EQUAL(0, get_fpreg_s(1));

// 	/* From Princ Ops page 157 */
// 	ASSERT_EQUAL(0, floatToFpreg(0, 1.0));
// 	ASSERT_EQUAL_X(0x41100000, get_fpreg_s(0));
// 	ASSERT_EQUAL(0, get_fpreg_s(1));

// 	ASSERT_EQUAL(0, floatToFpreg(0, 0.5));
// 	ASSERT_EQUAL_X(0x40800000, get_fpreg_s(0));
// 	ASSERT_EQUAL(0, get_fpreg_s(1));

// 	ASSERT_EQUAL(0, floatToFpreg(0, 1.0/64.0));
// 	ASSERT_EQUAL_X(0x3f400000, get_fpreg_s(0));
// 	ASSERT_EQUAL(0, get_fpreg_s(1));

// 	ASSERT_EQUAL(0, floatToFpreg(0, -15.0));
// 	ASSERT_EQUAL_X(0xc1f00000, get_fpreg_s(0));
// 	ASSERT_EQUAL(0, get_fpreg_s(1));
// }

// CTEST(instruct, fp_32_conversion) {
// 	int   i;

// 	ASSERT_EQUAL(0, floatToFpreg(0, 0.0));

// 	set_fpreg_s(0, 0xff000000);
// 	ASSERT_EQUAL(0.0, cnvt_32_float(0));

// 	set_fpreg_s(0, 0x41100000);
// 	ASSERT_EQUAL(1.0, cnvt_32_float(0));

// 	set_fpreg_s(0, 0x40800000);
// 	ASSERT_EQUAL(0.5, cnvt_32_float(0));

// 	set_fpreg_s(0, 0x3f400000);
// 	ASSERT_EQUAL(1.0/64.0, cnvt_32_float(0));

// 	set_fpreg_s(0, 0xc1f00000);
// 	ASSERT_EQUAL(-15.0, cnvt_32_float(0));

// 	srand(1);
// 	for (i = 0; i < 20; i++) {
// 		double f = rand() / (double)(RAND_MAX);
// 		int p = (rand() / (double)(RAND_MAX) * 400) - 200;
// 		double fp, ratio;
// 		f = f * pow(2, p);
// 		if (rand() & 1) {
// 			f = -f;
// 		}
// 		(void)floatToFpreg(0, f);
// 		fp = cnvt_32_float(0);
// 		/* Compare within tolerance */
// 		ratio = fabs((fp - f) / f);
// 		ASSERT_TRUE(ratio < .000001);
// 	}
// }

// CTEST(instruct, fp_64_conversion) {
// 	int   i;
// 	ASSERT_EQUAL(0, floatToFpreg(0, 0.0));

// 	set_fpreg_s(0, 0xff000000);
// 	set_fpreg_s(1, 0);
// 	ASSERT_EQUAL(0.0, cnvt_64_float(0));

// 	set_fpreg_s(0, 0x41100000);
// 	set_fpreg_s(1, 0);
// 	ASSERT_EQUAL(1.0, cnvt_64_float(0));

// 	set_fpreg_s(0, 0x40800000);
// 	set_fpreg_s(1, 0);
// 	ASSERT_EQUAL(0.5, cnvt_64_float(0));

// 	set_fpreg_s(0, 0x3f400000);
// 	set_fpreg_s(1, 0);
// 	ASSERT_EQUAL(1.0/64.0, cnvt_64_float(0));

// 	set_fpreg_s(0, 0xc1f00000);
// 	set_fpreg_s(1, 0);
// 	ASSERT_EQUAL(-15.0, cnvt_64_float(0));

// 	srand(1);
// 	for (i = 0; i < 20; i++) {
// 		double f = rand() / (double)(RAND_MAX);
// 		int p = (rand() / (double)(RAND_MAX) * 400) - 200;
// 		double fp;
// 		f = f * pow(2, p);
// 		if (rand() & 1) {
// 			f = -f;
// 		}
// 		(void)floatToFpreg(0, f);
// 		fp = cnvt_64_float(0);
// 		ASSERT_EQUAL(f, fp);
// 	}
// }

// /* Roughly test characteristics of random number generator */
// CTEST(instruct, randfloat) {
// 	int pos = 0, neg = 0;
// 	int big = 0, small = 0;
// 	int i;

// 	srand(5);
// 	for (int i = 0; i < 100; i++) {
// 		double f = randfloat(200);
// 		if (f < 0) {
// 			neg ++;
// 		} else {
// 			pos ++;
// 		}
// 		if (fabs(f) > pow(2, 100)) {
// 			big++;
// 		} else if (fabs(f) < pow(2, -100)) {
// 			small++;
// 		}
// 	}
// 	ASSERT_TRUE(pos > 30);
// 	ASSERT_TRUE(neg > 30);
// 	ASSERT_TRUE(big > 15);
// 	ASSERT_TRUE(small > 15);

//		/* Test scaling */
//		big = 0;
//		small = 0;
//		for (i = 0; i < 100; i++) {
//			double f = randfloat(10);
//			if (f < 0) {
//				neg ++;
//			} else {
//				pos ++;
//			}
//			if (fabs(f) > pow(2, 10)) {
//				big++;
//			} else if (fabs(f) < pow(2, -10)) {
//				small++;
//			}
//		}
//		ASSERT_TRUE(big < 8);
//		ASSERT_TRUE(small < 8);
//	}

var trap_flag bool

func setup() {
	memory.SetSize(16)
	InitializeCPU()
	cpu.cc = 3
}

func (cpu *CPU) test_inst(mask uint8, steps int) {
	cpu.PC = 0x400
	cpu.pmask = mask & 0xf
	memory.SetMemory(0x68, 0)
	memory.SetMemory(0x6c, 0x800)
	trap_flag = false
	for range steps {
		_ = Cycle()
		// Stop it next opcode = 0
		w := memory.GetMemory(cpu.PC)
		if cpu.PC == 0x800 {
			trap_flag = true
		}
		if (cpu.PC & 2) == 0 {
			w >>= 16
		}
		if (w & 0xfff) == 0 {
			break
		}
	}
}

// Test LR instruction
func TestCycleLR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x18310000) // LR 3,1
	cpu.regs[1] = 0x12345678
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x12345678 {
		t.Errorf("LR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x12345678)
	}
	if cpu.regs[1] != 0x12345678 {
		t.Errorf("LR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x12345678)
	}
	if cpu.cc != 3 {
		t.Errorf("LR CC changed got: %x wanted: %x", cpu.cc, 3)
	}
}

// Test LTR instruction
func TestCycleLTR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x12340000) // LTR 3,4
	// Test negative number
	cpu.regs[4] = 0xcdef1234
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0xcdef1234 {
		t.Errorf("LTR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x12345678)
	}
	if cpu.regs[4] != 0xcdef1234 {
		t.Errorf("LTR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0xcdef1234)
	}
	if cpu.cc != 1 {
		t.Errorf("LTR CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	// Test zero
	cpu.regs[4] = 0x00000000
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x00000000 {
		t.Errorf("LTR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000000)
	}
	if cpu.regs[4] != 0x00000000 {
		t.Errorf("LTR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x00000000)
	}
	if cpu.cc != 0 {
		t.Errorf("LTR CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	// Test positve
	cpu.regs[4] = 0x12345678
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x12345678 {
		t.Errorf("LTR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x12345678)
	}
	if cpu.regs[4] != 0x12345678 {
		t.Errorf("LTR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x12345678)
	}
	if cpu.cc != 2 {
		t.Errorf("LTR CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
}

// Test LCR instruction
func TestCycleLCR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x13340000) // LCR 3,4

	// Test positve
	cpu.regs[4] = 0x00001000
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0xfffff000 {
		t.Errorf("LCR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0xfffff000)
	}
	if cpu.regs[4] != 0x00001000 {
		t.Errorf("LCR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x00001000)
	}
	if cpu.cc != 1 {
		t.Errorf("LCR CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	// Test negative number
	cpu.regs[4] = 0xffffffff
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x00000001 {
		t.Errorf("LCR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000001)
	}
	if cpu.regs[4] != 0xffffffff {
		t.Errorf("LCR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0xffffffff)
	}
	if cpu.cc != 2 {
		t.Errorf("LCR CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	// Test zero
	cpu.regs[4] = 0x00000000
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x00000000 {
		t.Errorf("LCR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000000)
	}
	if cpu.regs[4] != 0x00000000 {
		t.Errorf("LCR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x00000000)
	}
	if cpu.cc != 0 {
		t.Errorf("LCR CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	// Test overflow
	cpu.regs[4] = 0x80000000
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x80000000 {
		t.Errorf("LCR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x80000000)
	}
	if cpu.regs[4] != 0x80000000 {
		t.Errorf("LCR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x80000000)
	}
	if cpu.cc != 3 {
		t.Errorf("LCR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}
}

// Test LPR instruction
func TestCycleLPR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x10340000) // LPR 3,4

	// Test positve
	cpu.regs[4] = 0x00000001
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x00000001 {
		t.Errorf("LPR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000001)
	}
	if cpu.regs[4] != 0x00000001 {
		t.Errorf("LPR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x00000001)
	}
	if cpu.cc != 2 {
		t.Errorf("LPR CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	// Test negative number
	cpu.regs[4] = 0xffffffff
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x00000001 {
		t.Errorf("LPR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000001)
	}
	if cpu.regs[4] != 0xffffffff {
		t.Errorf("LPR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0xffffffff)
	}
	if cpu.cc != 2 {
		t.Errorf("LPR CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	// Test zero
	cpu.regs[4] = 0x00000000
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x00000000 {
		t.Errorf("LPR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000000)
	}
	if cpu.regs[4] != 0x00000000 {
		t.Errorf("LPR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x00000000)
	}
	if cpu.cc != 0 {
		t.Errorf("LPR CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	// Test overflow
	cpu.regs[4] = 0x80000000
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x80000000 {
		t.Errorf("LPR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x80000000)
	}
	if cpu.regs[4] != 0x80000000 {
		t.Errorf("LPR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x80000000)
	}
	if cpu.cc != 3 {
		t.Errorf("LPR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}
}

// Test LNR instruction
func TestCycleLNR(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x11340000) // LNR 3,4

	// Test positve
	cpu.regs[4] = 0x00000001
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0xffffffff {
		t.Errorf("LNR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0xffffffff)
	}
	if cpu.regs[4] != 0x00000001 {
		t.Errorf("LNR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x00000001)
	}
	if cpu.cc != 1 {
		t.Errorf("LNR CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	// Test negative number
	cpu.regs[4] = 0xffffffff
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0xffffffff {
		t.Errorf("LNR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0xffffffff)
	}
	if cpu.regs[4] != 0xffffffff {
		t.Errorf("LNR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0xffffffff)
	}
	if cpu.cc != 1 {
		t.Errorf("LNR CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	// Test zero
	cpu.regs[4] = 0x00000000
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x00000000 {
		t.Errorf("LNR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000000)
	}
	if cpu.regs[4] != 0x00000000 {
		t.Errorf("LNR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x00000000)
	}
	if cpu.cc != 0 {
		t.Errorf("LNR CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	// Test overflow
	cpu.regs[4] = 0x80000000
	cpu.test_inst(0, 20)
	if cpu.regs[3] != 0x80000000 {
		t.Errorf("LNR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x80000000)
	}
	if cpu.regs[4] != 0x80000000 {
		t.Errorf("LNR register 4 was incorrect got: %08x wanted: %08x", cpu.regs[4], 0x80000000)
	}
	if cpu.cc != 1 {
		t.Errorf("LNR CC not correct got: %x wanted: %x", cpu.cc, 1)
	}
}

// Test Add register
func TestCycleA(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1A120000) // AR 1,2

	// Test positve
	cpu.regs[1] = 0x12345678
	cpu.regs[2] = 0x00000005
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x1234567d {
		t.Errorf("AR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x1234567d)
	}
	if cpu.regs[2] != 0x00000005 {
		t.Errorf("AR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x00000005)
	}
	if cpu.cc != 2 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	cpu.cc = 3
	// Test negative number
	cpu.regs[1] = 0x81234567
	cpu.regs[2] = 0x00000001
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x81234568 {
		t.Errorf("AR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x81234568)
	}
	if cpu.regs[2] != 0x00000001 {
		t.Errorf("AR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x00000001)
	}
	if cpu.cc != 1 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	cpu.cc = 3
	// Test zero
	cpu.regs[1] = 0x00000002
	cpu.regs[2] = 0xfffffffe
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x00000000 {
		t.Errorf("AR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00000000)
	}
	if cpu.regs[2] != 0xfffffffe {
		t.Errorf("AR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0xfffffffe)
	}
	if cpu.cc != 0 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	// Test overflow
	cpu.regs[1] = 0x7fffffff
	cpu.regs[2] = 0x00000001
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x80000000 {
		t.Errorf("AR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x80000000)
	}
	if cpu.regs[2] != 0x00000001 {
		t.Errorf("AR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x00000001)
	}
	if cpu.cc != 3 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1A121A31) // AR 1,2; AR 3,1
	cpu.regs[1] = 0x12345678
	cpu.regs[2] = 0x00000001
	cpu.regs[3] = 0x00000010
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x12345679 {
		t.Errorf("AR 2 register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x12345679)
	}
	if cpu.regs[2] != 0x00000001 {
		t.Errorf("AR 2 register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x00000001)
	}
	if cpu.regs[3] != 0x12345689 {
		t.Errorf("AR 2 register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x12345689)
	}
	if cpu.cc != 2 {
		t.Errorf("AR CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
	cpu.cc = 3
	memory.SetMemory(0x400, 0x1a120000) // AR 1,2
	cpu.regs[1] = 0x7fffffff
	cpu.regs[2] = 0x00000001
	cpu.test_inst(8, 20)
	psw1 := memory.GetMemory(0x28)
	psw2 := memory.GetMemory(0x2c)
	if !trap_flag {
		t.Errorf("AR 3 did not trap")
	}
	if cpu.regs[1] != 0x80000000 {
		t.Errorf("AR 3 register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x80000000)
	}
	if cpu.regs[2] != 0x00000001 {
		t.Errorf("AR 3 register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x00000001)
	}
	if cpu.cc != 0 {
		t.Errorf("AR 3 CC not correct got: %x wanted: %x", cpu.cc, 0)
	}
	if psw1 != 0x00000008 {
		t.Errorf("AR 3 psw1 was incorrect got: %08x wanted: %08x", psw1, 0x00000008)
	}
	if psw2 != 0x78000402 {
		t.Errorf("AR 3 psw2 was incorrect got: %08x wanted: %08x", psw2, 0x78000402)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x5a156200) // AR 1,200(5,6)
	cpu.regs[1] = 0x12345678
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0x34567890)
	cpu.test_inst(0, 20)
	s := uint32(0x12345678) + uint32(0x34567890)
	if cpu.regs[1] != s {
		t.Errorf("A register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], s)
	}
	if cpu.cc != 2 {
		t.Errorf("A CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	// Test add with random values
	rnum := rand.New(rand.NewSource(42))

	for range testCycles {
		cpu.cc = 3
		n1 := rnum.Int31()
		n2 := rnum.Int31()
		r := int64(n1) + int64(n2)
		ur := uint64(r)
		sum := uint32(ur & FDMASK)
		cpu.regs[1] = uint32(n1)
		memory.SetMemory(0x100, uint32(n2))
		memory.SetMemory(0x400, 0x5a100100) // A 1,100(0,0)
		cpu.test_inst(0, 20)

		if r == 0 { // Zero
			if cpu.cc != 0 {
				t.Errorf("A rand not correct got: %x wanted: %x", cpu.cc, 0)
			}
		} else if r > 0 { // Positive
			if (ur & HDMASK) != 0 {
				if cpu.cc != 3 {
					t.Errorf("A rand not correct got: %x wanted: %x", cpu.cc, 3)
				}
				if cpu.regs[1] != sum {
					t.Errorf("A rand over register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], sum)
				}
				continue
			}
			if cpu.cc != 2 {
				t.Errorf("A rand not correct got: %x wanted: %x", cpu.cc, 2)
			}
		} else { // Negative
			if (ur & HDMASK) != HDMASK {
				if cpu.cc != 3 {
					t.Errorf("A rand not correct got: %x wanted: %x", cpu.cc, 3)
				}
				if cpu.regs[1] != sum {
					t.Errorf("A rand over register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], sum)
				}
				continue
			}
			if cpu.cc != 1 {
				t.Errorf("A rand not correct got: %x wanted: %x", cpu.cc, 1)
			}
		}
		if cpu.regs[1] != sum {
			t.Errorf("A rand register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], sum)
		}
	}
}

// Second test of Add Half
func TestCycleAH1(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x4a156200) // AH 1,200(5,6)
	cpu.regs[1] = 0x12345678
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000202
	memory.SetMemory(0x500, 0x34567890)
	cpu.test_inst(0, 20)
	s := uint32(0x12345678) + uint32(0x7890)
	if cpu.regs[1] != s {
		t.Errorf("AH register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], s)
	}
	if cpu.cc != 2 {
		t.Errorf("AH CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	// Sign extend
	memory.SetMemory(0x400, 0x4a156200) // AH 1,200(5,6)
	cpu.regs[1] = 0x00000001
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0xfffe1234)
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0xffffffff {
		t.Errorf("AH register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0xffffffff)
	}
	if cpu.cc != 1 {
		t.Errorf("AH CC not correct got: %x wanted: %x", cpu.cc, 1)
	}
}

// Test Add Logical
func TestCycleAL(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1e120000) // ALR 1,2
	cpu.regs[1] = 0x00000000
	cpu.regs[2] = 0x00000000
	cpu.test_inst(0, 20)

	if cpu.regs[1] != 0x00000000 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00000000)
	}
	if cpu.cc != 0 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1e120000) // ALR 1,2
	cpu.regs[1] = 0xffff0000
	cpu.regs[2] = 0x00000002
	cpu.test_inst(0, 20)

	if cpu.regs[1] != 0xffff0002 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0xffff0002)
	}
	if cpu.cc != 1 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1e120000) // ALR 1,2
	cpu.regs[1] = 0xfffffffe
	cpu.regs[2] = 0x00000002
	cpu.test_inst(0, 20)

	if cpu.regs[1] != 0x00000000 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00000000)
	}
	if cpu.cc != 2 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1e120000) // ALR 1,2
	cpu.regs[1] = 0xfffffffe
	cpu.regs[2] = 0x00000003
	cpu.test_inst(0, 20)

	if cpu.regs[1] != 0x00000001 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00000001)
	}
	if cpu.cc != 3 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	// Sign extend
	memory.SetMemory(0x400, 0x5e156200) // AL 1,200(5,6)
	cpu.regs[1] = 0x12345678
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0xf0000000)
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x02345678 {
		t.Errorf("ALR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x02345678)
	}
	if cpu.cc != 3 {
		t.Errorf("ALR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}
}

// Test subtract instruction
func TestCycleS(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1b120000) // SR 1,2
	cpu.regs[1] = 0x12345678
	cpu.regs[2] = 0x00000001
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x12345677 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x12345677)
	}
	if cpu.cc != 2 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x5b156200) // S 1,200(5,6)
	memory.SetMemory(0x500, 0x12300000)
	cpu.regs[1] = 0x12345678
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	cpu.test_inst(0, 20)

	if cpu.regs[1] != 0x00045678 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00045678)
	}
	if cpu.cc != 2 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpu.regs[1] = 0x8fffffff
	cpu.test_inst(0, 20)

	if cpu.regs[1] != 0x00000000 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00000000)
	}
	if cpu.cc != 0 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpu.regs[1] = 0xffffffff
	cpu.test_inst(0, 20)

	if cpu.regs[1] != 0x00000000 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00000000)
	}
	if cpu.cc != 0 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	memory.SetMemory(0x400, 0x1b110000) // SR 1,1
	cpu.regs[1] = 0x80000000
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x00000000 {
		t.Errorf("S register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00000000)
	}
	if cpu.cc != 0 {
		t.Errorf("S CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	// Test multiply with random values
	rnum := rand.New(rand.NewSource(42))

	for range testCycles {
		cpu.cc = 3
		n1 := rnum.Int31()
		n2 := rnum.Int31()
		r := n1 - n2
		ur := uint64(r)
		diff := uint32(n1) - uint32(n2)
		cpu.regs[1] = uint32(n1)
		memory.SetMemory(0x100, uint32(n2))
		memory.SetMemory(0x400, 0x5b100100) // S 1,100(0,0)
		cpu.test_inst(0, 20)

		if r == 0 { // Zero
			if cpu.cc != 0 {
				t.Errorf("S rand not correct got: %x wanted: %x", cpu.cc, 0)
			}
		} else if r > 0 { // Positive
			if (ur & HDMASK) != 0 {
				if cpu.cc != 3 {
					t.Errorf("S rand not correct got: %x wanted: %x", cpu.cc, 3)
				}
				if cpu.regs[1] != diff {
					t.Errorf("S rand over register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], diff)
				}
				continue
			}
			if cpu.cc != 2 {
				t.Errorf("S rand not correct got: %x wanted: %x", cpu.cc, 2)
			}
		} else { // Negative
			if (ur & HDMASK) != HDMASK {
				if cpu.cc != 3 {
					t.Errorf("S rand not correct got: %x wanted: %x", cpu.cc, 3)
				}
				if cpu.regs[1] != diff {
					t.Errorf("S rand over register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], diff)
				}
				continue
			}
			if cpu.cc != 1 {
				t.Errorf("S rand not correct got: %x wanted: %x", cpu.cc, 1)
			}
		}
		if cpu.regs[1] != diff {
			t.Errorf("S rand register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], diff)
		}
	}
}

// Test Subtract half
func TestCycleSH(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x4b156200) // SH 1,200(5,6)
	cpu.regs[1] = 0x12345678
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	cpu.test_inst(0, 20)
	s := uint32(0x12345678) - uint32(0x1230)
	if cpu.regs[1] != s {
		t.Errorf("SH register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], s)
	}
	if cpu.cc != 2 {
		t.Errorf("SH CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
}

// Test Subtract logical
func TestCycleSL(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1f120000) // SLR 1,2
	cpu.regs[1] = 0x12345678
	cpu.regs[2] = 0x12345678
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x00000000 {
		t.Errorf("SL register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x00000000)
	}
	if cpu.cc != 2 {
		t.Errorf("SL CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x5f156200) // SL 1,200(5,6)
	cpu.regs[1] = 0xffffffff
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0x11111111)
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0xeeeeeeee {
		t.Errorf("SL register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0xeeeeeeee)
	}
	if cpu.cc != 3 {
		t.Errorf("SL CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x5f156200) // SL 1,200(5,6)
	cpu.regs[1] = 0x12345678
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	memory.SetMemory(0x500, 0x23456789)
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0xeeeeeeef {
		t.Errorf("SL register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0xeeeeeeef)
	}
	if cpu.cc != 1 {
		t.Errorf("SL CC not correct got: %x wanted: %x", cpu.cc, 1)
	}
}

func TestCycleC(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x19120000) // CR 1,2
	cpu.regs[1] = 0x12345678
	cpu.regs[2] = 0x12345678
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0x12345678 {
		t.Errorf("CR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0x12345678)
	}
	if cpu.cc != 0 {
		t.Errorf("CR CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x19120000) // CR 1,2
	cpu.regs[1] = 0xfffffffe            // -2
	cpu.regs[2] = 0xfffffffd            // -3
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 0xfffffffe {
		t.Errorf("CR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0xfffffffe)
	}
	if cpu.cc != 2 {
		t.Errorf("CR CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x19120000) // CR 1,2
	cpu.regs[1] = 2
	cpu.regs[2] = 3
	cpu.test_inst(0, 20)
	if cpu.regs[1] != 2 {
		t.Errorf("CR register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 2)
	}
	if cpu.cc != 1 {
		t.Errorf("CR CC not correct got: %x wanted: %x", cpu.cc, 1)
	}
	cpu.cc = 3
	memory.SetMemory(0x400, 0x59156200) // C 1,200(5,6)
	memory.SetMemory(0x500, 0x12345678)
	cpu.regs[1] = 0xf0000000
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	cpu.test_inst(0, 20)

	if cpu.regs[1] != 0xf0000000 {
		t.Errorf("C register 1 was incorrect got: %08x wanted: %08x", cpu.regs[1], 0xf0000000)
	}
	if cpu.cc != 1 {
		t.Errorf("C CC not correct got: %x wanted: %x", cpu.cc, 1)
	}
}

func TestCycleM(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpu.regs[2] = 0
	cpu.regs[3] = 28
	cpu.regs[4] = 19
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0 {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0)
	}
	if cpu.regs[3] != (28 * 19) {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 28*16)
	}

	if cpu.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpu.regs[2] = 0
	cpu.regs[3] = 0x12345678
	cpu.regs[4] = 0x34567890
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0x3b8c7b8 {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x3b8c7b8)
	}
	if cpu.regs[3] != 0x3248e380 {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x3248e380)
	}

	if cpu.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpu.regs[2] = 0
	cpu.regs[3] = 0x7fffffff
	cpu.regs[4] = 0x7fffffff
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0x3fffffff {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x3fffffff)
	}
	if cpu.regs[3] != 0x00000001 {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000001)
	}

	if cpu.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpu.regs[2] = 0
	cpu.regs[3] = 0xfffffffc // -4
	cpu.regs[4] = 0xfffffffb // -5
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0 {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0)
	}
	if cpu.regs[3] != 20 {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 20)
	}

	if cpu.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1c240000) // MR 2,4
	cpu.regs[2] = 0
	cpu.regs[3] = 0xfffffffc // -4
	cpu.regs[4] = 0x0000000a // 10
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0xffffffff {
		t.Errorf("MR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0xffffffff)
	}
	if cpu.regs[3] != 0xffffffd8 {
		t.Errorf("MR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0xffffffd8)
	}

	if cpu.cc != 3 {
		t.Errorf("MR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x5c256200) // M 1,200(5,6)
	memory.SetMemory(0x500, 0x34567890)
	cpu.regs[2] = 0
	cpu.regs[3] = 0x12345678
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0x03b8c7b8 {
		t.Errorf("M register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x03b8c7b8)
	}
	if cpu.regs[3] != 0x3248e380 {
		t.Errorf("M register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x3248e380)
	}
	if cpu.cc != 3 {
		t.Errorf("M CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	// Test multiply with random values
	rnum := rand.New(rand.NewSource(1))
	for range testCycles {
		cpu.cc = 3
		n1 := rnum.Int31()
		n2 := rand.Int31()
		r := int64(n1) * int64(n2)
		h := uint32((uint64(r) >> 32) & uint64(memory.FMASK))
		l := uint32(uint64(r) & uint64(memory.FMASK))
		cpu.regs[2] = 0
		cpu.regs[3] = uint32(n1)
		cpu.regs[4] = uint32(n2)
		memory.SetMemory(0x400, 0x1c240000) // MR 2,4
		cpu.test_inst(0, 20)
		if cpu.regs[2] != h {
			t.Errorf("MR rand register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], h)
		}
		if cpu.regs[3] != l {
			t.Errorf("MR rand register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], l)
		}
		if cpu.cc != 3 {
			t.Errorf("MR rand not correct got: %x wanted: %x", cpu.cc, 3)
		}
	}
}

func TestCycleMH(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x4c356202) // MH 3,202(5,6)
	memory.SetMemory(0x500, 0x00000003)
	cpu.regs[2] = 0
	cpu.regs[3] = 4
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0 {
		t.Errorf("MHregister 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0)
	}
	if cpu.regs[3] != 12 {
		t.Errorf("MH register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 12)
	}

	if cpu.cc != 3 {
		t.Errorf("MH CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x4c250200) // MH 2,200(5)
	memory.SetMemory(0x500, 0xffd91111) // -39
	cpu.regs[2] = 0x00000015            // 21
	cpu.regs[3] = 0x00000005
	cpu.regs[5] = 0x00000300
	cpu.regs[6] = 0x00000200
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0xfffffccd {
		t.Errorf("MHregister 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0xfffffccd)
	}
	if cpu.regs[3] != 0x00000005 {
		t.Errorf("MH register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00000005)
	}

	if cpu.cc != 3 {
		t.Errorf("MH CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

}

func TestCycleD(t *testing.T) {
	setup()
	memory.SetMemory(0x400, 0x1d240000) // DR 2,4
	cpu.regs[2] = 0x1
	cpu.regs[3] = 0x12345678
	cpu.regs[4] = 0x00000234
	// divide R2/R3 by R4
	cpu.test_inst(0, 20)
	if cpu.regs[2] != (0x112345678 % 0x234) {
		t.Errorf("DR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x112345678%0x234)
	}
	if cpu.regs[3] != (0x112345678 / 0x234) {
		t.Errorf("DR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x112345678/0x234)
	}

	if cpu.cc != 3 {
		t.Errorf("DR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x1d240000) // DR 2,4
	cpu.regs[2] = 0x1
	cpu.regs[3] = 0x12345678
	cpu.regs[4] = 0xfffffdcc
	// divide R2/R3 by R4
	cpu.test_inst(0, 20)
	if cpu.regs[2] != (0x112345678 % 0x234) {
		t.Errorf("DR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x112345678%0x234)
	}
	if cpu.regs[3] != (((0x112345678 / 0x234) ^ FMASK) + 1) {
		t.Errorf("DR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], ((0x112345678/0x234)^FMASK)+1)
	}
	if cpu.cc != 3 {
		t.Errorf("DR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	// Divide big value
	cpu.cc = 3
	memory.SetMemory(0x400, 0x1d240000) // DR 2,4
	cpu.regs[2] = 0x00112233
	cpu.regs[3] = 0x44556677
	cpu.regs[4] = 0x12345678 // 0x1122334455667788 / 0x12345678
	// divide R2/R3 by R4
	cpu.test_inst(0, 20)
	if cpu.regs[2] != (0x11b3d5f7) {
		t.Errorf("DR register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x11b3d5f7)
	}
	if cpu.regs[3] != 0x00f0f0f0 {
		t.Errorf("DR register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x00f0f0f0)
	}

	if cpu.cc != 3 {
		t.Errorf("DR CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0x5d256200) // D 2,200(5,6)
	memory.SetMemory(0x500, 0x73456789)
	cpu.regs[2] = 0x12345678
	cpu.regs[3] = 0x9abcdef0
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200
	cpu.test_inst(0, 20)
	if cpu.regs[2] != 0x50c0186a {
		t.Errorf("D register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x50c0186a)
	}
	if cpu.regs[3] != 0x286dead6 {
		t.Errorf("D register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x286dead6)
	}
	if cpu.cc != 3 {
		t.Errorf("D CC not correct got: %x wanted: %x", cpu.cc, 3)
	}

	// Divide overflow
	cpu.cc = 3
	memory.SetMemory(0x400, 0x5d256200) // D 2,200(5,6)
	memory.SetMemory(0x500, 0x23456789)
	cpu.regs[2] = 0x12345678
	cpu.regs[3] = 0x9abcdef0
	cpu.regs[5] = 0x00000100
	cpu.regs[6] = 0x00000200

	cpu.test_inst(0x8, 20)
	if cpu.regs[2] != 0x12345678 {
		t.Errorf("D register 2 was incorrect got: %08x wanted: %08x", cpu.regs[2], 0x12345678)
	}
	if cpu.regs[3] != 0x9abcdef0 {
		t.Errorf("D register 3 was incorrect got: %08x wanted: %08x", cpu.regs[3], 0x9abcdef0)
	}
	if !trap_flag {
		t.Errorf("D over did not trap")
	}
}

func TestCycleCLM(t *testing.T) {
	setup()
	cpu.regs[1] = 0xFF00FF00
	cpu.regs[2] = 0x00FFFF00
	memory.SetMemory(0x500, 0xFFFFFFFF)
	memory.SetMemory(0x400, 0xbd1a0500) // CLM 1,b'1010', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // CLM 2,b'0110', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd130500) // CLM 1,b'0011', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 1 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	cpu.regs[1] = 0x01050102
	cpu.regs[2] = 0x00010203
	memory.SetMemory(0x500, 0x01020304)
	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd190500) // CLM 1,b'1001', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // CLM 2,b'0110', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd150500) // CLM 1,b'0101', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd230500) // CLM 2,b'0011', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("CLM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
}

func TestCycleICM(t *testing.T) {
	setup()
	cpu.regs[1] = 0x00000000
	memory.SetMemory(0x500, 0x01020304)
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
	if cpu.regs[1] != 0x00000102 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpu.regs[1], 0x00000102)
	}

	cpu.cc = 3
	cpu.regs[1] = 0x00000000
	memory.SetMemory(0x400, 0xbF190500) // ICM 1,b'1001', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
	if cpu.regs[1] != 0x01000002 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpu.regs[1], 0x01000002)
	}

	cpu.cc = 3
	cpu.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
	if cpu.regs[1] != 0xd0d00102 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpu.regs[1], 0xd0d00102)
	}

	cpu.cc = 3
	cpu.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x400, 0xbF190500) // ICM 1,b'1001', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
	if cpu.regs[1] != 0x01d0d002 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpu.regs[1], 0x01d0d002)
	}

	cpu.cc = 3
	cpu.regs[1] = 0x00000000
	memory.SetMemory(0x400, 0xbF170500) // ICM 1,b'0111', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
	if cpu.regs[1] != 0x00010203 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpu.regs[1], 0x00010203)
	}

	cpu.cc = 3
	cpu.regs[1] = 0x00000000
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 1 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 1)
	}
	if cpu.regs[1] != 0x0000F0F1 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpu.regs[1], 0x0000F0F1)
	}

	cpu.cc = 3
	cpu.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbF190500) // ICM 1,b'1001', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 1 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 1)
	}
	if cpu.regs[1] != 0xf0d0d0f1 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpu.regs[1], 0xf0d0d0f1)
	}

	cpu.cc = 3
	cpu.regs[1] = 0xd0d0d0d0
	memory.SetMemory(0x500, 0x00000000)
	memory.SetMemory(0x400, 0xbF130500) // ICM 1,b'0011', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 0)
	}
	if cpu.regs[1] != 0xD0D00000 {
		t.Errorf("ICM R1 not correct got: %x wanted: %x", cpu.regs[1], 0xD0D00000)
	}

	cpu.regs[1] = 0x01050102
	cpu.regs[2] = 0x00010203
	memory.SetMemory(0x500, 0x01020304)
	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd190500) // ICM 1,b'1001', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd260500) // ICM 2,b'0110', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd150500) // ICM 1,b'0101', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	cpu.cc = 3
	memory.SetMemory(0x400, 0xbd230500) // ICM 2,b'0011', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("ICM CC not correct got: %x wanted: %x", cpu.cc, 2)
	}
}

func TestCycleSTCM(t *testing.T) {
	setup()
	cpu.cc = 3
	cpu.regs[3] = 0xf0f1f2f3
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbe330500) // STCM 3,b'0011', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 3 {
		t.Errorf("STCM CC not correct got: %x wanted: %x", cpu.cc, 3)
	}
	v := memory.GetMemory(0x500)
	if v != 0xf2f3f2f3 {
		t.Errorf("STCM memory not correct got: %x wanted: %x", v, 0xf2f3f2f3)
	}
	if cpu.regs[3] != 0xf0f1f2f3 {
		t.Errorf("STCM R3 not correct got: %x wanted: %x", cpu.regs[3], 0xf0f1f2f3)
	}

	cpu.regs[3] = 0xf0f1f2f3
	memory.SetMemory(0x500, 0xf0f1f2f3)
	memory.SetMemory(0x400, 0xbe390500) // STCM 3,b'1001', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 3 {
		t.Errorf("STCM CC not correct got: %x wanted: %x", cpu.cc, 3)
	}
	v = memory.GetMemory(0x500)
	if v != 0xf0f3f2f3 {
		t.Errorf("STCM memory not correct got: %x wanted: %x", v, 0xf0f3f2f3)
	}
	if cpu.regs[3] != 0xf0f1f2f3 {
		t.Errorf("STCM R3 not correct got: %x wanted: %x", cpu.regs[3], 0xf0f1f2f3)
	}
	cpu.regs[1] = 0xf0f1f2f3
	memory.SetMemory(0x500, 0x00000000)
	memory.SetMemory(0x400, 0xbe3f0500) // STCM 3,b'1111', 500
	cpu.test_inst(0, 20)
	if cpu.cc != 3 {
		t.Errorf("STCM CC not correct got: %x wanted: %x", cpu.cc, 3)
	}
	v = memory.GetMemory(0x500)
	if v != 0xf0f1f2f3 {
		t.Errorf("STCM memory not correct got: %x wanted: %x", v, 0xf0f1f2f3)
	}
	if cpu.regs[1] != 0xf0f1f2f3 {
		t.Errorf("STCM R3 not correct got: %x wanted: %x", cpu.regs[1], 0xf0f1f2f3)
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
	cpu.regs[2] = 0x500
	cpu.regs[3] = 20
	cpu.regs[4] = 0x600
	cpu.regs[5] = 20
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	if cpu.regs[2] != 0x500+20 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpu.regs[2], 0x500+20)
	}

	if cpu.regs[3] != 0 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpu.regs[3], 0)
	}

	if cpu.regs[4] != 0x600+20 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpu.regs[4], 0x600+20)
	}

	if cpu.regs[5] != 0 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpu.regs[5], 0)
	}

	cpu.regs[2] = 0x500
	cpu.regs[3] = 20
	cpu.regs[4] = 0x600
	cpu.regs[5] = 0xf0000000 + 10
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpu.cc, 0)
	}

	if cpu.regs[2] != 0x500+20 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpu.regs[2], 0x500+20)
	}

	if cpu.regs[3] != 0 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpu.regs[3], 0)
	}

	if cpu.regs[4] != 0x600+10 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpu.regs[4], 0x600+10)
	}

	if cpu.regs[5] != 0xf0000000 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpu.regs[5], 0xf0000000)
	}

	cpu.regs[2] = 0x500
	cpu.regs[3] = 10
	cpu.regs[4] = 0x600
	cpu.regs[5] = 0xf0000000 + 20
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4
	cpu.test_inst(0, 20)
	if cpu.cc != 0 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	if cpu.regs[2] != 0x500+10 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpu.regs[2], 0x500+10)
	}

	if cpu.regs[3] != 0 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpu.regs[3], 0)
	}

	if cpu.regs[4] != 0x600+20 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpu.regs[4], 0x600+20)
	}

	if cpu.regs[5] != 0xf0000000 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpu.regs[5], 0xf0000000)
	}

	memory.SetMemory(0x600, 0xf0f1f2f3)
	memory.SetMemory(0x604, 0xf4f5f6f7)
	memory.SetMemory(0x608, 0xf8f9f9f9)
	memory.SetMemory(0x60c, 0xf9f9f9f9)
	memory.SetMemory(0x610, 0xf9f9f9f9)
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4

	cpu.regs[2] = 0x500
	cpu.regs[3] = 20
	cpu.regs[4] = 0x600
	cpu.regs[5] = 20
	cpu.test_inst(0, 20)
	if cpu.cc != 1 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	if cpu.regs[2] != 0x500+10 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpu.regs[2], 0x500+10)
	}

	if cpu.regs[3] != 10 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpu.regs[3], 10)
	}

	if cpu.regs[4] != 0x600+10 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpu.regs[4], 0x600+10)
	}

	if cpu.regs[5] != 10 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpu.regs[5], 10)
	}

	memory.SetMemory(0x400, 0x0f420000) // CLCL 4,2

	cpu.regs[2] = 0x500
	cpu.regs[3] = 20
	cpu.regs[4] = 0x600
	cpu.regs[5] = 20
	cpu.test_inst(0, 20)
	if cpu.cc != 2 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpu.cc, 2)
	}

	if cpu.regs[2] != 0x500+10 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpu.regs[2], 0x500+10)
	}

	if cpu.regs[3] != 10 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpu.regs[3], 10)
	}

	if cpu.regs[4] != 0x600+10 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpu.regs[4], 0x600+10)
	}

	if cpu.regs[5] != 10 {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpu.regs[5], 10)
	}

	cpu.regs[2] = 0x500
	cpu.regs[3] = 5
	cpu.regs[4] = 0x600
	cpu.regs[5] = 0xf5000000 + 20
	memory.SetMemory(0x400, 0x0f240000) // CLCL 2,4
	cpu.test_inst(0, 20)
	if cpu.cc != 1 {
		t.Errorf("CLCL CC not correct got: %x wanted: %x", cpu.cc, 1)
	}

	if cpu.regs[2] != 0x500+5 {
		t.Errorf("CLCL R2 not correct got: %x wanted: %x", cpu.regs[2], 0x500+5)
	}

	if cpu.regs[3] != 0 {
		t.Errorf("CLCL R3 not correct got: %x wanted: %x", cpu.regs[3], 0)
	}

	if cpu.regs[4] != 0x600+6 {
		t.Errorf("CLCL R4 not correct got: %x wanted: %x", cpu.regs[4], 0x600+6)
	}

	if cpu.regs[5] != 0xf500000e {
		t.Errorf("CLCL R5 not correct got: %x wanted: %x", cpu.regs[5], 0xf500000e)
	}
}
