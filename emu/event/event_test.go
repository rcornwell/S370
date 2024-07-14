/*
 * S370 - Event system test cases.
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

package event

import (
	"testing"

	dev "github.com/rcornwell/S370/emu/device"
)

var stepCount uint64

type device struct {
	iarg int
	time uint64
}

var (
	deviceA device
	deviceB device
	deviceC device
	deviceD device
)

// Callbacks, save step count in routine time and set argument to iarg.
func (d *device) aCallback(iarg int) {
	d.iarg = iarg
	d.time = stepCount
}

// Callbacks, save step count in routine time and set argument to iarg.
func (d *device) bCallback(iarg int) {
	d.iarg = iarg
	d.time = stepCount
}

// Callbacks, save step count in routine time and set argument to iarg.
func (d *device) cCallback(iarg int) {
	d.iarg = iarg
	d.time = stepCount
	AddEvent(deviceA, deviceA.aCallback, iarg, iarg)
}

// Callbacks, save step count in routine time and set argument to iarg.
func (d *device) dCallback(iarg int) {
	d.iarg = iarg
	d.time = stepCount
}

func (d device) StartIO() uint8 {
	return 0
}

func (d device) StartCmd(_ uint8) uint8 {
	return 0
}

func (d device) HaltIO() uint8 {
	return 0
}

func (d device) InitDev() uint8 {
	return 0
}

// Attach file to device.
func (d device) Attach(_ []dev.CmdOption) error {
	return nil
}

// Detach device.
func (d device) Detach() error {
	return nil
}

// Set command.
func (d device) Set(_ []dev.CmdOption) error {
	return nil
}

// Show command.
func (d device) Show(_ []dev.CmdOption) error {
	return nil
}

// Shutdown device.
func (d device) Shutdown() {
}

// Initialize for each test.
func initTest() {
	stepCount = 0
	deviceA.time = 0
	deviceB.time = 0
	deviceC.time = 0
	deviceD.time = 0
	deviceA.iarg = 0
	deviceB.iarg = 0
	deviceC.iarg = 0
	deviceD.iarg = 0
}

func TestAddEvent1(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 10, 1)
	for range 20 {
		stepCount++
		Advance(1)
	}
	if deviceA.time != 10 {
		t.Errorf("Event did not fire at correct time %d got %d", 10, deviceA.time)
	}
	if deviceA.iarg != 1 {
		t.Errorf("Event did not set data correct %d got %d", 1, deviceA.iarg)
	}
}

// Add two events.
func TestAddEvent2(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 10, 1)
	AddEvent(deviceB, deviceB.bCallback, 5, 2)
	for range 20 {
		stepCount++
		Advance(1)
	}
	if deviceA.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, deviceA.time)
	}
	if deviceA.iarg != 1 {
		t.Errorf("Event A did not set data correct %d got %d", 1, deviceA.iarg)
	}
	if deviceB.time != 5 {
		t.Errorf("Event B did not fire at correct time %d got %d", 5, deviceB.time)
	}
	if deviceB.iarg != 2 {
		t.Errorf("Event B did not set data correct %d got %d", 2, deviceB.iarg)
	}
}

// Add two events.
func TestAddEvent2a(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 10, 1)
	AddEvent(deviceB, deviceB.aCallback, 5, 2)
	for range 20 {
		stepCount++
		Advance(1)
	}
	if deviceA.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, deviceA.time)
	}
	if deviceA.iarg != 1 {
		t.Errorf("Event A did not set data correct %d got %d", 1, deviceA.iarg)
	}
	if deviceB.time != 5 {
		t.Errorf("Event B did not fire at correct time %d got %d", 5, deviceB.time)
	}
	if deviceB.iarg != 2 {
		t.Errorf("Event B did not set data correct %d got %d", 2, deviceB.iarg)
	}
}

// Add event With same time.
func TestAddEvent3(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 10, 1)
	AddEvent(deviceB, deviceB.bCallback, 10, 2)
	for range 20 {
		stepCount++
		Advance(1)
	}
	if deviceA.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, deviceA.time)
	}
	if deviceA.iarg != 1 {
		t.Errorf("Event A did not set data correct %d got %d", 1, deviceA.iarg)
	}
	if deviceB.time != 10 {
		t.Errorf("Event B did not fire at correct time %d got %d", 10, deviceB.time)
	}
	if deviceB.iarg != 2 {
		t.Errorf("Event B did not set data correct %d got %d", 2, deviceB.iarg)
	}
}

// Add event during event.
func TestAddEvent4(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 20, 5)
	AddEvent(deviceC, deviceC.cCallback, 10, 2)
	for range 30 {
		stepCount++
		Advance(1)
	}
	if deviceA.time != 20 {
		t.Errorf("Event A did not fire at correct time %d got %d", 20, deviceA.time)
	}
	if deviceA.iarg != 5 {
		t.Errorf("Event A did not set data correct %d got %d", 5, deviceA.iarg)
	}
	if deviceC.time != 10 {
		t.Errorf("Event C did not fire at correct time %d got %d", 10, deviceC.time)
	}
	if deviceC.iarg != 2 {
		t.Errorf("Event C did not set data correct %d got %d", 2, deviceC.iarg)
	}
}

// Schedule 3 events, last one before first, make sure all are correct.
func TestAddEvent5(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 20, 1)
	AddEvent(deviceB, deviceB.bCallback, 20, 2)
	AddEvent(deviceD, deviceD.dCallback, 25, 3)
	for range 30 {
		stepCount++
		Advance(1)
	}
	if deviceA.time != 20 {
		t.Errorf("Event A did not fire at correct time %d got %d", 20, deviceA.time)
	}
	if deviceA.iarg != 1 {
		t.Errorf("Event A did not set data correct %d got %d", 1, deviceA.iarg)
	}
	if deviceB.time != 20 {
		t.Errorf("Event B did not fire at correct time %d got %d", 20, deviceB.time)
	}
	if deviceB.iarg != 2 {
		t.Errorf("Event B did not set data correct %d got %d", 2, deviceB.iarg)
	}
	if deviceD.time != 25 {
		t.Errorf("Event D did not fire at correct time %d got %d", 25, deviceD.time)
	}
	if deviceD.iarg != 3 {
		t.Errorf("Event D did not set data correct %d got %d", 3, deviceD.iarg)
	}
}

// Cancel an event.
func TestAddEvent6(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 10, 5)
	AddEvent(deviceB, deviceB.bCallback, 20, 2)
	for range 30 {
		stepCount++
		Advance(1)
		if deviceA.iarg == 5 {
			CancelEvent(deviceB, 2)
		}
	}
	if deviceA.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, deviceA.time)
	}
	if deviceA.iarg != 5 {
		t.Errorf("Event A did not set data correct %d got %d", 5, deviceA.iarg)
	}
	if deviceB.time != 0 {
		t.Errorf("Event D did not fire at correct time %d got %d", 0, deviceB.time)
	}
	if deviceB.iarg != 0 {
		t.Errorf("Event D did not set data correct %d got %d", 0, deviceB.iarg)
	}
}

// Schedule 3 events, cancel one while events in queue.
func TestAddEvent7(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 10, 5)
	AddEvent(deviceB, deviceB.bCallback, 20, 2)
	AddEvent(deviceD, deviceD.dCallback, 30, 3)
	for range 30 {
		stepCount++
		Advance(1)
		if deviceA.iarg == 5 {
			CancelEvent(deviceB, 2)
		}
	}
	if deviceA.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, deviceA.time)
	}
	if deviceA.iarg != 5 {
		t.Errorf("Event A did not set data correct %d got %d", 5, deviceA.iarg)
	}
	if deviceB.time != 0 {
		t.Errorf("Event B did not fire at correct time %d got %d", 0, deviceB.time)
	}
	if deviceB.iarg != 0 {
		t.Errorf("Event B did not set data correct %d got %d", 0, deviceB.iarg)
	}
	if deviceD.time != 30 {
		t.Errorf("Event D did not fire at correct time %d got %d", 30, deviceD.time)
	}
	if deviceD.iarg != 3 {
		t.Errorf("Event D did not set data correct %d got %d", 3, deviceD.iarg)
	}
}

// Schedule 4 events, cancel two while events in queue.
func TestAddEvent8(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 10, 5)
	AddEvent(deviceB, deviceB.bCallback, 40, 2)
	AddEvent(deviceD, deviceD.dCallback, 30, 3)
	AddEvent(deviceD, deviceD.dCallback, 50, 4)
	for range 60 {
		stepCount++
		Advance(1)
		if deviceA.iarg == 5 {
			CancelEvent(deviceB, 2)
			CancelEvent(deviceD, 4)
		}
	}
	if deviceA.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, deviceA.time)
	}
	if deviceA.iarg != 5 {
		t.Errorf("Event A did not set data correct %d got %d", 5, deviceA.iarg)
	}
	if deviceB.time != 0 {
		t.Errorf("Event B did not fire at correct time %d got %d", 0, deviceB.time)
	}
	if deviceB.iarg != 0 {
		t.Errorf("Event B did not set data correct %d got %d", 0, deviceB.iarg)
	}
	if deviceD.time != 30 {
		t.Errorf("Event D did not fire at correct time %d got %d", 30, deviceD.time)
	}
	if deviceD.iarg != 3 {
		t.Errorf("Event D did not set data correct %d got %d", 3, deviceD.iarg)
	}
}

// Test event at zero units.
func TestAddEvent9(t *testing.T) {
	initTest()
	AddEvent(deviceA, deviceA.aCallback, 0, 5)
	if deviceA.time != 0 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, deviceA.time)
	}
	if deviceA.iarg != 5 {
		t.Errorf("Event A did not set data correct %d got %d", 5, deviceA.iarg)
	}
}
