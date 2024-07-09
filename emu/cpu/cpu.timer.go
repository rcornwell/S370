/*
   CPU timer update routines.

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
   ROBERT M SUPNIK BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
   IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

*/

package cpu

import (
	"time"

	mem "github.com/rcornwell/S370/emu/memory"
)

// Update current interval timer and TOD clock.
func UpdateTimer() {
	timeMem := mem.GetMemory(0x50)
	timeMem -= 0x100
	mem.SetMemory(0x50, timeMem)

	// Check if should signal CPU
	if (timeMem & 0xfffff00) == 0 {
		cpuState.intIrq = true
	}

	// Update TOD clock if enabled.
	if cpuState.todSet && (cpuState.cregs[0]&0x20000000) == 0 {
		t := cpuState.todClock[1] + (13333333)
		if t < cpuState.todClock[1] {
			cpuState.todClock[0]++
		}
		cpuState.todClock[1] = t

		// Check if we should post a TOD irq
		cpuState.todIrq = false
		if (cpuState.clkCmp[0] < cpuState.todClock[0]) ||
			((cpuState.clkCmp[0] == cpuState.todClock[0]) && (cpuState.clkCmp[1] < cpuState.todClock[1])) {
			//     sim_debug(DEBUG_INST, &cpu_dev, "CPU TIMER CCK IRQ %08x %08x\n", clk_cmp[0],
			//               clk_cmp[1]);
			cpuState.todIrq = true
		}
	}

	// Update CPU timer.
	t := cpuState.cpuTimer[1] - (uint32(cpuState.timerTics) << 12)
	if t > cpuState.cpuTimer[1] {
		cpuState.cpuTimer[0]--
	}
	cpuState.cpuTimer[1] = t
	cpuState.timerTics = 3333
	if (cpuState.cpuTimer[0] & MSIGN) != 0 {
		cpuState.clkIrq = true
	}
}

// Set TOD to current date.
func SetTod() {
	if !cpuState.todSet {
		// Get current time
		now := time.Now()
		lsec := uint64(now.Unix())

		// IBM measures time from 1900, Unix starts at 1970
		// Add in number of years from 1900 to 1970 + 17 leap days
		lsec += ((70 * 365) + 17) * 86400
		lsec *= 1000000
		lsec <<= 12
		cpuState.todClock[0] = uint32(lsec >> 32)
		cpuState.todClock[1] = uint32(lsec & uint64(FMASK))
	}
}
