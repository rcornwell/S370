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

func setup(devNum uint16) *Test_dev {
	M.SetSize(64)
	InitializeChannels()
	AddChannel(0, TYPE_MUX, 192)
	d := &Test_dev{addr: devNum, mask: 0xff}
	AddDevice(d, devNum)
	_ = d.InitDev()
	for i := range 0x10 {
		d.data[i] = uint8(0xf0 + i)
	}
	d.max = 0x10
	return d
}

/* Read byte from main memory */
func getMemByte(addr uint32) uint8 {
	v := M.GetMemory(addr)
	b := uint8((v >> (8 * (3 - (addr & 3))) & 0xff))
	return b
}

/* write byte to main memory */
func setMemByte(addr uint32, data uint32) {
	off := 8 * (3 - (addr & 3))
	m := uint32(0xff << off)
	d := (data & 0xff) << off
	M.SetMemoryMask(addr, d, m)
}

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
func TestTestChan(t *testing.T) {
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
	cc = TestChan(0x200)
	if cc != 3 {
		t.Errorf("Test Channel on non-existing channel failed expected %d got: %d", 3, cc)
	}
	AddChannel(2, TYPE_BMUX, 0)
	cc = TestChan(0x200)
	if cc != 0 {
		t.Errorf("Test Channel on existing channel failed expected %d got: %d", 0, cc)
	}
}

func TestTestIO(t *testing.T) {
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

func TestStartIO(t *testing.T) {
	var v uint32

	td := setup(0xf)
	M.SetMemory(0x40, 0)
	M.SetMemory(0x44, 0)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	// Load memory with value not equal to rea1 CSW2d data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

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
			t.Errorf("Start I/O 2 Invalid data %02x expected: %02x got %02x", i, 0x0f+i, b)
		}
	}
}

func TestStartIOSense(t *testing.T) {
	var v uint32

	td := setup(0xf)
	M.SetMemory(0x40, 0)
	M.SetMemory(0x44, 0)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x04000600) // Set channel words
	M.SetMemory(0x504, 0x00000001)
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

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
		t.Errorf("Start I/O sense expected %08x got: %08x", 0x00555555, v)
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
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

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
		t.Errorf("Start I/O sense Data expected %08x got: %08x", 0xff555555, v)
	}
}

func TestStartIONop(t *testing.T) {
	var v uint32

	_ = setup(0xf)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x03000600) // Set channel words
	M.SetMemory(0x504, 0x00000001)
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

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
		t.Errorf("Start I/O nop CSW2 expected %08x got: %08x", 0x0c000001, v)
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
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

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
		t.Errorf("Start I/O zero count Dsts expected %08x got: %08x", 0x55555555, v)
	}
}

func TestStartIOCEOnly(t *testing.T) {
	var v uint32

	_ = setup(0xf)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x13000600) // Set channel words
	M.SetMemory(0x504, 0x00000001)
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

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
		t.Errorf("Start I/O ce only Data expected %08x got: %08x", 0x55555555, v)
	}
}

func TestStartIOCCNop(t *testing.T) {
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
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

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
		t.Errorf("Start I/O ce only Data expected %08x got: %08x", 0x55555555, v)
	}
}

func TestStartIORead(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x00000020)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Read expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Read  expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Read  CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Read CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x20 {
		vb := getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Read Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
	for i := range 0x20 {
		vb := getMemByte(uint32(0x620 + i))
		if vb != 0x55 {
			t.Errorf("Start I/O Read Data expected %02x got: %02x at: %02x", 0x55, vb, i)
		}
	}
}

func TestStartIOShortRead(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Short Read expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Short Read  expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Short Read CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c400000 {
		t.Errorf("Start I/O Short Read CSW2 expected %08x got: %08x", 0x0c400000, v)
	}

	for i := range 0x10 {
		vb := getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O  Short Read CSW2 expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
	for i := range 0x10 {
		vb := getMemByte(uint32(0x610 + i))
		if vb != 0x55 {
			t.Errorf("Start I/O Short Read Data expected %02x got: %02x at: %02x", 0x55, vb, i)
		}
	}
}

func TestStartIOShortReadSLI(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x20000010)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Short Read expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Short Read  expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Short Read CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Short Read CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O  Short Read Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
	for i := range 0x10 {
		vb := getMemByte(uint32(0x610 + i))
		if vb != 0x55 {
			t.Errorf("Start I/O Short Read Data expected %02x got: %02x at: %02x", 0x55, vb, i)
		}
	}
}

func TestStartIOWrite(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x55)
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x00000020)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := range 0x20 {
		setMemByte(uint32(i+0x600), uint32(0x10+i))
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Write expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Write expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Write CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Write CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x20 {
		vb := d.data[i]
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Write Dev Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
		vb = getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Write Mem Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
}

func TestStartIOShortWrite(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x55)
	}
	d.max = 0x10

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x00000020)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := range 0x20 {
		setMemByte(uint32(i+0x600), uint32(0x10+i))
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Short Write expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Short Write expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Short Write CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c40000f {
		t.Errorf("Start I/O Short Write CSW2 expected %08x got: %08x", 0x0c40000f, v)
	}

	for i := range 0x20 {
		mb := uint8(0x10 + i)
		if i > 0x10 {
			mb = 0x55
		}
		vb := d.data[i]
		if vb != mb {
			t.Errorf("Start I/O Write Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
		vb = getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Short Write Mem Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
}

func TestStartIOShortWriteSLI(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x55)
	}
	d.max = 0x10

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x20000020)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := range 0x20 {
		setMemByte(uint32(i+0x600), uint32(0x10+i))
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Short Write expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Short Write expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Short Write CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c00000f {
		t.Errorf("Start I/O Short Write CSW2 expected %08x got: %08x", 0x0c00000f, v)
	}

	for i := range 0x20 {
		mb := uint8(0x10 + i)
		if i > 0x10 {
			mb = 0x55
		}
		vb := d.data[i]
		if vb != mb {
			t.Errorf("Start I/O Write Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
		vb = getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Short Write Mem Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
}

func TestStartIOReadCDA(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x80000010)
	M.SetMemory(0x508, 0x01000700)
	M.SetMemory(0x50c, 0x00000010)
	// Load memory with value not equal to read data.
	for i := range uint32(0x20) {
		M.SetMemory(0x600+i, 0x55555555)
		M.SetMemory(0x700+i, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Raad CDA expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Read CDA expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O Read CDA CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Read CDA CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x20 {
		a := uint32(0x600 + i)
		if i >= 0x10 {
			vb := getMemByte(a)
			if vb != 0x55 {
				t.Errorf("Start I/O Read CDA expected %02x got: %02x at: %08x", 0x55, vb, a)
			}
			a += 0x0f0
			vb = getMemByte(a + 0x10)
			if vb != 0x55 {
				t.Errorf("Start I/O Read CDA expected %02x got: %02x at: %08x", 0x55, vb, a)
			}
		}
		vb := getMemByte(a)
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Read CDA expected %02x got: %02x at: %02x", 0x10+i, vb, a)
		}
	}
}

// Test writing with CDA enabled
func TestStartIOWriteCDA(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = 0x55
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x80000010)
	M.SetMemory(0x508, 0x00000700)
	M.SetMemory(0x50c, 0x00000010) //  CTEST2(io_test, tic_tic) {
	M.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	M.SetMemory(0x604, 0x4f5f6f7f)
	M.SetMemory(0x608, 0x8f9fafbf)
	M.SetMemory(0x60c, 0xcfdfefff)
	M.SetMemory(0x700, 0x0c1c2c3c) // Data to send
	M.SetMemory(0x704, 0x4c5c6c7c)
	M.SetMemory(0x708, 0x8c9cacbc)
	M.SetMemory(0x70c, 0xccdcecfc)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Write CDA expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Write CDA expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O Write CDA CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Write CDA CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := d.data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Write CDA Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
	for i := range 0x10 {
		vb := d.data[i+0x10]
		mb := uint8(0xc + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Write CDA Data expected %02x got: %02x at: %02x", mb, vb, i+0x10)
		}
	}
}

func TestStartIOReadCDASkip(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x10

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x90000005)
	M.SetMemory(0x508, 0x01000606)
	M.SetMemory(0x50c, 0x0000000b)
	// Load memory with value not equal to read data.
	for i := range uint32(0x20) {
		M.SetMemory(0x600+i, 0x55555555)
		M.SetMemory(0x700+i, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Raad CDA expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Read CDA expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O Read CDA CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Read CDA CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	var vb uint8
	for i := range 0x10 {
		if i >= 6 {
			vb = getMemByte(uint32(0x600 + i + 1))
			if vb != uint8(0x10+i) {
				t.Errorf("Start I/O Read Skip CDA expected %02x got: %02x at: %08x", 0x10+i, vb, 0x600+i+1)
			}

		} else {
			vb = getMemByte(uint32(0x600 + i))
			if vb != 0x55 {
				t.Errorf("Start I/O Read Skip CDA expected %02x got: %02x at: %08x", 0x55, vb, 0x600+i)
			}
		}
	}
}

func TestStartIOReadBkwd(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x10 {
		d.data[i] = uint8(0x10 + (0x0f - i))
	}
	d.max = 0x10

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x0c00060f) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Read Bkwd expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Read Bkwd expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Read Bkwd CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Read Bkwd CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Read Bkwd Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
}

func TestStartIOCChain(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = 0x55
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x40000010)
	M.SetMemory(0x508, 0x03000701)
	M.SetMemory(0x50c, 0x40000001)
	M.SetMemory(0x508, 0x04000701)
	M.SetMemory(0x50c, 0x00000001)
	M.SetMemory(0x700, 0xffffffff)
	M.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	M.SetMemory(0x604, 0x4f5f6f7f)
	M.SetMemory(0x608, 0x8f9fafbf)
	M.SetMemory(0x60c, 0xcfdfefff)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O CChain expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O CChain expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O CChain CSW1 expected %08x got: %08x", 0x00000518, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O CChain CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := getMemByte(uint32(0x600 + i))
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O CChain Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
	v = M.GetMemory(0x701)
	if v != 0xff00ffff {
		t.Errorf("Start I/O CChain Sebnse expected %08x got: %08x", 0xff00ffff, v)
	}
}

func TestStartIOCChainSLI(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x60000010)
	M.SetMemory(0x508, 0x02000700)
	M.SetMemory(0x50c, 0x00000020)
	M.SetMemory(0x700, 0xffffffff)
	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		M.SetMemory(i, 0x55555555)
		M.SetMemory(i+0x100, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O CChain SLI expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O CChain SLI expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O CChain SLI CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O CChain SLI CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x20 {
		vb := getMemByte(uint32(0x700 + i))
		mb := uint8(0x10 + i)
		if vb != mb {
			t.Errorf("Start I/O CChain SLI Data expected %02x got: %02x at: %08x", mb, vb, 0x600+i)
		}
		vb = getMemByte(uint32(0x600 + i))
		if i >= 0x10 {
			mb = 0x55
		}
		if vb != mb {
			t.Errorf("Start I/O CChain SLI Data expected %02x got: %02x at: %08x", mb, vb, 0x600+i)
		}
	}
}

func TestStartIOCChainNop(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x03000600) // Set channel words
	M.SetMemory(0x504, 0x40000001)
	M.SetMemory(0x508, 0x03000700)
	M.SetMemory(0x50c, 0x00000001)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O CChain Nop expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O CChain Nop expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O CChain Nop CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000001 {
		t.Errorf("Start I/O CChain Nop CSW2 expected %08x got: %08x", 0x0c000001, v)
	}
}

// Test TIC
func TestStartIOTic(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = 0x55
	}
	d.max = 0x10

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x40000010)
	M.SetMemory(0x508, 0x08000520) // TIC to 520
	M.SetMemory(0x50c, 0x40000001)
	M.SetMemory(0x520, 0x03000701) // NOP
	M.SetMemory(0x524, 0x40000001)
	M.SetMemory(0x528, 0x04000701) // Sense
	M.SetMemory(0x52c, 0x00000001)
	M.SetMemory(0x700, 0xffffffff)
	M.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	M.SetMemory(0x604, 0x4f5f6f7f)
	M.SetMemory(0x608, 0x8f9fafbf)
	M.SetMemory(0x60c, 0xcfdfefff)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Tic expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Tic expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000530 {
		t.Errorf("Start I/O Tic CSW1 expected %08x got: %08x", 0x00000530, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Tic CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = M.GetMemory(0x700)
	if v != 0xff00ffff {
		t.Errorf("Start I/O Tic Sense Data expected %08x got: %08x", 0xff00ffff, v)
	}

	for i := range 0x10 {
		vb := d.data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Tic Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
}

// Test TIC to another TIC
func TestStartIOTicTic(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x10

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x40000010)
	M.SetMemory(0x508, 0x08000518) // TIC to 518
	M.SetMemory(0x50c, 0x40000001)
	M.SetMemory(0x510, 0x04000701) // Sense
	M.SetMemory(0x514, 0x00000001)
	M.SetMemory(0x518, 0x08000510) // TIC to 510
	M.SetMemory(0x51c, 0x00000000) // TIC to 510
	M.SetMemory(0x700, 0xffffffff)
	M.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	M.SetMemory(0x604, 0x4f5f6f7f)
	M.SetMemory(0x608, 0x8f9fafbf)
	M.SetMemory(0x60c, 0xcfdfefff)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Tic Tic expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Tic Tic expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000520 {
		t.Errorf("Start I/O Tic Tic CSW1 expected %08x got: %08x", 0x00000520, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x00200000 {
		t.Errorf("Start I/O Tic Tic CSW2 expected %08x got: %08x", 0x00200000, v)
	}

	v = M.GetMemory(0x700)
	if v != 0xffffffff {
		t.Errorf("Start I/O Tic Tic Sense Data expected %08x got: %08x", 0xfffffff, v)
	}

	for i := range 0x10 {
		vb := d.data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Tic Tic Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
}

// Test TIC as first command
func TestStartIOTicError(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x20

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x08000520) // Set channel words
	M.SetMemory(0x504, 0x40000001)
	M.SetMemory(0x508, 0x04000702)
	M.SetMemory(0x50c, 0x40000001)
	M.SetMemory(0x700, 0xffffffff)

	cc := StartIO(0x00f)
	if cc != 1 {
		t.Errorf("Start I/O TIC Error expected %d got: %d", 1, cc)
	}

	v = M.GetMemory(0x40)
	if v != 0xffffffff {
		t.Errorf("Start I/O TIC Error CSW1 expected %08x got: %08x", 0xffffffff, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0020ffff {
		t.Errorf("Start I/O TIC Error CSW2 expected %08x got: %08x", 0x0020ffff, v)
	}

	v = M.GetMemory(0x700)
	if v != 0xffffffff {
		t.Errorf("Start I/O Tic Error Sense Data expected %08x got: %08x", 0xfffffff, v)
	}

}

// Test TIC
func TestStartIOSMSTic(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x20 {
		d.data[i] = 0x55
	}
	d.max = 0x10
	d.sms = true

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x01000600) // Set channel words
	M.SetMemory(0x504, 0x40000010)
	M.SetMemory(0x508, 0x08000520) // TIC to 520
	M.SetMemory(0x50c, 0x00000000)
	M.SetMemory(0x510, 0x08000540)
	M.SetMemory(0x514, 0x00000000)
	M.SetMemory(0x520, 0x03000701) // NOP
	M.SetMemory(0x524, 0x40000001)
	M.SetMemory(0x528, 0x04000701) // Sense
	M.SetMemory(0x52c, 0x00000001)
	M.SetMemory(0x540, 0x04000703) // Sense
	M.SetMemory(0x544, 0x00000001)
	M.SetMemory(0x700, 0xffffffff)
	M.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	M.SetMemory(0x604, 0x4f5f6f7f)
	M.SetMemory(0x608, 0x8f9fafbf)
	M.SetMemory(0x60c, 0xcfdfefff)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O SMS expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O SMS expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000548 {
		t.Errorf("Start I/O SMS CSW1 expected %08x got: %08x", 0x00000548, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O SMS CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = M.GetMemory(0x700)
	if v != 0xffffff00 {
		t.Errorf("Start I/O SMS Memory expected %08x got: %08x", 0xffffff00, v)
	}

	for i := range 0x10 {
		vb := d.data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O SMS Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
}

// Test if PCI interrupts work
func TestStartIOPCI(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x40 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x40

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0x80000005)
	M.SetMemory(0x508, 0x00000605)
	M.SetMemory(0x50c, 0x8800000b)
	M.SetMemory(0x510, 0x00000610)
	M.SetMemory(0x514, 0x20000020)
	M.SetMemory(0x600, 0x55555555) // Invalid data
	M.SetMemory(0x604, 0x55555555)
	M.SetMemory(0x608, 0x55555555)
	M.SetMemory(0x60c, 0x55555555)
	M.SetMemory(0x610, 0x55555555)
	M.SetMemory(0x614, 0x55555555)
	M.SetMemory(0x618, 0x55555555)
	M.SetMemory(0x61c, 0x55555555)
	M.SetMemory(0x620, 0x55555555)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O PCI expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O PCI expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x44) & M.UMASK
	if v != 0x00800000 {
		t.Errorf("Start I/O PCI CSW2 PCI expected %08x got: %08x", 0x00800000, v)
	}

	dev = runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O PCI expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000518 {
		t.Errorf("Start I/O PCI CSW1 expected %08x got: %08x", 0x00000518, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O PCI CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x20 {
		vb := d.data[i]
		mb := uint8(0x10 + i)
		if vb != mb {
			t.Errorf("Start I/O PCI Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
}

func TestStartIOHaltIO1(t *testing.T) {
	d := setup(0xf)

	// Load Data
	for i := range 0x40 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x40

	_ = TestIO(0x00f)
	cc := HaltIO(0x00f)

	if cc != 0 {
		t.Errorf("Start I/O HaltIO expected %d got: %d", 0, cc)
	}
}

// Halt I/O on running device.
func TestStartIOHaltIO2(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x80 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x80

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0xc8000001)
	M.SetMemory(0x508, 0x00000601)
	M.SetMemory(0x50c, 0x8000003f)
	M.SetMemory(0x510, 0x00000640)
	M.SetMemory(0x514, 0x00000040)
	M.SetMemory(0x518, 0x04000700)
	M.SetMemory(0x51c, 0x00000001)
	for i := range uint32(0x100) {
		M.SetMemory(0x600+i, 0x55555555) // Invalid data
	}
	M.SetMemory(0x700, 0xffffffff)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Haltio2 expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Haltio2 expected %d got: %d", 0xf, dev)
	}

	v = M.GetMemory(0x44) & M.UMASK
	if v != 0x00800000 {
		t.Errorf("Start I/O Haltio2 CSW2 PCI expected %08x got: %08x", 0x00800000, v)
	}

	cc = HaltIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Haltio2 expected %d got: %d", 0, cc)
	}

	dev = runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Haltio2 expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Haltio2 CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = M.GetMemory(0x44) & 0xffbf0000
	if v != 0x0c000000 {
		t.Errorf("Start I/O Haltio2 CSW2 expected %08x got: %08x", 0x0c000000, v)
	}
}

func TestStartIOTIOBusy(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x80 {
		d.data[i] = uint8(0x10 + i)
	}
	d.max = 0x80

	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x500)
	M.SetMemory(0x500, 0x02000600) // Set channel words
	M.SetMemory(0x504, 0xc8000001)
	M.SetMemory(0x508, 0x00000601)
	M.SetMemory(0x50c, 0x8000003f)
	M.SetMemory(0x510, 0x00000640)
	M.SetMemory(0x514, 0x00000040)
	M.SetMemory(0x518, 0x04000700)
	M.SetMemory(0x51c, 0x00000001)
	for i := range uint32(0x100) {
		M.SetMemory(0x600+i, 0x55555555) // Invalid data
	}
	M.SetMemory(0x700, 0xffffffff)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O TIO Busy expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O TIO Busy expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x44) & M.UMASK
	if v != 0x00800000 {
		t.Errorf("Start I/O TIO Busy CSW2 PCI expected %08x got: %08x", 0x00800000, v)
	}

	cc = TestIO(0x00f)
	if cc != 2 {
		t.Errorf("Start I/O TIO Busy expected %d got: %d", 2, cc)
	}

	dev = runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O TIO Busy expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x00000518 {
		t.Errorf("Start I/O TIO Busy CSW1 expected %08x got: %08x", 0x00000518, v)
	}
	v = M.GetMemory(0x44) & 0xffbf0000
	if v != 0x0c000000 {
		t.Errorf("Start I/O TIO Busy CSW2 expected %08x got: %08x", 0x0c000000, v)
	}
}

// Read Protection check
func TestStartIOReadProt(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x10 {
		d.data[i] = 0x55
	}
	d.max = 0x10

	M.PutKey(0x4000, 0x30)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x20000500)
	M.SetMemory(0x500, 0x01004000) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	M.SetMemory(0x4000, 0x0f1f2f3f) // Data to send
	M.SetMemory(0x4004, 0x4f5f6f7f)
	M.SetMemory(0x4008, 0x8f9fafbf)
	M.SetMemory(0x400c, 0xcfdfefff)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Read Prot expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Read Prot expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x20000508 {
		t.Errorf("Start I/O Read Prot CSW1 expected %08x got: %08x", 0x20000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Read Prot CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := d.data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Read Prot Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
	M.PutKey(0x4000, 0x0)
}

func TestStartIOWriteProt(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x10 {
		d.data[i] = uint8(0xf0 + i)
	}
	d.max = 0x10

	M.PutKey(0x4000, 0x30)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x20000500)
	M.SetMemory(0x500, 0x02004000) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := uint32(0x4000); i < 0x4040; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Write Prot expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Write Prot expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x20000508 {
		t.Errorf("Start I/O Write Prot  CSW1 expected %08x got: %08x", 0x20000508, v)
	}
	v = M.GetMemory(0x44) & M.UMASK
	if v != 0x0c500000 {
		t.Errorf("Start I/O Write Prot CSW2 expected %08x got: %08x", 0x0c500000, v)
	}

	for i := range 0x10 {
		vb := getMemByte(uint32(0x4000 + i))
		if vb != 0x55 {
			t.Errorf("Start I/O Write Prot Data expected %02x got: %02x at: %02x", 0x55, vb, i)
		}
	}
	M.PutKey(0x4000, 0x0)
}

// Read Protection check
func TestStartIOReadProt2(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x10 {
		d.data[i] = 0x55
	}
	d.max = 0x10

	M.PutKey(0x4000, 0x30)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x30000500)
	M.SetMemory(0x500, 0x01004000) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	M.SetMemory(0x4000, 0x0f1f2f3f) // Data to send
	M.SetMemory(0x4004, 0x4f5f6f7f)
	M.SetMemory(0x4008, 0x8f9fafbf)
	M.SetMemory(0x400c, 0xcfdfefff)

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Read Prot expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Read Prot expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x30000508 {
		t.Errorf("Start I/O Read Prot CSW1 expected %08x got: %08x", 0x30000508, v)
	}
	v = M.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Read Prot CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := d.data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Read Prot Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
	M.PutKey(0x4000, 0x0)
}

func TestStartIOWriteProt2(t *testing.T) {
	var v uint32

	d := setup(0xf)

	// Load Data
	for i := range 0x10 {
		d.data[i] = uint8(0xf0 + i)
	}
	d.max = 0x10

	M.PutKey(0x4000, 0x30)
	M.SetMemory(0x40, 0xffffffff)
	M.SetMemory(0x44, 0xffffffff)
	M.SetMemory(0x78, 0)
	M.SetMemory(0x7c, 0x420)
	M.SetMemory(0x48, 0x30000500)
	M.SetMemory(0x500, 0x02004000) // Set channel words
	M.SetMemory(0x504, 0x00000010)
	M.SetMemory(0x508, 0)
	M.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := uint32(0x4000); i < 0x4040; i += 4 {
		M.SetMemory(i, 0x55555555)
	}

	cc := StartIO(0x00f)
	if cc != 0 {
		t.Errorf("Start I/O Write Prot expected %d got: %d", 0, cc)
	}

	dev := runChannel()
	if dev != 0xf {
		t.Errorf("Start I/O Write Prot expected %d got: %d", 0xf, dev)
	}
	v = M.GetMemory(0x40)
	if v != 0x30000508 {
		t.Errorf("Start I/O Write Prot  CSW1 expected %08x got: %08x", 0x30000508, v)
	}
	v = M.GetMemory(0x44) & M.UMASK
	if v != 0x0c000000 {
		t.Errorf("Start I/O Write Prot CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := getMemByte(uint32(0x4000 + i))
		mb := uint8(0xf0 + i)
		if vb != mb {
			t.Errorf("Start I/O Write Prot Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
	M.PutKey(0x4000, 0x0)
}
