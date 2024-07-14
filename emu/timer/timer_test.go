/*
   S370 IBM 370 Regular timer event test.

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
   RICHARD CORNWELL BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
   IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

*/

package timer

import (
	"testing"
	"time"

	"github.com/rcornwell/S370/emu/master"
)

type timerTest struct {
	timer   *Timer        // Timer device.
	done    chan struct{} // Stop routine.
	counter int
}

// Test function to receive timer ticks.
func (test *timerTest) runTimer(t *testing.T) {
	test.counter = 0
	for {
		select {
		case v := <-test.timer.master:
			if v.Msg != master.TimeClock {
				t.Errorf("Did not receive correct message from timer: %d", v.Msg)
				return
			}
			test.counter++
		case <-test.done:
			break
		}
	}
}

// Debug interval timer.
func TestTimer(t *testing.T) {
	// Create a new interval timer.
	masterChannel := make(chan master.Packet)
	timer := NewTimer(masterChannel)

	test := timerTest{
		timer:   timer,
		done:    make(chan struct{}),
		counter: 0,
	}

	defer close(test.done)

	// Start test listener
	go test.runTimer(t)

	// Start timer and wait for 1/2 second and make sure count is correct.
	timer.Start()
	time.Sleep(time.Second)
	if test.counter < 148 || test.counter > 152 {
		t.Errorf("Expected 75 ticks during a second got: %d", test.counter)
	}

	// Stop timer and wait for 1/2 second and make sure no events sent
	timer.Stop()
	test.counter = 0
	time.Sleep(505 * time.Millisecond)

	if test.counter != 0 {
		t.Errorf("Expected 0 ticks during a second got: %d", test.counter)
	}

	// Restart timer and wait 1/2 second and make sure correct number of events.
	test.counter = 0
	timer.Start()
	time.Sleep(505 * time.Millisecond)

	if test.counter < 74 || test.counter > 75 {
		t.Errorf("Expected 75 ticks during a second got: %d", test.counter)
	}
	timer.Stop()

	// Lastly run for 2 seconds and verify count correct. 150 pulses per second.
	test.counter = 0
	timer.Start()
	time.Sleep(2 * time.Second)
	if test.counter < 299 || test.counter > 301 {
		t.Errorf("Expected 300 ticks during a second got: %d", test.counter)
	}
	timer.Shutdown()
}
