/*
 * S370 - Tape emulation test cases.
 *
 * Copyright 2022, Richard Cornwell
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

package tape

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
)

var (
	fileTap  string
	fileE11  string
	fileP7B  string
	fileTemp string
	ctx      *TapeContext
)

// Write out tape block
func writeBlock(file *os.File, buffer []byte, format int) {
	recl := len(buffer)
	length := recl

	switch format {
	case TapeFmtTap:
		if (length & 1) != 0 {
			buffer = append(buffer, 0)
		}
		fallthrough
	case TapeFmtE11:
		lrecl := []byte{
			byte(recl & 0xff),
			byte((recl >> 8) & 0xff),
			byte((recl >> 16) & 0xff),
			byte((recl >> 24) & 0xff),
		}
		_, _ = file.Write(lrecl)
		_, _ = file.Write(buffer)
		_, _ = file.Write(lrecl)
	case TapeFmtP7B:
		// Put IRG on head of record.
		buffer[0] |= p7bIRG
		_, _ = file.Write(buffer)
	}
}

// Write a tape mark.
func writeMark(file *os.File, format int) {
	switch format {
	case TapeFmtTap, TapeFmtE11:
		lrecl := []byte{0, 0, 0, 0}
		_, _ = file.Write(lrecl)
	case TapeFmtP7B:
		// Put IRG on head of record.
		rec := []byte{BCDTM | p7bIRG}
		_, _ = file.Write(rec)
	}
}

// Create tape file.
func createTapeFile(file *os.File, recs int, format int) {
	// Write bunch of odd recordds.
	for i := range recs {
		rec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", i)
		writeBlock(file, []byte(rec), format)
	}
	writeMark(file, format)
	// Write bunch of even records.
	for i := range recs {
		rec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789XY", i+recs)
		writeBlock(file, []byte(rec), format)
	}
	writeMark(file, format)
	file.Close()
}

// Creat test tape files
func setupTape() error {
	f, err := os.CreateTemp("", "tapeP7B")
	if err != nil {
		return err
	}
	fileP7B = f.Name()
	createTapeFile(f, 100, TapeFmtP7B)
	f.Close()

	f, err = os.CreateTemp("", "tapeTAP")
	if err != nil {
		return err
	}
	fileTap = f.Name()
	createTapeFile(f, 100, TapeFmtTap)
	f.Close()

	f, err = os.CreateTemp("", "tapeE11")
	if err != nil {
		return err
	}
	fileE11 = f.Name()
	createTapeFile(f, 100, TapeFmtE11)
	f.Close()
	return nil
}

func freeCTX() {
	if ctx == nil {
		return
	}
	if ctx.file != nil {
		ctx.file.Close()
	}
}

func cleanup() {
	freeCTX()
	os.Remove(fileP7B)
	os.Remove(fileTap)
	os.Remove(fileE11)
}

// Check that we can attach to a tape.
func TestAttach(t *testing.T) {
	ctx = NewTapeContext()
	defer cleanup()
	err := setupTape()
	if err != nil {
		t.Error(err)
		return
	}

	err = ctx.Attach(fileTap, TapeFmtTap, false, false)
	if err != nil {
		t.Error(err)
	}
	if !ctx.TapeAtLoadPt() {
		t.Error("Tap format not at load point")
	}
	_ = ctx.Detach()

	err = ctx.Attach(fileTap, TapeFmtE11, false, false)
	if err != nil {
		t.Error(err)
	}
	if !ctx.TapeAtLoadPt() {
		t.Error("E11 format not at load point")
	}
	_ = ctx.Detach()

	err = ctx.Attach(fileTap, TapeFmtP7B, false, false)
	if err != nil {
		t.Error(err)
	}
	if !ctx.TapeAtLoadPt() {
		t.Error("P7B format not at load point")
	}
	_ = ctx.Detach()
	cleanup()
	err = ctx.Attach(fileTap, TapeFmtE11, false, false)
	if !errors.Is(err, os.ErrNotExist) {
		t.Error(err)
	}
}

// Read test of tape.
func testRead(fileName string, fmtStr string, format int, t *testing.T) {
	ctx = NewTapeContext()
	err := ctx.Attach(fileName, format, false, false)
	if err != nil {
		t.Error(err)
	}
	defer ctx.Detach()
	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}
	if ctx.TapeRing() {
		t.Error(fmtStr + " not write protect")
	}

	rec := 0
	mark := false
	for !mark {
		err = ctx.ReadForwStart()
		if err != nil {
			if !errors.Is(err, TapeMARK) {
				t.Error(err)
			} else {
				_ = ctx.FinishRecord()
				mark = true
			}
			break
		}

		buffer := []byte{}
		for {
			var data byte
			data, err = ctx.ReadFrame()
			if err != nil {
				if !errors.Is(err, TapeEOR) {
					t.Error(err)
				}
				break
			}
			buffer = append(buffer, data)
		}

		err = ctx.FinishRecord()
		if errors.Is(err, TapeMARK) {
			mark = true
			break
		}

		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", rec)
		if string(buffer) != testRec {
			t.Error(fmtStr + "Read failed go: " + string(buffer))
			t.Error(fmtStr + "Expected:       " + testRec)
		}

		if ctx.TapeAtLoadPt() {
			t.Error(fmtStr + " format at load point")
			break
		}
		rec++
	}

	if !mark {
		t.Error(fmtStr + " Mark not found")
	}

	if rec != 100 {
		t.Errorf("%s Got %d records, expected: 100", fmtStr, rec)
	}

	mark = false
	for !mark {
		err = ctx.ReadForwStart()
		if err != nil {
			if !errors.Is(err, TapeMARK) {
				t.Error(err)
			} else {
				_ = ctx.FinishRecord()
				mark = true
			}
			break
		}

		buffer := []byte{}
		for {
			var data byte
			data, err = ctx.ReadFrame()
			if err != nil {
				if !errors.Is(err, TapeEOR) {
					t.Error(err)
				}
				break
			}
			buffer = append(buffer, data)
		}

		err = ctx.FinishRecord()
		if errors.Is(err, TapeMARK) {
			mark = true
			break
		}

		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789XY", rec)
		if string(buffer) != testRec {
			t.Errorf("%s Read failed: %s", fmtStr, string(buffer))
		}

		if ctx.TapeAtLoadPt() {
			t.Error(fmtStr + " format at load point")
			break
		}
		rec++
	}

	if !mark {
		t.Error(fmtStr + " Mark not found")
	}

	if rec != 200 {
		t.Errorf("%s Got %d records, expected: 100", fmtStr, rec)
	}
}

// Test read of E11 tape.
func TestReadE11(t *testing.T) {
	defer cleanup()
	err := setupTape()
	if err != nil {
		t.Error(err)
		return
	}

	testRead(fileE11, "E11", TapeFmtE11, t)
}

// Test read of TAP tape.
func TestReadTap(t *testing.T) {
	defer cleanup()
	err := setupTape()
	if err != nil {
		t.Error(err)
		return
	}

	testRead(fileTap, "Tap", TapeFmtTap, t)
}

// Test read of P7B tape.
func TestReadP7B(t *testing.T) {
	defer cleanup()
	err := setupTape()
	if err != nil {
		t.Error(err)
		return
	}

	testRead(fileP7B, "P7B", TapeFmtP7B, t)
}

// Read test of tape.
func testWrite(fmtStr string, format int, t *testing.T) {
	ctx = NewTapeContext()
	f, err := os.CreateTemp("", "tapeFile")
	if err != nil {
		return
	}
	fileTemp = f.Name()
	f.Close()
	defer os.Remove(fileTemp)

	mark := false
	err = ctx.Attach(fileTemp, format, true, false)
	if err != nil {
		t.Error(err)
	}
	defer ctx.Detach()
	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}
	if !ctx.TapeRing() {
		t.Error(fmtStr + " is write protect")
	}

	rec := 0
	sz := 0
	// Write a long enough tape to run over a couple buffers
	for sz < (80 * 1024) {
		err = ctx.WriteStart()
		if err != nil {
			t.Error(err)
			break
		}

		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", rec)
		for _, data := range testRec {
			err = ctx.WriteFrame(byte(data))
			if err != nil {
				t.Error(err)
				break
			}
			sz++
		}

		err = ctx.FinishRecord()
		if err != nil {
			t.Error(err)
			break
		}
		rec++
	}

	// Write tape mark.
	err = ctx.WriteMark()
	if err != nil {
		t.Error(err)
	}

	// Rewind tape to beginning
	err = ctx.StartRewind()
	if err != nil {
		t.Error(err)
	}

	for !ctx.RewindFrames(10000) {
	}

	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}
	readSz := 0
	readRec := 0

	mark = false
	for !mark {
		err = ctx.ReadForwStart()
		if err != nil {
			if !errors.Is(err, TapeMARK) {
				t.Error(err)
			} else {
				mark = true
			}
			break
		}
		buffer := []byte{}
		for {
			var data byte
			data, err = ctx.ReadFrame()
			if err != nil {
				if !errors.Is(err, TapeEOR) {
					t.Error(err)
				}
				break
			}
			buffer = append(buffer, data)
			readSz++
		}

		err = ctx.FinishRecord()
		if errors.Is(err, TapeMARK) {
			mark = true
			break
		}

		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", readRec)
		if string(buffer) != testRec {
			t.Errorf(fmtStr+" Read failed: %s", string(buffer))
		}

		if ctx.TapeAtLoadPt() {
			t.Error(fmtStr + " format at load point")
			break
		}
		readRec++
	}

	if !mark {
		t.Error(fmtStr + " Mark not found")
	}

	if rec != readRec {
		t.Errorf(fmtStr+" Got %d records, expected: %d", readRec, rec)
	}
}

// Test write of E11 tape.
func TestWriteE11(t *testing.T) {
	testWrite("E11", TapeFmtE11, t)
}

// Test write of Tap tape.
func TestWriteTap(t *testing.T) {
	testWrite("Tap", TapeFmtTap, t)
}

// Test write of P7B tape.
func TestWriteP7B(t *testing.T) {
	testWrite("P7B", TapeFmtP7B, t)
}

// Write a series of records of increasing size.
func testWriteLong(fmtStr string, format int, t *testing.T) {
	ctx = NewTapeContext()
	f, err := os.CreateTemp("", "tapeFile")
	if err != nil {
		return
	}
	fileTemp = f.Name()
	f.Close()
	defer os.Remove(fileTemp)

	mark := false
	err = ctx.Attach(fileTemp, format, true, false)
	if err != nil {
		t.Error(err)
	}
	defer ctx.Detach()
	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}
	if !ctx.TapeRing() {
		t.Error(fmtStr + " is write protect")
	}

	rec := 0
	sz := 2000
	// Write a long enough tape to run over a couple buffers
	for sz < (80 * 1024) {
		err = ctx.WriteStart()
		if err != nil {
			t.Error(err)
			break
		}

		for i := range sz {
			data := (i & 0xff)
			err = ctx.WriteFrame(byte(data))
			if err != nil {
				t.Error(err)
				break
			}
		}

		err = ctx.FinishRecord()
		if err != nil {
			t.Error(err)
			break
		}
		sz += 2000
		rec++
	}

	// Write tape mark.
	err = ctx.WriteMark()
	if err != nil {
		t.Error(err)
	}

	// Rewind tape to beginning
	err = ctx.StartRewind()
	if err != nil {
		t.Error(err)
	}

	done := false
	for !done {
		done = ctx.RewindFrames(10000)
	}

	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}

	readSz := 2000
	readRec := 0

	// Make sure we can read them back.
	mark = false
	for !mark {
		err = ctx.ReadForwStart()
		if err != nil {
			if !errors.Is(err, TapeMARK) {
				t.Error(err)
			} else {
				mark = true
			}
			break
		}

		i := 0
		fail := false
		for {
			var data byte
			data, err = ctx.ReadFrame()
			if err != nil {
				if !errors.Is(err, TapeEOR) {
					t.Error(err)
				}
				break
			}
			if format == TapeFmtP7B {
				if data != byte(i&0x7f) {
					fail = true
				}
			} else {
				if data != byte(i&0xff) {
					fail = true
				}
			}
			i++
		}

		err = ctx.FinishRecord()
		if errors.Is(err, TapeMARK) {
			mark = true
			break
		}

		if fail {
			t.Error(fmtStr + " record did not match")
		}

		if i != readSz {
			t.Errorf(fmtStr+" Got %d wrong size, expected: %d", readSz, i)
		}

		if ctx.TapeAtLoadPt() {
			t.Error(fmtStr + " format at load point")
			break
		}
		readRec++
		readSz += 2000
	}

	if !mark {
		t.Error(fmtStr + " Mark not found")
	}

	if rec != readRec {
		t.Errorf(fmtStr+" Got %d records, expected: %d", readRec, rec)
	}
}

// Write a series of record in increasing size.
func TestWriteLongE11(t *testing.T) {
	testWriteLong("E11", TapeFmtE11, t)
}

// Write a series of record in increasing size.
func TestWriteLongTap(t *testing.T) {
	testWriteLong("Tap", TapeFmtTap, t)
}

// Write a series of record in increasing size.
func TestWriteLongP7B(t *testing.T) {
	testWriteLong("P7B", TapeFmtP7B, t)
}

// Write a tape mark ever 10 records, verify read correct forward and backwards.
func testMark(fmtStr string, format int, t *testing.T) {
	ctx = NewTapeContext()
	f, err := os.CreateTemp("", "tapeFile")
	if err != nil {
		return
	}
	fileTemp = f.Name()
	f.Close()
	defer os.Remove(fileTemp)

	err = ctx.Attach(fileTemp, format, true, false)
	if err != nil {
		t.Error(err)
	}
	defer ctx.Detach()
	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}
	if !ctx.TapeRing() {
		t.Error(fmtStr + " is write protect")
	}

	rec := 0
	// Every 10 records write a tape mark, then put 2 on end
	for rec <= 200 {
		err = ctx.WriteStart()
		if err != nil {
			t.Error(err)
			break
		}

		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", rec)
		if rec > 100 {
			testRec += "Y"
		}
		for _, data := range testRec {
			err = ctx.WriteFrame(byte(data))
			if err != nil {
				t.Error(err)
				break
			}
		}

		err = ctx.FinishRecord()
		if err != nil {
			t.Error(err)
			break
		}

		if (rec % 10) == 0 {
			err = ctx.WriteMark()
			if err != nil {
				t.Error(err)
				break
			}
		}
		rec++
	}

	// Write tape mark.
	err = ctx.WriteMark()
	if err != nil {
		t.Error(err)
	}

	// Rewind tape to beginning
	err = ctx.StartRewind()
	if err != nil {
		t.Error(err)
	}

	done := false
	for !done {
		done = ctx.RewindFrames(10000)
	}

	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}

	readRec := 0
	mark := 0

	// Make sure we can read them back.
	for {
		err = ctx.ReadForwStart()

		// Check if tape Mark
		if errors.Is(err, TapeMARK) {
			mark++
			err = ctx.FinishRecord()
			if err != nil {
				t.Error(err)
				continue
			}
			err = ctx.ReadForwStart()
			if errors.Is(err, TapeMARK) {
				break
			}
			if (readRec % 10) != 1 {
				t.Errorf(fmtStr+" Mark not correct: %d", readRec)
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Error(err)
		}
		buffer := []byte{}
		for {
			var data byte
			data, err = ctx.ReadFrame()
			if err != nil {
				if !errors.Is(err, TapeEOR) {
					t.Error(err)
				}
				break
			}
			buffer = append(buffer, data)
		}

		err = ctx.FinishRecord()
		if err != nil {
			t.Error(err)
		}

		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", readRec)
		if readRec > 100 {
			testRec += "Y"
		}
		if string(buffer) != testRec {
			t.Error(fmtStr + " Expected: " + testRec)
			t.Error(fmtStr + " Read failed: " + string(buffer))
		}

		if ctx.TapeAtLoadPt() {
			t.Error(fmtStr + " format at load point")
			break
		}
		readRec++
	}

	if mark != 21 {
		t.Errorf(fmtStr+" Mark count wrong %d", mark)
	}

	if rec != readRec {
		t.Errorf(fmtStr+" Got %d records, expected: %d", readRec, rec)
	}

	// Skip double MARK
	err = ctx.ReadBackStart()

	// Check if tape Mark
	if !errors.Is(err, TapeMARK) {
		t.Error(err)
	}
	err = ctx.FinishRecord()
	if err != nil {
		t.Error(err)
	}

	// Make sure we can read backward
	for !ctx.TapeAtLoadPt() {
		err = ctx.ReadBackStart()

		// Check if tape Mark
		if errors.Is(err, TapeMARK) {
			mark--
			err = ctx.FinishRecord()
			if err != nil {
				t.Error(err)
				continue
			}
			_ = ctx.ReadBackStart()
			if (readRec % 10) != 1 {
				t.Errorf(fmtStr+" Mark not correct: %d", readRec)
			}
		} else if errors.Is(err, TapeBOT) {
			break
		} else if err != nil {
			t.Error(err)
		}

		readRec--
		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", readRec)
		if readRec > 100 {
			testRec += "Y"
		}
		l := len(testRec)
		buffer := make([]byte, l)

		for l > 0 {
			var data byte
			data, err = ctx.ReadFrame()
			if err != nil {
				if !errors.Is(err, TapeEOR) {
					t.Error(err)
				}
				break
			}
			l--
			buffer[l] = data
		}

		err = ctx.FinishRecord()
		if err != nil {
			t.Error(err)
		}

		if string(buffer) != testRec {
			t.Error(fmtStr + " Expected:    " + testRec)
			t.Errorf(fmtStr + " Read failed: " + string(buffer))
		}
	}

	if mark != 0 {
		t.Errorf(fmtStr+" Mark count wrong %d", mark)
	}

	if readRec != 0 {
		t.Errorf(fmtStr+" Got %d records, expected: %d", rec-readRec, rec)
	}
}

// Write a tape mark ever 10 records, verify read correct forward and backwards.
func TestWMarkE11(t *testing.T) {
	testMark("E11", TapeFmtE11, t)
}

// Write a tape mark ever 10 records, verify read correct forward and backwards.
func TestWMarkTAP(t *testing.T) {
	testMark("TAP", TapeFmtTap, t)
}

// Write a tape mark ever 10 records, verify read correct forward and backwards.
func TestWMarkP7B(t *testing.T) {
	testMark("P7B", TapeFmtP7B, t)
}

// Write a tape mark ever 10 records, verify read correct forward and backwards.
func testShortRead(fmtStr string, format int, t *testing.T) {
	ctx = NewTapeContext()
	f, err := os.CreateTemp("", "tapeFile")
	if err != nil {
		return
	}
	fileTemp = f.Name()
	f.Close()
	defer os.Remove(fileTemp)

	err = ctx.Attach(fileTemp, format, true, false)
	if err != nil {
		t.Error(err)
	}
	defer ctx.Detach()
	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}
	if !ctx.TapeRing() {
		t.Error(fmtStr + " is write protect")
	}

	rec := 0
	// Write out 200 records 100 odd/100 even.
	for rec <= 200 {
		err = ctx.WriteStart()
		if err != nil {
			t.Error(err)
			break
		}

		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", rec)
		if rec > 100 {
			testRec += "Y"
		}
		for _, data := range testRec {
			err = ctx.WriteFrame(byte(data))
			if err != nil {
				t.Error(err)
				break
			}
		}

		err = ctx.FinishRecord()
		if err != nil {
			t.Error(err)
			break
		}
		rec++
	}

	// Write tape mark.
	err = ctx.WriteMark()
	if err != nil {
		t.Error(err)
	}

	// Rewind tape to beginning
	err = ctx.StartRewind()
	if err != nil {
		t.Error(err)
	}

	done := false
	for !done {
		done = ctx.RewindFrames(10000)
	}

	if !ctx.TapeAtLoadPt() {
		t.Error(fmtStr + " format not at load point")
	}

	readRec := 0
	mark := false

	length := 79
	// Make sure we can read them back.
	for !mark {
		err = ctx.ReadForwStart()

		if err != nil {
			if !errors.Is(err, TapeMARK) {
				t.Error(err)
			} else {
				mark = true
			}
			break
		}

		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Error(err)
		}
		buffer := []byte{}
		for range length {
			var data byte
			data, err = ctx.ReadFrame()
			if err != nil {
				if !errors.Is(err, TapeEOR) {
					t.Error(err)
				}
				break
			}
			buffer = append(buffer, data)
		}

		err = ctx.FinishRecord()
		if err != nil {
			t.Error(err)
		}

		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", readRec)
		if readRec > 100 {
			testRec += "Y"
		}
		if string(buffer) != testRec[:length] {
			t.Error(fmtStr + " Expected:    " + testRec[:length])
			t.Error(fmtStr + " Read failed: " + string(buffer))
		}

		length--
		if length < 0 {
			length = 79
			if readRec > 100 {
				length++
			}
		}
		if ctx.TapeAtLoadPt() {
			t.Error(fmtStr + " format at load point")
			break
		}
		readRec++
	}

	if rec != readRec {
		t.Errorf(fmtStr+" Got %d records, expected: %d", readRec, rec)
	}

	// Skip double MARK
	err = ctx.ReadBackStart()

	// Check if tape Mark
	if !errors.Is(err, TapeMARK) {
		t.Error(err)
	}
	err = ctx.FinishRecord()
	if err != nil {
		t.Error(err)
	}

	length = 80
	// Make sure we can read backward
	for !ctx.TapeAtLoadPt() {
		err = ctx.ReadBackStart()

		// Check if BOT
		if errors.Is(err, TapeBOT) {
			break
		}
		if err != nil {
			t.Error(err)
			break
		}

		readRec--
		testRec := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789X", readRec)
		if readRec > 100 {
			testRec += "Y"
		}
		buffer := make([]byte, length)

		for l := length - 1; l >= 0; l-- {
			var data byte
			data, err = ctx.ReadFrame()
			if err != nil {
				if !errors.Is(err, TapeEOR) {
					t.Error(err)
				}
				break
			}
			buffer[l] = data
		}

		err = ctx.FinishRecord()
		if err != nil {
			t.Error(err)
		}

		sz := len(testRec) - length
		if string(buffer) != testRec[sz:] {
			t.Error(fmtStr + " Expected:    " + testRec[sz:])
			t.Error(fmtStr + " Read failed: " + string(buffer))
		}

		length--
		if length < 0 {
			length = 79
			if readRec > 100 {
				length++
			}
		}
	}

	if readRec != 0 {
		t.Errorf(fmtStr+" Got %d records, expected: %d", rec-readRec, rec)
	}
}

// Read part of record, make sure unread is skipped.
func TestShortReadE11(t *testing.T) {
	testShortRead("E11", TapeFmtE11, t)
}

// Read part of record, make sure unread is skipped.
func TestShortReadTAP(t *testing.T) {
	testShortRead("TAP", TapeFmtTap, t)
}

// Read part of record, make sure unread is skipped.
func TestShortReadP7B(t *testing.T) {
	testShortRead("P7B", TapeFmtP7B, t)
}

// Test Odd record has pad character.
func TestOddTap(t *testing.T) {
	ctx = NewTapeContext()
	f, err := os.CreateTemp("", "tapeFile")
	if err != nil {
		return
	}
	fileTemp = f.Name()
	f.Close()
	defer os.Remove(fileTemp)

	err = ctx.Attach(fileTemp, TapeFmtTap, true, false)
	if err != nil {
		t.Error(err)
	}

	if !ctx.TapeAtLoadPt() {
		t.Error("Tap format not at load point")
	}
	if !ctx.TapeRing() {
		t.Error("Tap is write protect")
	}

	err = ctx.WriteStart()
	if err != nil {
		t.Error(err)
	}

	testRec := "ABCDE"
	for _, data := range testRec {
		err = ctx.WriteFrame(byte(data))
		if err != nil {
			t.Error(err)
			break
		}
	}

	err = ctx.FinishRecord()
	if err != nil {
		t.Error(err)
	}
	ctx.Detach()

	buffer, _ := os.ReadFile(fileTemp)

	match := []byte{5, 0, 0, 0, 'A', 'B', 'C', 'D', 'E', 0, 5, 0, 0, 0}
	if len(buffer) != len(match) {
		t.Errorf("Tap odd record not correct length: %d should be: %d", len(buffer), len(match))
	}
	for i, m := range match {
		if i < len(buffer) && buffer[i] != m {
			t.Errorf("Tap record mismatch: %d  %02x != %02x", i, buffer[i], match[i])
		}
	}
}
