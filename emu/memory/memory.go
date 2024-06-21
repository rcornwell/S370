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

type mem struct {
	mem  [4 * 1024 * 1024]uint32
	key  [8192]uint8
	size uint32
}

var Mem mem

const (
	AMASK uint32 = 0x00ffffff // Mask address bits
	HMASK uint32 = 0x0000ffff // Half word mask
	UMASK uint32 = 0xffff0000 // Upper half word mask
	PMASK uint32 = 0xf0000000 // Storage protection mask
	FMASK uint32 = 0xffffffff // Full word mask
)

// Set size in K
func SetSize(k int) {
	if k > (16 * 1024) {
		k = 16 * 1024
	}
	Mem.size = uint32(k * 1024)
}

// Get memory value without range check
func GetMemory(addr uint32) uint32 {
	Mem.key[addr>>11] |= 0x4 // Update access bits
	return Mem.mem[addr>>2]
}

// Set memory to a value, without range check
func SetMemory(addr, data uint32) {
	Mem.key[addr>>11] |= 0x6 // Update Access and modify bits
	Mem.mem[addr>>2] = data
}

// Set memory to a value, without range check
func SetMemoryMask(addr uint32, data uint32, mask uint32) {
	Mem.key[addr>>11] |= 0x6 // Update Access and modify bits
	addr >>= 2
	Mem.mem[addr] &= ^mask
	Mem.mem[addr] |= data & mask
}

// Check if address out of range
func CheckAddr(addr uint32) bool {
	return addr < Mem.size
}

// Get a word from memory
func GetWord(addr uint32) (value uint32, error bool) {
	if addr >= Mem.size {
		return 0, true
	}
	Mem.key[addr>>11] |= 0x4 // Update Access bits
	return Mem.mem[addr>>2], false
}

// Put a word to memory
func PutWord(addr, data uint32) bool {
	if addr >= Mem.size {
		return true
	}
	Mem.key[addr>>11] |= 0x6 // Update Access and modify bits
	Mem.mem[addr>>2] = data
	return false
}

// Put a word to memory, under mask
func PutWordMask(addr, data, mask uint32) bool {
	if addr >= Mem.size {
		return true
	}
	Mem.key[addr>>11] |= 0x6 // Update Access and modify bits
	addr >>= 2
	Mem.mem[addr] &= ^mask
	Mem.mem[addr] |= data & mask
	return false
}

func GetKey(addr uint32) uint8 {
	if addr >= Mem.size {
		return 0
	}
	return Mem.key[addr>>11]
}

func PutKey(addr uint32, key uint8) {
	if addr < Mem.size {
		Mem.key[addr>>11] = key
	}
}
