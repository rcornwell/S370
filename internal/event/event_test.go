package event

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

import (
	"testing"
)

//  #include "ctest.h"
//  #include "device.h"
//  #include "event.h"

//  uint64_t   step_count;

//  int        a_time;
//  int        b_time;
//  int        c_time;
//  int        d_time;

//  int        a_data;
//  int        b_data;
//  int        c_data;
//  int        d_data;

var step_count uint64

type device struct {
	iarg int
	time uint64
}

var device_a device
var device_b device
var device_c device
var device_d device

// Callbacks, save step count in routine time and set argument to iarg
func (d *device) a_callback(iarg int) {
	d.iarg = iarg
	d.time = step_count
}

// Callbacks, save step count in routine time and set argument to iarg
func (d *device) b_callback(iarg int) {
	d.iarg = iarg
	d.time = step_count
}

// Callbacks, save step count in routine time and set argument to iarg
func (d *device) c_callback(iarg int) {
	d.iarg = iarg
	d.time = step_count
	Add_event(device_a, device_a.a_callback, iarg, iarg)
}

// Callbacks, save step count in routine time and set argument to iarg
func (d *device) d_callback(iarg int) {
	d.iarg = iarg
	d.time = step_count
}

func (d device) Start_IO() uint8 {
	return 0
}

func (d device) Start_cmd(cmd uint8) uint8 {
	return 0
}

func (d device) Halt_IO() uint8 {
	return 0
}

func (d device) Init_Dev() uint8 {
	return 0
}
func init_test() {
	step_count = 0
	device_a.time = 0
	device_b.time = 0
	device_c.time = 0
	device_d.time = 0
	device_a.iarg = 0
	device_b.iarg = 0
	device_c.iarg = 0
	device_d.iarg = 0
}

func TestAdd_event_1(t *testing.T) {
	init_test()
	Add_event(device_a, device_a.a_callback, 10, 1)
	for range 20 {
		step_count++
		Advance(1)
	}
	if device_a.time != 10 {
		t.Errorf("Event did not fire at correct time %d got %d", 10, device_a.time)
	}
	if device_a.iarg != 1 {
		t.Errorf("Event did not set data correct %d got %d", 1, device_a.iarg)
	}
}

// Add two events.
func TestAdd_event_2(t *testing.T) {
	init_test()
	Add_event(device_a, device_a.a_callback, 10, 1)
	Add_event(device_b, device_b.b_callback, 5, 2)
	for range 20 {
		step_count++
		Advance(1)
	}
	if device_a.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, device_a.time)
	}
	if device_a.iarg != 1 {
		t.Errorf("Event A did not set data correct %d got %d", 1, device_a.iarg)
	}
	if device_b.time != 5 {
		t.Errorf("Event B did not fire at correct time %d got %d", 5, device_b.time)
	}
	if device_b.iarg != 2 {
		t.Errorf("Event B did not set data correct %d got %d", 2, device_b.iarg)
	}
}

// Add two events.
func TestAdd_event_2a(t *testing.T) {
	init_test()
	Add_event(device_a, device_a.a_callback, 10, 1)
	Add_event(device_b, device_b.a_callback, 5, 2)
	for range 20 {
		step_count++
		Advance(1)
	}
	if device_a.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, device_a.time)
	}
	if device_a.iarg != 1 {
		t.Errorf("Event A did not set data correct %d got %d", 1, device_a.iarg)
	}
	if device_b.time != 5 {
		t.Errorf("Event B did not fire at correct time %d got %d", 5, device_b.time)
	}
	if device_b.iarg != 2 {
		t.Errorf("Event B did not set data correct %d got %d", 2, device_b.iarg)
	}
}

// Add event With same time
func TestAdd_event_3(t *testing.T) {
	init_test()
	Add_event(device_a, device_a.a_callback, 10, 1)
	Add_event(device_b, device_b.b_callback, 10, 2)
	for range 20 {
		step_count++
		Advance(1)
	}
	if device_a.time != 10 {
		t.Errorf("Event A did not fire at correct time %d got %d", 10, device_a.time)
	}
	if device_a.iarg != 1 {
		t.Errorf("Event A did not set data correct %d got %d", 1, device_a.iarg)
	}
	if device_b.time != 10 {
		t.Errorf("Event B did not fire at correct time %d got %d", 10, device_b.time)
	}
	if device_b.iarg != 2 {
		t.Errorf("Event B did not set data correct %d got %d", 2, device_b.iarg)
	}
}

// Add event during event.
func TestAdd_event_4(t *testing.T) {
	init_test()
	Add_event(device_a, device_a.a_callback, 20, 5)
	Add_event(device_c, device_c.c_callback, 10, 2)
	for range 30 {
		step_count++
		Advance(1)
	}
	if device_a.time != 20 {
		t.Errorf("Event A did not fire at correct time %d got %d", 20, device_a.time)
	}
	if device_a.iarg != 5 {
		t.Errorf("Event A did not set data correct %d got %d", 5, device_a.iarg)
	}
	if device_c.time != 10 {
		t.Errorf("Event C did not fire at correct time %d got %d", 10, device_c.time)
	}
	if device_c.iarg != 2 {
		t.Errorf("Event C did not set data correct %d got %d", 2, device_c.iarg)
	}

	// 	 ASSERT_EQUAL(15, a_time);
	// 	 ASSERT_EQUAL(5, a_data);
	// 	 ASSERT_EQUAL(20, b_time);
	// 	 ASSERT_EQUAL(2, b_data);
	// 	 ASSERT_EQUAL(10, c_time);
	// 	 ASSERT_EQUAL(0, c_data);
	//  }
}

// Schedule 3 events, last one before first, make sure all are correct
func TestAdd_event_5(t *testing.T) {
	init_test()
	Add_event(device_a, device_a.a_callback, 20, 1)
	Add_event(device_b, device_b.b_callback, 20, 2)
	Add_event(device_d, device_d.d_callback, 25, 3)
	for range 30 {
		step_count++
		Advance(1)
	}
	if device_a.time != 20 {
		t.Errorf("Event A did not fire at correct time %d got %d", 20, device_a.time)
	}
	if device_a.iarg != 1 {
		t.Errorf("Event A did not set data correct %d got %d", 1, device_a.iarg)
	}
	if device_b.time != 20 {
		t.Errorf("Event B did not fire at correct time %d got %d", 20, device_b.time)
	}
	if device_b.iarg != 2 {
		t.Errorf("Event B did not set data correct %d got %d", 2, device_b.iarg)
	}
	if device_d.time != 25 {
		t.Errorf("Event D did not fire at correct time %d got %d", 25, device_d.time)
	}
	if device_d.iarg != 3 {
		t.Errorf("Event D did not set data correct %d got %d", 3, device_d.iarg)
	}

	// 	 ASSERT_EQUAL(15, a_time);
	// 	 ASSERT_EQUAL(5, a_data);
	// 	 ASSERT_EQUAL(20, b_time);
	// 	 ASSERT_EQUAL(2, b_data);
	// 	 ASSERT_EQUAL(10, c_time);
	// 	 ASSERT_EQUAL(0, c_data);
	//  }
}

//  /* Schedule 3 events, last one before first, make sure all are correct */
//  CTEST(event, test5) {
// 	 struct _device  dev;

// 	 init_test();
// 	 add_event(&dev, &a_callback, 20, (void *)&a_data, 1);
// 	 add_event(&dev, &b_callback, 20, (void *)&b_data, 2);
// 	 add_event(&dev, &d_callback, 25, (void *)&d_data, 3);
// 	 while (step_count < 30) {
// 		 step_count++;
// 		 advance();
// 	 };
// 	 ASSERT_EQUAL(20, a_time);
// 	 ASSERT_EQUAL(1, a_data);
// 	 ASSERT_EQUAL(20, b_time);
// 	 ASSERT_EQUAL(2, b_data);
// 	 ASSERT_EQUAL(25, d_time);
// 	 ASSERT_EQUAL(3, d_data);
//  }
//  CTEST(event, test6) {
// 	 struct _device  dev;

// 	 init_test();
// 	 add_event(&dev, &a_callback, 10, (void *)&a_data, 5);
// 	 add_event(&dev, &b_callback, 20, (void *)&b_data, 2);
// 	 while (step_count < 30) {
// 		 step_count++;
// 		 advance();
// 		 if (a_data == 5) {
// 			cancel_event(&dev, &b_callback);
// 		 }
// 	 };
// 	 ASSERT_EQUAL(10, a_time);
// 	 ASSERT_EQUAL(5, a_data);
// 	 ASSERT_EQUAL(0, b_time);
// 	 ASSERT_EQUAL(0, b_data);
//  }

//  CTEST(event, test7) {
// 	 struct _device  dev;

// 	 init_test();
// 	 add_event(&dev, &a_callback, 10, (void *)&a_data, 5);
// 	 add_event(&dev, &b_callback, 20, (void *)&b_data, 2);
// 	 add_event(&dev, &d_callback, 30, (void *)&d_data, 3);
// 	 while (step_count < 30) {
// 		 step_count++;
// 		 advance();
// 		 if (a_data == 5) {
// 			cancel_event(&dev, &b_callback);
// 		 }
// 	 };
// 	 ASSERT_EQUAL(10, a_time);
// 	 ASSERT_EQUAL(5, a_data);
// 	 ASSERT_EQUAL(0, b_time);
// 	 ASSERT_EQUAL(0, b_data);
// 	 ASSERT_EQUAL(30, d_time);
// 	 ASSERT_EQUAL(3, d_data);
//  }
