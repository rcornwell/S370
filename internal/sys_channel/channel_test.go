package sys_channel

/*
 * S370 - Channel I/O tests.
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

import (
	"testing"

	Ev "github.com/rcornwell/S370/internal/event"
	M "github.com/rcornwell/S370/internal/memory"
)

func setup(dev_num uint16) *Test_dev {
	M.SetSize(16)
	InitializeChannels()
	AddChannel(0, TYPE_MUX, 192)
	d := &Test_dev{addr: dev_num, mask: 0xff}
	AddDevice(d, dev_num)
	_ = d.InitDev()
	for i := range 0x10 {
		d.data[i] = uint8(0xf0 + i)
	}
	d.count = 0
	d.sense = 0
	d.max = 0x10
	return d
}

/* Read byte from main memory */
func getMemByte(addr uint32) uint8 {
	v := M.GetMemory(addr)
	b := uint8((v >> (8 * (3 - (addr & 3))) & 0xff))
	return b
}

// /* Set byte into main memory */
// func set_mem_b(addr uint32, data uint8) {
// 	o := 8 * (3 - (addr & 3))
// 	m := uint32(0xff)
// 	v := M.GetMemory(addr)
// 	v &= ^(m << o)
// 	v |= uint32(data) << o
// 	M.SetMemory(addr, v)
// }

func runChannel() uint16 {
	var d uint16 = NO_DEV

	for d == NO_DEV {
		Ev.Advance(1)
		d = ChanScan(0x8000, true)
	}
	IrqPending = false
	return d
}

// Debug channel test.
func TestTestChan_a(t *testing.T) {
	InitializeChannels()
	cc := TestChan(0)
	if cc != 3 {
		t.Errorf("Test Channel on non-existing channel failed expected %d got: %d", 3, cc)
	}
	AddChannel(0, TYPE_MUX, 192)
	cc = TestChan(0)
	if cc != 0 {
		t.Errorf("Test Channel on existing channel failed expected %d got: %d", 0, cc)
	}

	cc = TestChan(0x100)
	if cc != 3 {
		t.Errorf("Test Channel on non-existing channel failed expected %d got: %d", 3, cc)
	}
	AddChannel(1, TYPE_SEL, 0)
	cc = TestChan(0x100)
	if cc != 0 {
		t.Errorf("Test Channel on existing channel failed expected %d got: %d", 0, cc)
	}
}

func TestTestIO_1(t *testing.T) {
	_ = setup(0xf)
	cc := TestIO(0x00f)
	if cc != 0 {
		t.Errorf("Test I/O expected %d got: %d", 0, cc)
	}
	cc = TestIO(0x004)
	if cc != 3 {
		t.Errorf("Test I/O expected %d got: %d", 3, cc)
	}
}

func TestStartIO_1(t *testing.T) {
	var v uint32

	td := setup(0xf)
	M.SetMemory(0x40, 0)
	M.SetMemory(0x44, 0)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	M.SetMemory(0x600, 0x55555555) // Invalidate data
	M.SetMemory(0x604, 0x55555555)
	M.SetMemory(0x608, 0x55555555)
	M.SetMemory(0x60C, 0x55555555)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O expected %d got: %d", 0, cc)
	}
	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O 1 expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O 1 CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		b := getMemByte(uint32(0x600 + i))
		if b != uint8(0xf0+i) {
			t.Errorf("Start I/O 1 Invalid data %02x expected: %02x got %02x", i, 0x0f+i, b)
		}
	}

	IrqPending = false
	M.SetMemory(0x40, 0)
	M.SetMemory(0x44, 0)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	M.SetMemory(0x600, 0xf0f1f2f3) // Validate data
	M.SetMemory(0x604, 0xf4f5f6f7)
	M.SetMemory(0x608, 0xf8f9fafb)
	M.SetMemory(0x60C, 0xfcfdfeff)

	cc = StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O 2 expected %d got: %d", 0, cc)
	}
	dev = runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O 2 expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O 2 CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O 2 CSW2 expected %08x got: %08x", 0x0c000000, v)
	}
	for i := range 0x10 {
		b := td.data[i]
		if b != uint8(0xf0+i) {
			t.Errorf("Start I/O 1 Invalid data %02x expected: %02x got %02x", i, 0x0f+i, b)
		}
	}
}

func TestStartIO_sense(t *testing.T) {
	var v uint32

	td := setup(0xf)
	M.SetMemory(0x40, 0)
	M.SetMemory(0x44, 0)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x04000600) // Set channel words
	M.SetMemory(0x504, 0x00000001)
	M.SetMemory(0x600, 0x55555555) // Invalidate data
	M.SetMemory(0x604, 0x55555555)
	M.SetMemory(0x608, 0x55555555)
	M.SetMemory(0x60C, 0x55555555)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O sense expected %d got: %d", 0, cc)
	}
	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O sense expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O sense CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O sense CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = M.GetMemory(0x600)
	if v != 0x00555555 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0x00555555, v)
	}

	IrqPending = false
	td.sense = 0xff
	M.SetMemory(0x40, 0)
	M.SetMemory(0x44, 0)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x04000600) // Set channel words
	M.SetMemory(0x504, 0x00000001)
	M.SetMemory(0x600, 0x55555555) // Invalidate data
	M.SetMemory(0x604, 0x55555555)
	M.SetMemory(0x608, 0x55555555)
	M.SetMemory(0x60C, 0x55555555)

	cc = StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O sense expected %d got: %d", 0, cc)
	}
	dev = runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O sense expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O sense CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O sense CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = M.GetMemory(0x600)
	if v != 0xff555555 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0xff555555, v)
	}
}

func TestStartIO_nop(t *testing.T) {
	var v uint32

	_ = setup(0xf)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x03000600) // Set channel words
	M.SetMemory(0x504, 0x00000001)
	M.SetMemory(0x600, 0x55555555) // Invalidate data
	M.SetMemory(0x604, 0x55555555)
	M.SetMemory(0x608, 0x55555555)
	M.SetMemory(0x60C, 0x55555555)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O nop expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O nop expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O nop CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000001 {
		t.Errorf("Start I/O sense CSW2 expected %08x got: %08x", 0x0c000001, v)
	}

	v = M.GetMemory(0x600)
	if v != 0x55555555 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0x55555555, v)
	}

	IrqPending = false
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x03000600) // Set channel words
	M.SetMemory(0x504, 0x00000000)
	M.SetMemory(0x600, 0x55555555) // Invalidate data
	M.SetMemory(0x604, 0x55555555)
	M.SetMemory(0x608, 0x55555555)
	M.SetMemory(0x60C, 0x55555555)

	cc = StartIO(0x00f)
	if cc != 1 {
		t.Errorf("Start I/O zero count expected %d got: %d", 1, cc)
	}
	v = M.GetMemory(0x40)
	if v != 0xffffffff {
		t.Errorf("Start I/O zero count CSW1 expected %08x got: %08x", 0xffffffff, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0020ffff {
		t.Errorf("Start I/O zero count CSW2 expected %08x got: %08x", 0x0020ffff, v)
	}

	v = M.GetMemory(0x600)
	if v != 0x55555555 {
		t.Errorf("Start I/O zero count CSW2 expected %08x got: %08x", 0x55555555, v)
	}
}

func TestStartIO_ce_only(t *testing.T) {
	var v uint32

	_ = setup(0xf)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x13000600) // Set channel words
	M.SetMemory(0x504, 0x00000001)
	M.SetMemory(0x600, 0x55555555) // Invalidate data
	M.SetMemory(0x604, 0x55555555)
	M.SetMemory(0x608, 0x55555555)
	M.SetMemory(0x60C, 0x55555555)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O ce only expected %d got: %d", 0, cc)
	}

	dev := runChannel()

	if dev != 0xf {
		t.Errorf("Start I/O ce only expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O ce only CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000001 {
		t.Errorf("Start I/O ce only CSW2 expected %08x got: %08x", 0x0c000001, v)
	}
	v = M.GetMemory(0x600)
	if v != 0x55555555 {
		t.Errorf("Start I/O ce only CSW2 expected %08x got: %08x", 0x55555555, v)
	}
}

func TestStartIO_cc_nop(t *testing.T) {
	var v uint32

	_ = setup(0xf)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x13000600) // Set channel words
	M.SetMemory(0x504, 0x40000001)
	M.SetMemory(0x508, 0x03000600)
	M.SetMemory(0x50c, 0x00000001)
	M.SetMemory(0x600, 0x55555555) // Invalidate data
	M.SetMemory(0x604, 0x55555555)
	M.SetMemory(0x608, 0x55555555)
	M.SetMemory(0x60c, 0x55555555)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O ce only expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O ce only expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O ce only CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000001 {
		t.Errorf("Start I/O ce only CSW2 expected %08x got: %08x", 0x0c000001, v)
	}

	v = M.GetMemory(0x600)
	if v != 0x55555555 {
		t.Errorf("Start I/O ce only CSW2 expected %08x got: %08x", 0x55555555, v)
	}
}
