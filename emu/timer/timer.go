/*
   S370 IBM 370 Regular timer event.

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
	"log/slog"
	"sync"
	"time"

	"github.com/rcornwell/S370/emu/master"
)

type Timer struct {
	wg      sync.WaitGroup
	running bool // Indicate when simulator should run or not.
	master  chan master.Packet
	enable  chan bool     // Enable or disable timer.
	done    chan struct{} // Stop timer task.
	ticker  *time.Ticker  // Regular timer intervale.
}

// Create instance of Clock timer.
func NewTimer(masterChannel chan master.Packet) *Timer {
	timer := &Timer{
		master:  masterChannel,
		running: false,
		enable:  make(chan bool, 1),
		done:    make(chan struct{}),
	}
	// Run ticker to deliver regular commands on master channel.
	timer.wg.Add(1)
	go timer.run()
	return timer
}

// Start timer process to deliver 5ms clock pulses.
func (timer *Timer) Start() {
	timer.enable <- true
}

// Stop a timer for some time.
func (timer *Timer) Stop() {
	timer.enable <- false
}

// Shutdown a running server.
func (timer *Timer) Shutdown() {
	close(timer.done)
	done := make(chan struct{})
	go func() {
		timer.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		slog.Warn("Timed out waiting for timer to finish.")
		return
	}
}

// Internval timer routine to send timer events on master channel.
func (timer *Timer) run() {
	defer timer.wg.Done()
	timer.ticker = time.NewTicker(6666666 * time.Nanosecond)
	defer timer.ticker.Stop()
	timer.running = false

	for {
		select {
		case <-timer.ticker.C:
			if timer.running {
				timer.master <- master.Packet{Msg: master.TimeClock}
			}
		case timer.running = <-timer.enable:
			if timer.running {
				timer.ticker.Reset(6666666 * time.Nanosecond)
			}
		case <-timer.done:
			return
		}
	}
}
