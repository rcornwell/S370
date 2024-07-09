/* S370 IBM 370 Regular timer event.

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
	"fmt"
	"sync"
	"time"

	"github.com/rcornwell/S370/emu/master"
)

type timer struct {
	wg       sync.WaitGroup
	shutdown bool // Signal to shutdown simulator.
	running  bool // Indicate when simulator should run or not.
	master   chan master.Packet
	enable   chan bool    // Enable or disable timer.
	ticker   *time.Ticker // Regular timer intervale.
}

// Create instance of Clock timer.
func NewTimer(master chan master.Packet) *timer {
	timer := &timer{
		master: master,
	}
	timer.ticker = time.NewTicker(20 * time.Microsecond)
	timer.ticker.Stop()
	return timer
}

// Start timer process to deliver 20ms clock pulses.
func (timer *timer) Start() {
	timer.wg.Add(1)
	defer timer.ticker.Stop()
	for !timer.shutdown {
		select {
		case <-timer.ticker.C:
			timer.master <- master.Packet{Msg: master.TimeClock}
		case timer.running = <-timer.enable:
			if timer.running {
				timer.ticker.Reset(20 * time.Millisecond)
			} else {
				timer.ticker.Stop()
			}
		default:
		}
	}
}

// Stop a running server.
func (timer *timer) Stop() {
	timer.shutdown = true
	done := make(chan struct{})
	go func() {
		timer.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		fmt.Println("Timed out waiting for connections to finish.")
		return
	}
}
