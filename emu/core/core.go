/*
   Core S370 emulator loop.

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
	"log/slog"
	"sync"
	"time"

	cpu "github.com/rcornwell/S370/emu/cpu"
	"github.com/rcornwell/S370/emu/event"
	"github.com/rcornwell/S370/emu/master"
	syschannel "github.com/rcornwell/S370/emu/sys_channel"
)

type core struct {
	wg      sync.WaitGroup
	done    chan struct{} // Signal to shutdown simulator.
	running bool          // Indicate when simulator should run or not.
	master  chan master.Packet
}

// Create instance of CPU.
func NewCPU(master chan master.Packet) *core {
	return &core{
		master: master,
		done:   make(chan struct{}),
	}
}

// Start CPU running.
func (core *core) Start() {
	core.wg.Add(1)
	defer core.wg.Done()
	cpu.InitializeCPU()
	cpu.SetTod()
	for {
		if core.running {
			var cycle int
			cycle, core.running = cpu.CycleCPU()
			event.Advance(cycle)
		} else if event.AnyEvent() {
			event.Advance(1)
		}
		select {
		case <-core.done:
			// Shutdone all devices.
			cpu.Shutdown()
			slog.Info("Shutdown CPU core")
			return
		case packet := <-core.master:
			core.processPacket(packet)
		default:
		}
	}
}

// Stop a running server.
func (core *core) Stop() {
	close(core.done)
	done := make(chan struct{})
	go func() {
		core.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		slog.Warn("Timed out waiting for CPU to finish.")
		return
	}
}

// Post an external interrupt to CPU.
func PostExtIrq() {
	cpu.PostExtIrq()
}

// Get IPL line option.
func IPLDevice() uint16 {
	return cpu.IPLDev
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
		cpu.UpdateTimer()
	case master.IPLdevice:
		err := cpu.IPLDevice(packet.DevNum)
		if err != nil {
			slog.Error(err.Error())
		} else {
			core.running = true
		}
	case master.Start:
		core.running = true
	case master.Stop:
		core.running = false
	}
}
