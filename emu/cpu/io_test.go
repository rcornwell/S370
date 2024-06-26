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

package cpu

import (
	"testing"

	dev "github.com/rcornwell/S370/emu/device"
	ev "github.com/rcornwell/S370/emu/event"
	mem "github.com/rcornwell/S370/emu/memory"
	ch "github.com/rcornwell/S370/emu/sys_channel"
	Td "github.com/rcornwell/S370/emu/test_dev"
)

func ioSetup() *Td.TestDev {
	setup()
	ch.InitializeChannels()
	ch.AddChannel(0, dev.TypeMux, 192)
	d := &Td.TestDev{Addr: 0xf, Mask: 0xff}
	ch.AddDevice(d, 0xf)
	_ = d.InitDev()
	for i := range 0x10 {
		d.Data[i] = uint8(0xf0 + i)
	}
	d.Max = 0x10
	return d
}

// Read byte from main memory.
func getMemByte(addr uint32) uint8 {
	v := mem.GetMemory(addr)
	b := uint8((v >> (8 * (3 - (addr & 3))) & 0xff))
	return b
}

// write byte to main memory.
func setMemByte(addr uint32, data uint32) {
	off := 8 * (3 - (addr & 3))
	m := uint32(0xff << off)
	d := (data & 0xff) << off
	mem.SetMemoryMask(addr, d, m)
}

// Run a test of an I/O instruction.
func (cpu *cpu) iotestInst(mask uint8, steps int) {
	cpu.PC = 0x400
	cpu.progMask = mask & 0xf
	cpu.sysMask = 0x0000
	cpu.irqEnb = false
	mem.SetMemory(0x68, 0)
	mem.SetMemory(0x6c, 0x800)
	trapFlag = false
	cy := 0
	for range steps {
		cy++
		c := CycleCPU()

		if cpu.PC == 0x800 {
			trapFlag = true
		}
		// Stop it next opcode = 0
		w := mem.GetMemory(cpu.PC)
		if (cpu.PC & 2) == 0 {
			w >>= 16
		}
		if (w & 0xffff) == 0 {
			break
		}
		if c == 0 {
			c = 1
		}
		ev.Advance(c)
	}
}

// Debug channel test.
func TestCycleTch(t *testing.T) {
	ioSetup()
	mem.SetMemory(0x400, 0x9f00040f)
	mem.SetMemory(0x404, 0)
	cpuState.iotestInst(0, 20)
	if cpuState.cc != 3 {
		t.Errorf("Test Channel on non-existing channel failed expected %d got: %d", 3, cpuState.cc)
	}
	mem.SetMemory(0x400, 0x9f00000f)
	mem.SetMemory(0x404, 0)
	cpuState.iotestInst(0, 20)
	if cpuState.cc != 0 {
		t.Errorf("Test Channel on non-existing channel failed expected %d got: %d", 0, cpuState.cc)
	}
}

func TestTestIO(t *testing.T) {
	_ = ioSetup()
	mem.SetMemory(0x400, 0x9d00000f)
	mem.SetMemory(0x404, 0)
	cpuState.iotestInst(0, 20)
	if cpuState.cc != 0 {
		t.Errorf("Test I/O expected %d got: %d", 0, cpuState.cc)
	}
	mem.SetMemory(0x400, 0x9d00000d)
	mem.SetMemory(0x404, 0)
	cpuState.iotestInst(0, 20)
	if cpuState.cc != 3 {
		t.Errorf("Test I/O expected %d got: %d", 3, cpuState.cc)
	}
}

func TestCycleSIO(t *testing.T) {
	var v uint32

	td := ioSetup()
	mem.SetMemory(0x40, 0)
	mem.SetMemory(0x44, 0)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47000424) // BC  0,424
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0x00000010)

	// Load memory with value not equal to rea1 CSW2d data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O 1 CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = mem.GetMemory(0x38)
	if v != 0xff06000f {
		t.Errorf("Start I/O 1 OIOPSW2 expected %08x got: %08x", 0xff06000f, v)
	}
	v = mem.GetMemory(0x3c)
	if v != 0x14000408 {
		t.Errorf("Start I/O 1 OIOPSW2 expected %08x got: %08x", 0x94000408, v)
	}
	for i := range 0x10 {
		b := getMemByte(uint32(0x600 + i))
		mb := uint8(0xf0 + i)
		if b != mb {
			t.Errorf("Start I/O 1 Invalid data %02x expected: %02x got %02x", i, mb, b)
		}
	}

	mem.SetMemory(0x40, 0)
	mem.SetMemory(0x44, 0)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47000424) // BC  0,424
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x00000010)

	mem.SetMemory(0x600, 0xf0f1f2f3) // Validate data
	mem.SetMemory(0x604, 0xf4f5f6f7)
	mem.SetMemory(0x608, 0xf8f9fafb)
	mem.SetMemory(0x60C, 0xfcfdfeff)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O 1 CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = mem.GetMemory(0x38)
	if v != 0xff06000f {
		t.Errorf("Start I/O 1 OIOPSW2 expected %08x got: %08x", 0xff06000f, v)
	}
	v = mem.GetMemory(0x3c)
	if v != 0x14000408 {
		t.Errorf("Start I/O 1 OIOPSW2 expected %08x got: %08x", 0x94000408, v)
	}

	for i := range 0x10 {
		b := td.Data[i]
		if b != uint8(0xf0+i) {
			t.Errorf("Start I/O 2 Invalid data %02x expected: %02x got %02x", i, 0x0f+i, b)
		}
	}
}

func TestCycleSense(t *testing.T) {
	var v uint32

	_ = ioSetup()
	mem.SetMemory(0x40, 0)
	mem.SetMemory(0x44, 0)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x408, 0x47700404) // BC  7,404
	mem.SetMemory(0x410, 0x00000000)

	mem.SetMemory(0x500, 0x04000600) // Set channel words
	mem.SetMemory(0x504, 0x00000001)

	mem.SetMemory(0x600, 0xffffffff) // Invalidate data

	cpuState.iotestInst(0, 50)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O 1 CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = mem.GetMemory(0x600)
	if v != 0x00ffffff {
		t.Errorf("Start I/O sense expected %08x got: %08x", 0x00ffffff, v)
	}
}

func TestCycleNop(t *testing.T) {
	var v uint32

	_ = ioSetup()
	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x408, 0x47700404) // BC  7,424
	mem.SetMemory(0x410, 0xc0000000)

	mem.SetMemory(0x500, 0x03000600) // Set channel words
	mem.SetMemory(0x504, 0x00000001)

	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}
	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O nop CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000001 {
		t.Errorf("Start I/O nop CSW2 expected %08x got: %08x", 0x0c000001, v)
	}

	v = mem.GetMemory(0x600)
	if v != 0x55555555 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0x55555555, v)
	}

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x408, 0x47700404) // BC  7,424
	mem.SetMemory(0x410, 0xc0000000)

	mem.SetMemory(0x500, 0x03000600) // Set channel words
	mem.SetMemory(0x504, 0x00000000)

	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}
	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0xffffffff {
		t.Errorf("Start I/O zero count CSW1 expected %08x got: %08x", 0xffffffff, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0020ffff {
		t.Errorf("Start I/O zero count CSW2 expected %08x got: %08x", 0x0020ffff, v)
	}

	v = mem.GetMemory(0x600)
	if v != 0x55555555 {
		t.Errorf("Start I/O zero count Dsts expected %08x got: %08x", 0x55555555, v)
	}
}

func TestCycleCEOnly(t *testing.T) {
	var v uint32

	_ = ioSetup()
	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x47800410) // BC 8,410
	mem.SetMemory(0x408, 0x58100040) // L 1,40  Save initial status
	mem.SetMemory(0x40c, 0x58200044) // L 2,44
	mem.SetMemory(0x410, 0x82000430) // LPSW 0430
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x430, 0xff060000) // Wait PSW
	mem.SetMemory(0x434, 0x14000420)

	mem.SetMemory(0x500, 0x13000600) // Set channel words
	mem.SetMemory(0x504, 0x00000001)

	cpuState.iotestInst(0, 2000)

	v = cpuState.regs[1]
	if v != 0xffffffff {
		t.Errorf("Start I/O Initial CSW1 expected %08x got: %08x", 0xffffffff, v)
	}
	v = cpuState.regs[2]
	if v != 0x0800ffff {
		t.Errorf("Start I/O Initial CSW2 expected %08x got: %08x", 0x0800ffff, v)
	}

	v = mem.GetMemory(0x40)
	if v != 0x00000000 {
		t.Errorf("Start I/O 1 CSW1 expected %08x got: %08x", 0x00000000, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x04000000 {
		t.Errorf("Start I/O 1 CSW2 expected %08x got: %08x", 0x04000000, v)
	}

	v = mem.GetMemory(0x38)
	if v != 0xff06000f {
		t.Errorf("Start I/O 1 OIOPSW2 expected %08x got: %08x", 0xff06000f, v)
	}
	v = mem.GetMemory(0x3c)
	if v != 0x14000420 {
		t.Errorf("Start I/O 1 OIOPSW2 expected %08x got: %08x", 0x14000420, v)
	}
}

func TestCycleCCNop(t *testing.T) {
	var v uint32

	_ = ioSetup()
	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x47800410) // BC 8,410
	mem.SetMemory(0x408, 0x58100040) // L 1,40  Save initial status
	mem.SetMemory(0x40c, 0x58200044) // L 2,44
	mem.SetMemory(0x410, 0x82000430) // LPSW 0430
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x430, 0xff060000) // Wait PSW
	mem.SetMemory(0x434, 0x14000420)

	mem.SetMemory(0x500, 0x13000600) // Set channel words
	mem.SetMemory(0x504, 0x40000001)
	mem.SetMemory(0x508, 0x03000600)
	mem.SetMemory(0x50c, 0x00000001)

	cpuState.iotestInst(0, 2000)

	v = cpuState.regs[1]
	if v != 0xffffffff {
		t.Errorf("Start I/O Initial CCNOP CSW1 expected %08x got: %08x", 0xffffffff, v)
	}
	v = cpuState.regs[2]
	if v != 0x0800ffff {
		t.Errorf("Start I/O Initial CCNOP CSW2 expected %08x got: %08x", 0x0800ffff, v)
	}

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O 1 CSW1 CCNOP expected %08x got: %08x", 0x00000510, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000001 {
		t.Errorf("Start I/O 1 CSW2 CCNOP expected %08x got: %08x", 0x0c000001, v)
	}

	v = mem.GetMemory(0x38)
	if v != 0xff06000f {
		t.Errorf("Start I/O 1 OIOPSW2 CCNOP expected %08x got: %08x", 0xff06000f, v)
	}
	v = mem.GetMemory(0x3c)
	if v != 0x14000420 {
		t.Errorf("Start I/O 1 OIOPSW2 CCNOP expected %08x got: %08x", 0x14000420, v)
	}
}

func TestCycleRead(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0x00000020)

	// Load memory with value not equal to rea1 CSW2d data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Read  CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
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

func TestCycleReadShort(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0x00000010)

	// Load memory with value not equal to rea1 CSW2d data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Read Short CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c400000 {
		t.Errorf("Start I/O Short Read Short CSW2 expected %08x got: %08x", 0x0c400000, v)
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

func TestCycleReadShortSLI(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0x20000010)

	// Load memory with value not equal to rea1 CSW2d data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}
	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Short Read SLI CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Short Read SLI CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O  Short Read SLI CSW2 expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
	for i := range 0x10 {
		vb := getMemByte(uint32(0x610 + i))
		if vb != 0x55 {
			t.Errorf("Start I/O Short Read SLI Data expected %02x got: %02x at: %02x", 0x55, vb, i)
		}
	}
}

func TestCycleWrite(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x55)
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x00000020)

	// Load memory with value not equal to read data.
	for i := range 0x20 {
		setMemByte(uint32(i+0x600), uint32(0x10+i))
	}
	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Write CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Write CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x20 {
		vb := d.Data[i]
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Write Dev Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
		vb = getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Write Mem Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
}

func TestCycleWriteShort(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x55)
	}
	d.Max = 0x10

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x00000020)

	// Load memory with value not equal to read data.
	for i := range 0x20 {
		setMemByte(uint32(i+0x600), uint32(0x10+i))
	}

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Short Write CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c40000f {
		t.Errorf("Start I/O Short Write CSW2 expected %08x got: %08x", 0x0c40000f, v)
	}

	for i := range 0x20 {
		mb := uint8(0x10 + i)
		if i > 0x10 {
			mb = 0x55
		}
		vb := d.Data[i]
		if vb != mb {
			t.Errorf("Start I/O Write Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
		vb = getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Short Write Mem Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
}

func TestCycleWriteShortSLI(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x55)
	}
	d.Max = 0x10

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x20000020)

	// Load memory with value not equal to read data.
	for i := range 0x20 {
		setMemByte(uint32(i+0x600), uint32(0x10+i))
	}
	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Short Write CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c00000f {
		t.Errorf("Start I/O Short Write CSW2 expected %08x got: %08x", 0x0c00000f, v)
	}

	for i := range 0x20 {
		mb := uint8(0x10 + i)
		if i > 0x10 {
			mb = 0x55
		}
		vb := d.Data[i]
		if vb != mb {
			t.Errorf("Start I/O Write Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
		vb = getMemByte(uint32(0x600 + i))
		if vb != uint8(0x10+i) {
			t.Errorf("Start I/O Short Write Mem Data expected %02x got: %02x at: %02x", 0x10+i, vb, i)
		}
	}
}

func TestCycleReadCDA(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0x80000010)
	mem.SetMemory(0x508, 0x01000700)
	mem.SetMemory(0x50c, 0x00000010)

	// Load memory with value not equal to read data.
	for i := range uint32(0x20) {
		mem.SetMemory(0x600+i, 0x55555555)
		mem.SetMemory(0x700+i, 0x55555555)
	}
	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O Read CDA CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = mem.GetMemory(0x44)
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

func TestCycleWriteCDA(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = 0x55
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x80000010)
	mem.SetMemory(0x508, 0x00000700)
	mem.SetMemory(0x50c, 0x00000010)

	mem.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	mem.SetMemory(0x604, 0x4f5f6f7f)
	mem.SetMemory(0x608, 0x8f9fafbf)
	mem.SetMemory(0x60c, 0xcfdfefff)
	mem.SetMemory(0x700, 0x0c1c2c3c) // Data to send
	mem.SetMemory(0x704, 0x4c5c6c7c)
	mem.SetMemory(0x708, 0x8c9cacbc)
	mem.SetMemory(0x70c, 0xccdcecfc)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O Write CDA CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Write CDA CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := d.Data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Write CDA Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
	for i := range 0x10 {
		vb := d.Data[i+0x10]
		mb := uint8(0xc + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Write CDA Data expected %02x got: %02x at: %02x", mb, vb, i+0x10)
		}
	}
}

func TestCycleReadCDASkip(t *testing.T) {
	var v uint32

	d := ioSetup()
	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x10

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0x90000005)
	mem.SetMemory(0x508, 0x02000606)
	mem.SetMemory(0x50c, 0x0000000b)

	// Load memory with value not equal to read data.
	for i := range uint32(0x20) {
		mem.SetMemory(0x600+i, 0x55555555)
		mem.SetMemory(0x700+i, 0x55555555)
	}
	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O Read CDA CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = mem.GetMemory(0x44)
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

func TestCycleReadBkwd(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x10 {
		d.Data[i] = uint8(0x10 + (0x0f - i))
	}
	d.Max = 0x10

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x0c00060f) // Set channel words
	mem.SetMemory(0x504, 0x00000010)
	mem.SetMemory(0x508, 0)
	mem.SetMemory(0x50c, 0)

	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000508 {
		t.Errorf("Start I/O Read Bkwd CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44)
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

func TestCycleCChain(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x20 {
		d.Data[i] = 0x55
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x40000010)
	mem.SetMemory(0x508, 0x03000701)
	mem.SetMemory(0x50c, 0x40000001)
	mem.SetMemory(0x508, 0x04000701)
	mem.SetMemory(0x50c, 0x00000001)

	mem.SetMemory(0x700, 0xffffffff)
	mem.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	mem.SetMemory(0x604, 0x4f5f6f7f)
	mem.SetMemory(0x608, 0x8f9fafbf)
	mem.SetMemory(0x60c, 0xcfdfefff)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O CChain CSW1 expected %08x got: %08x", 0x00000518, v)
	}
	v = mem.GetMemory(0x44)
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
	v = mem.GetMemory(0x701)
	if v != 0xff00ffff {
		t.Errorf("Start I/O CChain Sebnse expected %08x got: %08x", 0xff00ffff, v)
	}
}

func TestCycleCChainSLI(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0x60000010)
	mem.SetMemory(0x508, 0x02000700)
	mem.SetMemory(0x50c, 0x00000020)

	// Load memory with value not equal to read data.
	for i := uint32(0x600); i < 0x640; i += 4 {
		mem.SetMemory(i, 0x55555555)
		mem.SetMemory(i+0x100, 0x55555555)
	}

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O CChain SLI CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = mem.GetMemory(0x44)
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

func TestCycleCChainNop(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x03000600) // Set channel words
	mem.SetMemory(0x504, 0x40000001)
	mem.SetMemory(0x508, 0x03000700)
	mem.SetMemory(0x50c, 0x00000001)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O CChain Nop CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000001 {
		t.Errorf("Start I/O CChain Nop CSW2 expected %08x got: %08x", 0x0c000001, v)
	}
}

// Test TIC.
func TestStartIOTic(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x20 {
		d.Data[i] = 0x55
	}
	d.Max = 0x10

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x40000010)
	mem.SetMemory(0x508, 0x08000520) // TIC to 520
	mem.SetMemory(0x50c, 0x40000001)
	mem.SetMemory(0x520, 0x03000701) // NOP
	mem.SetMemory(0x524, 0x40000001)
	mem.SetMemory(0x528, 0x04000701) // Sense
	mem.SetMemory(0x52c, 0x00000001)

	mem.SetMemory(0x700, 0xffffffff)
	mem.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	mem.SetMemory(0x604, 0x4f5f6f7f)
	mem.SetMemory(0x608, 0x8f9fafbf)
	mem.SetMemory(0x60c, 0xcfdfefff)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000530 {
		t.Errorf("Start I/O Tic CSW1 expected %08x got: %08x", 0x00000530, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Tic CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = mem.GetMemory(0x700)
	if v != 0xff00ffff {
		t.Errorf("Start I/O Tic Sense Data expected %08x got: %08x", 0xff00ffff, v)
	}

	for i := range 0x10 {
		vb := d.Data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Tic Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
}

// Test TIC to another TIC.
func TestCycleTicTic(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x10

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)
	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x40000010)
	mem.SetMemory(0x508, 0x08000518) // TIC to 518
	mem.SetMemory(0x50c, 0x40000001)
	mem.SetMemory(0x510, 0x04000701) // Sense
	mem.SetMemory(0x514, 0x00000001)
	mem.SetMemory(0x518, 0x08000510) // TIC to 510
	mem.SetMemory(0x51c, 0x00000000) // TIC to 510

	mem.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	mem.SetMemory(0x604, 0x4f5f6f7f)
	mem.SetMemory(0x608, 0x8f9fafbf)
	mem.SetMemory(0x60c, 0xcfdfefff)

	mem.SetMemory(0x700, 0xffffffff)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000520 {
		t.Errorf("Start I/O Tic to Tic CSW1 expected %08x got: %08x", 0x00000520, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x00200000 {
		t.Errorf("Start I/O Tic to Tic CSW2 expected %08x got: %08x", 0x00200000, v)
	}

	v = mem.GetMemory(0x700)
	if v != 0xffffffff {
		t.Errorf("Start I/O Tic to Tic Sense Data expected %08x got: %08x", 0xfffffff, v)
	}

	for i := range 0x10 {
		vb := d.Data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Tic to Tic Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
}

// Test TIC as first command.
func TestCycleTicError(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x20 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x20

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 0f
	mem.SetMemory(0x404, 0x47300400) // BC 3,404
	mem.SetMemory(0x408, 0x47800428) // BC 8,428
	mem.SetMemory(0x40c, 0x9d00000f) // TIO 0f
	mem.SetMemory(0x410, 0x4770040c) // BC 7,40c
	mem.SetMemory(0x414, 0x47f00428) // BC 0xf,428
	mem.SetMemory(0x420, 0x00000000) // stop

	mem.SetMemory(0x500, 0x08000520) // Set channel words
	mem.SetMemory(0x504, 0x40000001)
	mem.SetMemory(0x508, 0x04000702)
	mem.SetMemory(0x50c, 0x40000001)
	mem.SetMemory(0x700, 0xffffffff)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0xffffffff {
		t.Errorf("Start I/O TIC Error CSW1 expected %08x got: %08x", 0xffffffff, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0020ffff {
		t.Errorf("Start I/O TIC Error CSW2 expected %08x got: %08x", 0x0020ffff, v)
	}

	v = mem.GetMemory(0x700)
	if v != 0xffffffff {
		t.Errorf("Start I/O Tic Error Sense Data expected %08x got: %08x", 0xfffffff, v)
	}
}

// Test TIC.
func TestCycleSMSTic(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x20 {
		d.Data[i] = 0x55
	}
	d.Max = 0x10
	d.Sms = true

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000408)

	mem.SetMemory(0x500, 0x01000600) // Set channel words
	mem.SetMemory(0x504, 0x40000010)
	mem.SetMemory(0x508, 0x08000520) // TIC to 520
	mem.SetMemory(0x50c, 0x00000000)
	mem.SetMemory(0x510, 0x08000540)
	mem.SetMemory(0x514, 0x00000000)
	mem.SetMemory(0x520, 0x03000701) // NOP
	mem.SetMemory(0x524, 0x40000001)
	mem.SetMemory(0x528, 0x04000701) // Sense
	mem.SetMemory(0x52c, 0x00000001)
	mem.SetMemory(0x540, 0x04000703) // Sense
	mem.SetMemory(0x544, 0x00000001)

	mem.SetMemory(0x600, 0x0f1f2f3f) // Data to send
	mem.SetMemory(0x604, 0x4f5f6f7f)
	mem.SetMemory(0x608, 0x8f9fafbf)
	mem.SetMemory(0x60c, 0xcfdfefff)

	mem.SetMemory(0x700, 0xffffffff)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x00000548 {
		t.Errorf("Start I/O SMS CSW1 expected %08x got: %08x", 0x00000548, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O SMS CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	v = mem.GetMemory(0x700)
	if v != 0xffffff00 {
		t.Errorf("Start I/O SMS Memory expected %08x got: %08x", 0xffffff00, v)
	}

	for i := range 0x10 {
		vb := d.Data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O SMS Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
}

// Test if PCI interrupts work.
func TestCyclePCI(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x40 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x40

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x408)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000430) // LPSW 0430
	mem.SetMemory(0x408, 0x58000040) // L 0, 040
	mem.SetMemory(0x40c, 0x58100044) // L 1, 044
	mem.SetMemory(0x410, 0x41200440) // LA 2,440
	mem.SetMemory(0x414, 0x5020007c) // ST 2,04c
	mem.SetMemory(0x418, 0x82000438) // LPSW 0438
	mem.SetMemory(0x440, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x444, 0x47700440) // BC  7,420
	mem.SetMemory(0x448, 0)
	mem.SetMemory(0x430, 0xff060000) // Wait PSW
	mem.SetMemory(0x434, 0x14000404)
	mem.SetMemory(0x438, 0xff060000) // Wait PSW
	mem.SetMemory(0x43c, 0x14000438)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0x80000005)
	mem.SetMemory(0x508, 0x00000605)
	mem.SetMemory(0x50c, 0x8800000b)
	mem.SetMemory(0x510, 0x00000610)
	mem.SetMemory(0x514, 0x20000020)

	mem.SetMemory(0x600, 0x55555555) // Invalid data
	mem.SetMemory(0x604, 0x55555555)
	mem.SetMemory(0x608, 0x55555555)
	mem.SetMemory(0x60c, 0x55555555)
	mem.SetMemory(0x610, 0x55555555)
	mem.SetMemory(0x614, 0x55555555)
	mem.SetMemory(0x618, 0x55555555)
	mem.SetMemory(0x61c, 0x55555555)
	mem.SetMemory(0x620, 0x55555555)

	cpuState.iotestInst(0, 2000)

	v = cpuState.regs[0] & 0xfffffff0
	if v != 0x00000510 {
		t.Errorf("Start I/O PCI CSW1 PCI expected 0x0000051x got: %08x", v)
	}

	v = cpuState.regs[1] & HMASK
	if v != 0x00800000 {
		t.Errorf("Start I/O PCI CSW2 PCI expected %08x got: %08x", 0x00800000, v)
	}

	v = mem.GetMemory(0x40)
	if v != 0x00000518 {
		t.Errorf("Start I/O PCI CSW1 expected %08x got: %08x", 0x00000518, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O PCI CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x20 {
		vb := d.Data[i]
		mb := uint8(0x10 + i)
		if vb != mb {
			t.Errorf("Start I/O PCI Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
}

func TestCycleHaltIO1(t *testing.T) {
	d := ioSetup()

	// Load Data
	for i := range 0x40 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x40

	mem.SetMemory(0x40, 0) // Set CSW to zero
	mem.SetMemory(0x44, 0)
	mem.SetMemory(0x78, 0x00000000)
	mem.SetMemory(0x7c, 0x00000408)

	mem.SetMemory(0x400, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x404, 0x47000400) // BC  0,400
	mem.SetMemory(0x408, 0x9e00000f) // HIO 00f
	mem.SetMemory(0x40c, 0)

	cpuState.iotestInst(0, 2000)

	if cpuState.cc != 1 {
		t.Errorf("Start I/O HaltIO expected %d got: %d", 1, cpuState.cc)
	}
}

// Halt I/O on running device.
func TestCycleHaltIO2(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x80 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x80

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x408)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000430) // LPSW 0430
	mem.SetMemory(0x408, 0x58000040) // L 0, 040
	mem.SetMemory(0x40c, 0x58100044) // L 1, 044
	mem.SetMemory(0x410, 0x9e00000f) // HIO 00f
	mem.SetMemory(0x414, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x418, 0x47700414) // BC  7,414
	mem.SetMemory(0x420, 0)
	mem.SetMemory(0x430, 0xff060000) // Wait PSW
	mem.SetMemory(0x434, 0x14000408)
	mem.SetMemory(0x438, 0xff060000) // Wait PSW
	mem.SetMemory(0x43c, 0x14000440)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0xc8000001)
	mem.SetMemory(0x508, 0x00000601)
	mem.SetMemory(0x50c, 0x8000003f)
	mem.SetMemory(0x510, 0x00000640)
	mem.SetMemory(0x514, 0x00000040)
	mem.SetMemory(0x518, 0x04000700)
	mem.SetMemory(0x51c, 0x00000001)
	for i := range uint32(0x100) {
		mem.SetMemory(0x600+i, 0x55555555) // Invalid data
	}
	mem.SetMemory(0x700, 0xffffffff)

	cpuState.iotestInst(0, 2000)

	v = cpuState.regs[1] & HMASK
	if v != 0x00800000 {
		t.Errorf("Start I/O Haltio2 CSW2 PCI expected %08x got: %08x", 0x00800000, v)
	}

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O Haltio2 CSW1 expected %08x got: %08x", 0x00000508, v)
	}
	v = mem.GetMemory(0x44) & 0xffbf0000
	if v != 0x0c000000 {
		t.Errorf("Start I/O Haltio2 CSW2 expected %08x got: %08x", 0x0c000000, v)
	}
	v = mem.GetMemory(0x700)
	if v != 0xffffffff {
		t.Errorf("Start I/O Haltio2 Memory expected %08x got: %08x", 0xffffffff, v)
	}
}

func TestCycleTIOBusy(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x80 {
		d.Data[i] = uint8(0x10 + i)
	}
	d.Max = 0x80

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x408)
	mem.SetMemory(0x48, 0x500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000430) // LPSW 0430
	mem.SetMemory(0x408, 0x9d00000f) // TIO  00f
	mem.SetMemory(0x40c, 0x05109d00) // BALR 1,0, TIO 00f
	mem.SetMemory(0x410, 0x000f0771) // 00f, BCR 7,1
	mem.SetMemory(0x414, 0)
	mem.SetMemory(0x430, 0xff060000) // Wait PSW
	mem.SetMemory(0x434, 0x14000408)
	mem.SetMemory(0x438, 0xff060000) // Wait PSW
	mem.SetMemory(0x43c, 0x14000440)

	mem.SetMemory(0x500, 0x02000600) // Set channel words
	mem.SetMemory(0x504, 0xc8000001)
	mem.SetMemory(0x508, 0x00000601)
	mem.SetMemory(0x50c, 0x0000007f)
	mem.SetMemory(0x510, 0x04000700)
	mem.SetMemory(0x514, 0x00000001)

	for i := range uint32(0x100) {
		mem.SetMemory(0x600+i, 0x55555555) // Invalid data
	}
	mem.SetMemory(0x700, 0xffffffff)

	cpuState.iotestInst(0, 2000)

	// The result of a PCI can have a variety of addresses
	v = cpuState.regs[1]
	if v != 0x6000040e {
		t.Errorf("Start I/O TIO Busy CSW2 PCI expected %08x got: %08x", 0x6000040e, v)
	}

	v = mem.GetMemory(0x40)
	if v != 0x00000510 {
		t.Errorf("Start I/O TIO Busy CSW1 expected %08x got: %08x", 0x00000510, v)
	}
	v = mem.GetMemory(0x44) & 0xffbf0000
	if v != 0x0c000000 {
		t.Errorf("Start I/O TIO Busy CSW2 expected %08x got: %08x", 0x0c000000, v)
	}
}

// Read Protection check.
func TestCycleReadProt(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x10 {
		d.Data[i] = 0x55
	}
	d.Max = 0x10

	mem.PutKey(0x4000, 0x30)
	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x20000500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x0)
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000410)
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700404) // BC  7,420

	mem.SetMemory(0x500, 0x01004000) // Set channel words
	mem.SetMemory(0x504, 0x00000010)

	mem.SetMemory(0x4000, 0x0f1f2f3f) // Data to send
	mem.SetMemory(0x4004, 0x4f5f6f7f)
	mem.SetMemory(0x4008, 0x8f9fafbf)
	mem.SetMemory(0x400c, 0xcfdfefff)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x20000508 {
		t.Errorf("Start I/O Read Prot CSW1 expected %08x got: %08x", 0x20000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Read Prot CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := d.Data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Read Prot Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
	mem.PutKey(0x4000, 0x0)
}

func TestCycleWriteProt(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x10 {
		d.Data[i] = uint8(0xf0 + i)
	}
	d.Max = 0x10

	mem.PutKey(0x4000, 0x30)
	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x20000500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000404)

	mem.SetMemory(0x500, 0x02004000) // Set channel words
	mem.SetMemory(0x504, 0x00000010)
	mem.SetMemory(0x508, 0)
	mem.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := uint32(0x4000); i < 0x4040; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x20000508 {
		t.Errorf("Start I/O Write Prot  CSW1 expected %08x got: %08x", 0x20000508, v)
	}
	v = mem.GetMemory(0x44) & HMASK
	if v != 0x0c500000 {
		t.Errorf("Start I/O Write Prot CSW2 expected %08x got: %08x", 0x0c500000, v)
	}

	for i := range 0x10 {
		vb := getMemByte(uint32(0x4000 + i))
		if vb != 0x55 {
			t.Errorf("Start I/O Write Prot Data expected %02x got: %02x at: %02x", 0x55, vb, i)
		}
	}
	mem.PutKey(0x4000, 0x0)
}

// Read Protection check.
func TestCycleReadProt2(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x10 {
		d.Data[i] = 0x55
	}
	d.Max = 0x10

	mem.PutKey(0x4000, 0x30)
	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x30000500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000404)

	mem.SetMemory(0x500, 0x01004000) // Set channel words
	mem.SetMemory(0x504, 0x00000010)

	mem.SetMemory(0x4000, 0x0f1f2f3f) // Data to send
	mem.SetMemory(0x4004, 0x4f5f6f7f)
	mem.SetMemory(0x4008, 0x8f9fafbf)
	mem.SetMemory(0x400c, 0xcfdfefff)

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x30000508 {
		t.Errorf("Start I/O Read Prot CSW1 expected %08x got: %08x", 0x30000508, v)
	}
	v = mem.GetMemory(0x44)
	if v != 0x0c000000 {
		t.Errorf("Start I/O Read Prot CSW2 expected %08x got: %08x", 0x0c000000, v)
	}

	for i := range 0x10 {
		vb := d.Data[i]
		mb := uint8(0xf + (i << 4))
		if vb != mb {
			t.Errorf("Start I/O Read Prot Data expected %02x got: %02x at: %02x", mb, vb, i)
		}
	}
	mem.PutKey(0x4000, 0x0)
}

func TestCycleWriteProt2(t *testing.T) {
	var v uint32

	d := ioSetup()

	// Load Data
	for i := range 0x10 {
		d.Data[i] = uint8(0xf0 + i)
	}
	d.Max = 0x10

	mem.PutKey(0x4000, 0x30)
	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x420)
	mem.SetMemory(0x48, 0x30000500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 00f
	mem.SetMemory(0x404, 0x82000410) // LPSW 0410
	mem.SetMemory(0x408, 0x47000408) // Dummy instruction
	mem.SetMemory(0x410, 0xff060000) // Wait PSW
	mem.SetMemory(0x414, 0x14000404)
	mem.SetMemory(0x420, 0x9d00000f) // TIO 00f
	mem.SetMemory(0x424, 0x47700420) // BC  7,420

	mem.SetMemory(0x500, 0x02004000) // Set channel words
	mem.SetMemory(0x504, 0x00000010)
	mem.SetMemory(0x508, 0)
	mem.SetMemory(0x50c, 0)
	// Load memory with value not equal to read data.
	for i := uint32(0x4000); i < 0x4040; i += 4 {
		mem.SetMemory(i, 0x55555555)
	}

	cpuState.iotestInst(0, 2000)

	v = mem.GetMemory(0x40)
	if v != 0x30000508 {
		t.Errorf("Start I/O Write Prot  CSW1 expected %08x got: %08x", 0x30000508, v)
	}
	v = mem.GetMemory(0x44) & HMASK
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
	mem.PutKey(0x4000, 0x0)
}

func TestCycleBusy(t *testing.T) {
	var v uint32

	d := ioSetup()
	d.Max = 0x10

	mem.SetMemory(0x40, 0xffffffff)
	mem.SetMemory(0x44, 0xffffffff)
	mem.SetMemory(0x78, 0)
	mem.SetMemory(0x7c, 0x430)
	mem.SetMemory(0x48, 0x00000500)

	mem.SetMemory(0x400, 0x9c00000f) // SIO 0f
	mem.SetMemory(0x404, 0x82000410) // LPSW 410
	mem.SetMemory(0x410, 0xff060000) // Wait state PSW
	mem.SetMemory(0x414, 0x12000404)

	mem.SetMemory(0x420, 0x9d00000f) // TIO 0f
	mem.SetMemory(0x424, 0x47700420) // BC 7,420

	mem.SetMemory(0x430, 0x58100040) // L 1,40
	mem.SetMemory(0x434, 0x58200044) // L 2,44
	mem.SetMemory(0x438, 0x41300448) // LA 3,448
	mem.SetMemory(0x43c, 0x5030007c) // ST 3,7c  Adjust address
	mem.SetMemory(0x440, 0x50300040) // ST 3,78  Overwrite csw
	mem.SetMemory(0x444, 0x82000410) // Wait some more
	mem.SetMemory(0x448, 0x58400040) // L 4,40
	mem.SetMemory(0x44c, 0x58500044) // L 5,44
	mem.SetMemory(0x450, 0x47f00420) // BC F,420 Wait for device

	mem.SetMemory(0x500, 0x03000600) // Set channel words
	mem.SetMemory(0x504, 0x60000001) // NOP
	mem.SetMemory(0x508, 0x13000520) // Chan end without data end NOP
	mem.SetMemory(0x50c, 0x60000001)
	mem.SetMemory(0x510, 0x13000540) // Chan end without data end NOP
	mem.SetMemory(0x514, 0x20000001)

	cpuState.iotestInst(0, 2000)

	v = cpuState.regs[1]
	if v != 0x00000518 {
		t.Errorf("Start I/O Busy Reg 1 expected %08x got: %08x", 0x00000518, v)
	}
	v = cpuState.regs[2]
	if v != 0x08000001 {
		t.Errorf("Start I/O Busy Reg 2 expected %08x got: %08x", 0x08000001, v)
	}
	v = cpuState.regs[4]
	if v != 0x00000000 {
		t.Errorf("Start I/O Busy Reg 4 expected %08x got: %08x", 0x00000000, v)
	}
	v = cpuState.regs[5]
	if v != 0x04000000 {
		t.Errorf("Start I/O Busy Reg 5 expected %08x got: %08x", 0x04000000, v)
	}

	v = mem.GetMemory(0x40) & HMASK
	if v != 0x00000000 {
		t.Errorf("Start I/O Busy CSW1 expected %08x got: %08x", 0x00000000, v)
	}

	v = mem.GetMemory(0x44) & HMASK
	if v != 0x04000000 {
		t.Errorf("Start I/O Busy CSW2 expected %08x got: %08x", 0x04000000, v)
	}
}
