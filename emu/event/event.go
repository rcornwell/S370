package event

/*
 * S370  - Event scheduler
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
	D "github.com/rcornwell/S370/emu/device"
)

type Callback = func(iarg int)

type Event struct {
	time int      // Number of cycles to event
	dev  D.Device // Device event is registered too
	cb   Callback // Function to callback
	iarg int      // Integer argument
	prev *Event
	next *Event
}

type EventList struct {
	head *Event
	tail *Event
}

var el EventList

// Add an event
func AddEvent(dev D.Device, cb Callback, time int, iarg int) bool {

	// If time is 0 process event immediately
	if time == 0 {
		cb(iarg)
		return false
	}

	ev := &Event{dev: dev, cb: cb, time: time, iarg: iarg, next: nil, prev: nil}

	evptr := el.head
	// If empty put on head
	if evptr == nil {
		el.head = ev
		el.tail = ev
		return false
	}

	// Scan for place to install it
	for evptr != nil {
		// Event before next event
		if ev.time <= evptr.time {
			// Remove current time from next time
			evptr.time -= ev.time
			ev.prev = evptr.prev
			ev.next = evptr
			evptr.prev = ev
			if ev.prev != nil {
				ev.prev.next = ev
			} else {
				el.head = ev
			}
			// All done
			return false
		}
		// Make new event relative to head of list
		ev.time -= evptr.time
		evptr = evptr.next
	}

	// Get here, put it on tail of list
	ev.prev = el.tail
	el.tail.next = ev
	el.tail = ev
	return false
}

func CancelEvent(dev D.Device, cb Callback, iarg int) {
	evptr := el.head

	// Nothing in list, return
	if evptr == nil {
		return
	}

	// Scan list
	for evptr != nil {
		if evptr.dev == dev && evptr.iarg == iarg {
			nxt := evptr.next
			// If next event give time to next event
			if nxt != nil {
				nxt.time += evptr.time
				// Point next event to previous to current previous
				nxt.prev = evptr.prev
				// } else {
				// 	// No next event
			} else {
				// No next event, point event_tail to prev
				el.tail = evptr.prev
			}

			// Point previous event next to next
			if evptr.prev != nil {
				evptr.prev.next = evptr.next
			} else {
				// No previous, at head of list
				el.head = evptr.next
			}
			evptr = nil
			return
		}
		evptr = evptr.next
	}
}

// Advance time by one clock cycle
func Advance(t int) {
	evptr := el.head
	if evptr == nil {
		return
	}
	evptr.time -= t
	for evptr != nil && evptr.time <= 0 {
		evptr.cb(evptr.iarg)
		el.head = evptr.next
		evptr = nil
		evptr = el.head
		if evptr != nil {
			evptr.prev = nil
		} else {
			el.tail = nil
		}
	}
}
