/* Core S370 emulator loop.

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

package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/rcornwell/S370/emu/cpu"
	"github.com/rcornwell/S370/emu/event"
	"github.com/rcornwell/S370/emu/master"
	syschannel "github.com/rcornwell/S370/emu/sys_channel"
)

type core struct {
	wg       sync.WaitGroup
	shutdown bool // Signal to shutdown simulator.
	running  bool // Indicate when simulator should run or not.
	master   chan master.Packet
}

func (core *core) Start() {
	core.wg.Add(1)
	cpu.InitializeCPU()
	for !core.shutdown {
		if core.running {
			var cycle int
			cycle, core.running = cpu.CycleCPU()
			event.Advance(cycle)
		} else if event.AnyEvent() {
			event.Advance(1)
		}
		select {
		case packet := <-core.master:
			core.processPacket(packet)
		default:
		}
	}
}

func NewCPU(master chan master.Packet) *core {
	return &core{
		master: master,
	}
}

// Stop a running server.
func (core *core) Stop() {
	core.shutdown = true
	done := make(chan struct{})
	go func() {
		core.wg.Wait()
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

// Process a packet sent to system simulation.
func (core *core) processPacket(packet master.Packet) {
	switch packet.Msg {
	case master.TelConnect:
		syschannel.SendConnect(packet.DevNum, packet.Conn)
	case master.TelDisconnect:
		syschannel.SendDisconnect(packet.DevNum)
	case master.TelReceive:
		syschannel.SendReceiveChar(packet.DevNum, packet.Data)
	case master.TimeClock:
	case master.IPLdevice:
		cpu.IPLDevice(packet.DevNum)
		core.running = true
	case master.Start:
		core.running = true
	case master.Stop:
		core.running = false
	case master.Shutdown:
		core.shutdown = true
	}
}

func PostExtIrq() {
	cpu.PostExtIrq()
}
