package memory

/*
 * S370  - Low level memory
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
)

// Set size in K.
func TestSetSize(t *testing.T) {
	for i := range 32 {
		SetSize(i)
		r := memory.size
		if i > (16 * 1024) {
			if r != (16 * 1024) {
				t.Errorf("Memory size not correct got: %d expected: %d", r, 16*1024)
			}
		} else {
			if r != uint32(i*1024) {
				t.Errorf("Memory size not correct got: %d expected: %d", r, i*1024)
			}
		}

		r = GetSize()
		if i > (16 * 1024) {
			if r != (16 * 1024) {
				t.Errorf("GetSize size not correct got: %d expected: %d", r, 16*1024)
			}
		} else {
			if r != uint32(i*1024) {
				t.Errorf("GetSize size not correct got: %d expected: %d", r, i*1024)
			}
		}
	}
}

// Check get memory.
func TestGetMemory(t *testing.T) {
	memory.size = 2048
	for i := range uint32(256) {
		memory.mem[i] = i
	}
	memory.mem[4096>>2] = 0xffffffff
	memory.key[0] = 0xf0
	memory.key[1] = 0xe0
	for i := range uint32(256) {
		j := i * 4
		r := GetMemory(j)
		if r != i {
			t.Errorf("GetMemory not correct got: %d expected: %d", r, i)
		}
	}
	for i := range uint32(256) {
		j := i * 4
		r := GetMemory(j + 1024)
		if r != 0 {
			t.Errorf("GetMemory not correct got: %d expected: %d", r, 0)
		}
	}
	k := memory.key[0]
	if k != 0xf4 {
		t.Errorf("GetMemory Key 0 not updated got: %02x expected: %02x", k, 0xf4)
	}
	k = memory.key[1]
	if k != 0xe0 {
		t.Errorf("GetMemory Key 1 updated got: %02x expected: %02x", k, 0xe0)
	}
	// Check if over memory size.
	r := GetMemory(4096)
	if r != 0xffffffff {
		t.Errorf("GetMemory not correct got: %d expected: %d", r, 0xffffffff)
	}
}

// Check get memory.
func TestSetMemory(t *testing.T) {
	memory.size = 2048
	for i := range uint32(256) {
		memory.mem[i] = i
		memory.mem[i+256] = 0
	}
	memory.mem[4096>>2] = 0xffffffff
	memory.key[0] = 0xf0
	memory.key[1] = 0xe0
	for i := range uint32(256) {
		j := i * 4
		SetMemory(j, 2048-i)
	}

	for i := range uint32(256) {
		j := i * 4
		r := GetMemory(j)
		if r != 2048-i {
			t.Errorf("GetMemory not correct got: %d expected: %d", r, i)
		}
	}
	for i := range uint32(256) {
		j := i * 4
		r := GetMemory(j + 1024)
		if r != 0 {
			t.Errorf("GetMemory not correct got: %d expected: %d", r, 0)
		}
	}
	k := memory.key[0]
	if k != 0xf6 {
		t.Errorf("GetMemory Key 0 not updated got: %02x expected: %02x", k, 0xf6)
	}
	k = memory.key[1]
	if k != 0xe0 {
		t.Errorf("GetMemory Key 1 updated got: %02x expected: %02x", k, 0xe0)
	}
	// Check if over memory size.
	SetMemory(4096, 0x0)
	r := GetMemory(4096)
	if r != 0 {
		t.Errorf("GetMemory not correct got: %d expected: %d", r, 0)
	}
}

// Check set memory under mask.
func TestSetMemoryMask(t *testing.T) {
	memory.size = 2048
	for i := range uint32(256) {
		memory.mem[i] = 0xffffffff
		memory.mem[i+256] = 0
	}
	memory.mem[4096>>2] = 0xffffffff
	memory.key[0] = 0xf0
	memory.key[1] = 0xe0
	for i := range uint32(256) {
		j := i * 4
		m := uint32(0xff) << (8 * (i & 3))
		SetMemoryMask(j, 0x12345678, m)
	}
	for i := range uint32(256) {
		j := i * 4
		m := ^(uint32(0xff) << (8 * (i & 3)))
		v := 0x12345678 | m
		r := GetMemory(j)
		if r != v {
			t.Errorf("GetMemory not correct got: %08x expected: %08x", r, v)
		}
	}
	for i := range uint32(256) {
		j := i * 4
		r := GetMemory(j + 1024)
		if r != 0 {
			t.Errorf("GetMemory not correct got: %d expected: %d", r, 0)
		}
	}
	k := memory.key[0]
	if k != 0xf6 {
		t.Errorf("GetMemory Key 0 not updated got: %02x expected: %02x", k, 0xf6)
	}
	k = memory.key[1]
	if k != 0xe0 {
		t.Errorf("GetMemory Key 1 updated got: %02x expected: %02x", k, 0xe0)
	}
	// Check if over memory size.
	SetMemoryMask(4096, 0x0, 0xff0000ff)
	r := GetMemory(4096)
	if r != 0x00ffff00 {
		t.Errorf("GetMemory not correct got: %08x expected: %08x", r, 0x00ffff00)
	}
}

// Check get memory word.
func TestGetWprd(t *testing.T) {
	memory.size = 2048
	for i := range uint32(256) {
		memory.mem[i] = i
	}
	memory.mem[4096>>2] = 0xffffffff
	memory.key[0] = 0xf0
	memory.key[1] = 0xe0
	for i := range uint32(256) {
		j := i * 4
		r, e := GetWord(j)
		if r != i {
			t.Errorf("GetWord not correct got: %d expected: %d", r, i)
		}
		if e {
			t.Errorf("GetWord got error %d", j)
		}
	}
	for i := range uint32(256) {
		j := i * 4
		r, e := GetWord(j + 1024)
		if r != 0 {
			t.Errorf("GetWord not correct got: %d expected: %d", r, 0)
		}
		if e {
			t.Errorf("GetWord got error %d", j)
		}
	}
	k := memory.key[0]
	if k != 0xf4 {
		t.Errorf("GetWord Key 0 not updated got: %02x expected: %02x", k, 0xf4)
	}
	k = memory.key[1]
	if k != 0xe0 {
		t.Errorf("GetWord Key 1 updated got: %02x expected: %02x", k, 0xe0)
	}
	// Check if over memory size.
	r, e := GetWord(4096)
	if r != 0 {
		t.Errorf("GetMemory not correct got: %d expected: %d", r, 0)
	}
	if !e {
		t.Errorf("GetWord got did not get error 4096")
	}
}

// Check set memory word.
func TestPutWord(t *testing.T) {
	memory.size = 2048
	for i := range uint32(256) {
		memory.mem[i] = i
		memory.mem[i+256] = 0
	}
	memory.mem[4096>>2] = 0xffffffff
	memory.key[0] = 0xf0
	memory.key[1] = 0xe0
	for i := range uint32(256) {
		j := i * 4
		e := PutWord(j, 2048-i)
		if e {
			t.Errorf("PutWord got error %d", j)
		}
	}

	for i := range uint32(256) {
		j := i * 4
		r := GetMemory(j)
		if r != 2048-i {
			t.Errorf("PutWord not correct got: %d expected: %d", r, i)
		}
	}
	for i := range uint32(256) {
		j := i * 4
		r := GetMemory(j + 1024)
		if r != 0 {
			t.Errorf("PutWord not correct got: %d expected: %d", r, 0)
		}
	}
	k := memory.key[0]
	if k != 0xf6 {
		t.Errorf("PutWord Key 0 not updated got: %02x expected: %02x", k, 0xf6)
	}
	k = memory.key[1]
	if k != 0xe0 {
		t.Errorf("PutWord Key 1 updated got: %02x expected: %02x", k, 0xe0)
	}
	// Check if over memory size.
	e := PutWord(4096, 0x0)
	r := GetMemory(4096)
	if !e {
		t.Errorf("GetWord got did not get error 4096")
	}
	if r != 0xffffffff {
		t.Errorf("PutWord modified above memory got: %d expected: %d", r, 0xffffffff)
	}
}

// Check set memory under mask.
func TestPutWordMask(t *testing.T) {
	memory.size = 2048
	for i := range uint32(256) {
		memory.mem[i] = 0xffffffff
		memory.mem[i+256] = 0
	}
	memory.mem[4096>>2] = 0xffffffff
	memory.key[0] = 0xf0
	memory.key[1] = 0xe0
	for i := range uint32(256) {
		j := i * 4
		m := uint32(0xff) << (8 * (i & 3))
		e := PutWordMask(j, 0x12345678, m)
		if e {
			t.Errorf("PutWordMask got error %d", j)
		}
	}
	for i := range uint32(256) {
		j := i * 4
		m := ^(uint32(0xff) << (8 * (i & 3)))
		v := 0x12345678 | m
		r := GetMemory(j)
		if r != v {
			t.Errorf("PutWordMask not correct got: %08x expected: %08x", r, v)
		}
	}
	for i := range uint32(256) {
		j := i * 4
		r := GetMemory(j + 1024)
		if r != 0 {
			t.Errorf("PutWordMask not correct got: %d expected: %d", r, 0)
		}
	}
	k := memory.key[0]
	if k != 0xf6 {
		t.Errorf("PutWordMask Key 0 not updated got: %02x expected: %02x", k, 0xf6)
	}
	k = memory.key[1]
	if k != 0xe0 {
		t.Errorf("PutWordMask Key 1 updated got: %02x expected: %02x", k, 0xe0)
	}
	// Check if over memory size.
	e := PutWordMask(4096, 0x0, 0xff0000ff)
	r := GetMemory(4096)
	if !e {
		t.Errorf("PutWordMask got did not get error 4096")
	}
	if r != 0xffffffff {
		t.Errorf("PutWordMask modified above memory got: %d expected: %d", r, 0xffffffff)
	}
}

// Check get memory.
func TestCheckAddr(t *testing.T) {
	memory.size = 2048

	if !CheckAddr(1024) {
		t.Errorf("CheckAddr return error below memory size")
	}
	if CheckAddr(2048) {
		t.Errorf("CheckAddr did not return error at memory size")
	}
	if CheckAddr(4096) {
		t.Errorf("CheckAddr did not return error above memory size")
	}
}

// Check set memory word.
func TestGetKey(t *testing.T) {
	memory.size = 4096
	for i := range uint32(2048) {
		memory.mem[i] = i
	}
	memory.key[0] = 0xf0
	memory.key[1] = 0xe0
	memory.key[2] = 0xa0
	memory.key[3] = 0xb0
	for i := range uint32(8192 / 4) {
		j := 4 * i
		k := GetKey(j)
		if j < 2048 {
			if k != 0xf0 {
				t.Errorf("GetKey Key 0 %d got: %02x expected: %02x", j, k, 0xf0)
			}
		} else if j < 4096 {
			if k != 0xe0 {
				t.Errorf("GetKey Key 1 %d got: %02x expected: %02x", j, k, 0xe0)
			}
		} else {
			if k != 0 {
				t.Errorf("GetKey Key 3 %d got: %02x expected: %02x", j, k, 0x00)
			}
		}
	}
	for i := range uint32(8192 / 4) {
		j := 4 * i
		r := GetMemory(j)
		if i < 2048 {
			if r != i {
				t.Errorf("GetKey modified memory got: %08x expected: %08x", r, i)
			}
		} else {
			if r != 0 {
				t.Errorf("GetKey modified memory got: %08x expected: %08x", r, 0)
			}
		}
	}
}

func TestPutKey(t *testing.T) {
	memory.size = 4096
	for i := range uint32(2048) {
		memory.mem[i] = i
	}
	memory.key[0] = 0x00
	memory.key[1] = 0x00
	memory.key[2] = 0x00
	memory.key[3] = 0x00

	for i := range uint32(8192 / 4) {
		j := 4 * i
		k := uint8(0xf0 - ((j / 2048) * 0x10))
		PutKey(j, k)
	}

	for i := range uint32(8192 / 4) {
		j := 4 * i
		k := GetKey(j)
		if j < 2048 {
			if k != 0xf0 {
				t.Errorf("PutKey Key 0 %d got: %02x expected: %02x", j, k, 0xf0)
			}
		} else if j < 4096 {
			if k != 0xe0 {
				t.Errorf("PutKey Key 1 %d got: %02x expected: %02x", j, k, 0xe0)
			}
		} else {
			if k != 0 {
				t.Errorf("PutKey Key 3 %d got: %02x expected: %02x", j, k, 0x00)
			}
		}
	}
	for i := range uint32(8192 / 4) {
		j := 4 * i
		r := GetMemory(j)
		if i < 2048 {
			if r != i {
				t.Errorf("PutKey modified memory got: %08x expected: %08x", r, i)
			}
		} else {
			if r != 0 {
				t.Errorf("PutKey modified memory got: %08x expected: %08x", r, 0)
			}
		}
	}
}
