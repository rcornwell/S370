/*
 * S370 - Generic tape interface.
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

package tape

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// Supported tape formats.
	TapeFmtTap = 1 + iota
	TapeFmtE11
	TapeFmtP7B
	TapeFmtAWS

	// P7B constants.
	p7bIRG byte = 0x80
	BCDTM  byte = 0x17

	irgLen = 1200

	// Supported densities.
	Density800 = 1 + iota
	Density1600
	Density6250

	// Currently running function.
	funcNone = 0
	funcRead = 1 + iota
	funcWrite
	funcRewind
	funcReadBack
	funcMark
)

var (
	TapeEOT    = errors.New("EOT")    // End of tape error.
	TapeMARK   = errors.New("MARK")   // Tape mark found.
	TapeBOT    = errors.New("BOT")    // Beginning of tape.
	TapeEOR    = errors.New("EOR")    // End of record.
	TapeFORMAT = errors.New("FORMAT") // Tape format error.
	TapeTYPE   = errors.New("TYPE")   // Tape type not supported.
)

// Structure to hold tape information.
type TapeContext struct {
	file     *os.File        // file handle
	mode     int             // Current input/output mode
	format   int             // Tape format
	ring     bool            // Has write ring
	mark     bool            // Last record was tape mark.
	bot      bool            // At beginning of tape
	eot      bool            // At end of tape
	seven    bool            // Seven track drive
	frame    int             // Current frame
	bufPos   int             // Position in buffer
	bufLen   int             // Length of buffer
	position int64           // Position of head of buffer in tape
	lrecl    uint32          // Length of current record.
	recPos   uint32          // Position in logical record.
	startRec int64           // Start of record in tape.
	dirty    bool            // Buffer is dirty.
	buffer   [32 * 1024]byte // Tape buffer.
}

// Check if tape is at load point.
func (tape *TapeContext) TapeAtLoadPt() bool {
	return tape.bot
}

// Determine if tape is attached and ready.
func (tape *TapeContext) TapeReady() bool {
	return tape.file != nil
}

// Determine if tape can be written.
func (tape *TapeContext) TapeRing() bool {
	return tape.ring
}

// Determine if tape is 7 track or 9 track.
func (tape *TapeContext) Tape9Track() bool {
	return !tape.seven
}

// Attach file to tape context.
func (tape *TapeContext) Attach(fileName string, format int, ring bool, seven bool) error {
	tape.format = format
	tape.seven = seven
	tape.ring = ring
	var err error
	if ring {
		tape.file, err = os.Create(fileName)
	} else {
		tape.file, err = os.Open(fileName)
	}
	tape.position = 0
	tape.bot = true
	tape.eot = false
	tape.mark = false
	tape.bufPos = 0
	tape.bufLen = 0
	tape.lrecl = 0
	tape.startRec = 0
	tape.dirty = false
	return err
}

// Detach a tape file from a tape context.
func (tape *TapeContext) Detach() error {
	var err error
	// If buffer is dirty flush it to the file
	if tape.dirty {
		n := 0
		_, _ = tape.file.Seek(tape.position, io.SeekStart)
		n, err = tape.file.Write(tape.buffer[:tape.bufLen])
		if n != tape.bufLen {
			err = errors.New("Write error on: " + tape.file.Name())
		}
		tape.dirty = false
	}
	tape.file.Close()
	tape.file = nil
	return err
}

// Start a tape write operation.
func (tape *TapeContext) WriteStart() error {
	// Error if not attached.
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	// Check if tape writable.
	if !tape.ring {
		return errors.New("tape write protected")
	}

	// Clear BOT and EOT indicators.
	tape.bot = false
	tape.eot = false
	tape.recPos = 0
	tape.mode = funcWrite

	// Save starting record to update later.
	tape.startRec = tape.position + int64(tape.bufPos)

	var err error
	// Set up start of record based on tape type.
	switch tape.format {
	case TapeFmtTap, TapeFmtE11:
		//  Write dummy record length
		for range 4 {
			err = tape.writeNextFrame(0)
			if err != nil {
				break
			}
		}

	case TapeFmtP7B:
	case TapeFmtAWS:
		hdr := []byte{0, 0,
			byte((tape.lrecl >> 8) & 0xff),
			byte(tape.lrecl & 0xff),
			0xA, 0,
		}
		if tape.mark {
			hdr[4] = 0x4
			tape.mark = false
		}

		//  Write dummy record length
		for _, d := range hdr {
			err = tape.writeNextFrame(d)
			if err != nil {
				break
			}
		}

	default:
		err = TapeTYPE
	}
	tape.lrecl = 0
	return err
}

// Write Mark to tape.
func (tape *TapeContext) WriteMark() error {
	// Error if not attached.
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	// Check if tape writable.
	if !tape.ring {
		return errors.New("tape write protected")
	}

	var err error
	// Clear BOT and EOT indicators.
	tape.bot = false
	tape.eot = false

	// Save starting record to update later.
	tape.startRec = tape.position + int64(tape.bufPos)

	tape.recPos = 0
	tape.mode = funcMark
	// Set up start of record based on tape type.
	switch tape.format {
	case TapeFmtTap, TapeFmtE11:
		//  Write dummy record length
		for range 4 {
			err = tape.writeNextFrame(0)
			if err != nil {
				break
			}
		}
		tape.lrecl = 0
	case TapeFmtP7B:
		err = tape.writeNextFrame(BCDTM | p7bIRG)
		tape.lrecl = 0
	case TapeFmtAWS:
		tape.mark = true
	default:
		err = TapeTYPE
	}
	tape.frame += irgLen
	return err
}

// Start a tape read forward operation.
func (tape *TapeContext) ReadForwStart() error {
	// Error if not attached.
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	// Clear BOT and EOT indicators.
	tape.bot = false
	tape.eot = false
	tape.mode = funcRead

	// Save starting record to update later.
	tape.startRec = tape.position + int64(tape.bufPos)

	// Set up start of record based on tape type.
	switch tape.format {
	case TapeFmtTap, TapeFmtE11:
		// Read in 4 byte record length
		hdr := [4]byte{}
		var err error
		for i := range 4 {
			hdr[i], err = tape.readNextFrame()
			if err != nil {
				return err
			}
		}
		tape.lrecl = uint32(hdr[0]) | (uint32(hdr[1]) << 8) |
			(uint32(hdr[2]) << 16) | (uint32(hdr[3]) << 24)
		if tape.lrecl == 0xffffffff {
			// We hit end of tape, backup so if write we erase it.
			tape.eot = true
			for range 4 {
				_, err = tape.readPrevFrame()
				if err != nil {
					return err
				}
			}
			return TapeEOT
		}

		// Check for tape mark
		if tape.lrecl == 0 {
			tape.frame += irgLen
			tape.mark = true
			return TapeMARK
		}

		// Debug log data.
		// 					j = tape->lrecl;
		// 					if (j > tape->len_buff)
		// 						j = tape->len_buff;
		// 					k = 0;
		// 					while (j > 0 && (tape->pos_buff + k + 16) < sizeof(tape->buffer)) {
		// 						log_tape_s("data ");
		// 						for(i = 0; i < j && i < 16; i++)
		// 							log_tape_c ("%02x ", tape->buffer[tape->pos_buff + i + k]);
		// 						log_tape_c(" ");
		// 						for(i = 0; i < j && i < 16; i++) {
		// 							uint8_t ch = ebcdic_to_ascii[tape->buffer[tape->pos_buff + i + k]];
		// 							log_tape_c ("%c", isprint(ch) ? ch : '.');
		// 						}
		// 						j -= 16;
		// 						k += 16;
		// 					}

		tape.recPos = 0 // Posision in logical record

	case TapeFmtP7B:
		// Peek at current character.
		// To see if it is a tape mark.
		data, err := tape.peekNextFrame()
		tape.lrecl = 2
		if err != nil {
			return err
		}

		// Check if tape mark.
		if data == (p7bIRG | BCDTM) {
			_, _ = tape.readNextFrame()
			tape.frame += irgLen
			tape.mark = true
			return TapeMARK
		}
		tape.lrecl = 0
		// 					tape->srec = tape->pos + tape->pos_buff;
		// 					r = tape_peek_byte(tape, &lrecl[0]);
		// 					tape->lrecl = 2;
		// 					if (r != 1)
		// 						return r;
		// 					/* If tape mark, move over it */
		// 					if (lrecl[0] == (IRG_MASK|BCD_TM)) {
		// 						r = tape_read_byte(tape, &lrecl[0]);
		// 						if (r < 0)
		// 							return r;
		// 						tape->pos_frame += IRG_LEN;
		// 						tape->format |= TAPE_MARK;
		// 						log_tape("Tape mark %d\n", r);
		// 						return (r == 0) ? 0 : 2;
		// 					}
		// 					tape->lrecl = 0;  /* Flag at beginning of record */
		// 					break;
	case TapeFmtAWS:
		//  Read record header
		hdr := [6]byte{}
		var err error
		for i := range 6 {
			hdr[i], err = tape.readNextFrame()
			if err != nil {
				return err
			}
		}

		tape.lrecl = (uint32(hdr[1]) << 8) | uint32(hdr[0])
		fmt.Printf("Header %02x %02x\n", hdr[4], hdr[5])
	default:
		return TapeTYPE
	}

	return nil
}

// Start a tape read backword operation.
func (tape *TapeContext) ReadBackStart() error {
	// Error if not attached.
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	// Clear BOT and EOT indicators.
	tape.bot = false
	tape.eot = false
	tape.mode = funcReadBack

	// Save starting record to update later.
	tape.startRec = tape.position + int64(tape.bufPos)

	// Set up start of record based on tape type.
	switch tape.format {
	case TapeFmtTap, TapeFmtE11:
		// Read in 4 byte record length
		recLen := [4]byte{}
		var err error
		for i := 3; i >= 0; i-- {
			recLen[i], err = tape.readPrevFrame()
			if err != nil {
				return err
			}
		}
		tape.lrecl = uint32(recLen[0]) | (uint32(recLen[1]) << 8) |
			(uint32(recLen[2]) << 16) | (uint32(recLen[3]) << 24)
		if tape.lrecl == 0xffffffff {
			// We hit end of tape, backup so if write we erase it.
			tape.eot = true
			return TapeEOT
		}

		// Check for tape mark
		if tape.lrecl == 0 {
			tape.frame += irgLen
			tape.mark = true
			return TapeMARK
		}

		// On tap format files odd records have 1 byte padding
		if tape.format == TapeFmtTap && (tape.lrecl&1) != 0 {
			_, err := tape.readPrevFrame()
			if err != nil {
				return err
			}
		}

		tape.recPos = tape.lrecl
		// Debug log data.
		// 					j = tape->lrecl;
		// 					if (j > tape->len_buff)
		// 						j = tape->len_buff;
		// 					k = 0;
		// 					while (j > 0 && (tape->pos_buff + k + 16) < sizeof(tape->buffer)) {
		// 						log_tape_s("data ");
		// 						for(i = 0; i < j && i < 16; i++)
		// 							log_tape_c ("%02x ", tape->buffer[tape->pos_buff + i + k]);
		// 						log_tape_c(" ");
		// 						for(i = 0; i < j && i < 16; i++) {
		// 							uint8_t ch = ebcdic_to_ascii[tape->buffer[tape->pos_buff + i + k]];
		// 							log_tape_c ("%c", isprint(ch) ? ch : '.');
		// 						}
		// 						j -= 16;
		// 						k += 16;
		// 					}

	case TapeFmtP7B:
		// Peek at current character.
		// To see if it is a tape mark.
		tape.startRec = tape.position + int64(tape.bufPos)
		data, err := tape.readPrevFrame()
		if err != nil {
			return err
		}

		tape.lrecl = 0
		// Check if tape mark.
		if data == (p7bIRG | BCDTM) {
			tape.startRec = tape.position + int64(tape.bufPos)
			tape.frame -= irgLen
			tape.lrecl = 2
			tape.mark = true
			return TapeMARK
		}
		// Reposition to start of next record.
		_, err2 := tape.readNextFrame()
		return err2

	case TapeFmtAWS:
		//  Read record header
		hdr := [6]byte{}
		var err error
		for i := 5; i >= 0; i-- {
			hdr[i], err = tape.readPrevFrame()
			if err != nil {
				return err
			}
		}

		tape.lrecl = (uint32(hdr[3]) << 8) | uint32(hdr[2])
		fmt.Printf("Header %02x %02x\n", hdr[4], hdr[5])
	default:
		return TapeTYPE
	}

	return nil
}

// Read one frame from tape.
func (tape *TapeContext) ReadFrame() (byte, error) {
	// Error if not attached.
	if tape.file == nil {
		return 0, errors.New("tape not attached")
	}

	if tape.mark {
		return 0, TapeMARK
	}

	var err error
	var data byte
	switch tape.format {
	case TapeFmtTap, TapeFmtE11:
		switch tape.mode {
		case funcRead:
			if tape.recPos == tape.lrecl {
				return 0, TapeEOR
			}
			data, err = tape.readNextFrame()
			tape.recPos++
			tape.frame++
		case funcReadBack:
			if tape.recPos == 0 {
				return 0, TapeEOR
			}
			data, err = tape.readPrevFrame()
			tape.recPos--
			tape.frame--
		}

	case TapeFmtP7B:
		switch tape.mode {
		case funcRead:
			if tape.lrecl == 2 {
				return 0, TapeEOR
			}
			data, err = tape.readNextFrame()
			if tape.lrecl == 1 && (data&p7bIRG) != 0 {
				_, _ = tape.readPrevFrame()
				tape.lrecl = 2
				return 0, TapeEOR
			} else {
				tape.lrecl = 1
			}
			tape.frame++
			data &= ^p7bIRG
		case funcReadBack:
			if tape.lrecl == 2 {
				return 0, TapeEOR
			}
			data, err = tape.readPrevFrame()
			if tape.lrecl == 1 && (data&p7bIRG) != 0 {
				tape.lrecl = 2
			} else {
				tape.lrecl = 1
			}
			data &= ^p7bIRG
			tape.frame--
		}
	case TapeFmtAWS:
		switch tape.mode {
		case funcRead:
			if tape.recPos == tape.lrecl {
				return 0, TapeEOR
			}
			data, err = tape.readNextFrame()
			tape.recPos++
			tape.frame++
		case funcReadBack:
			if tape.recPos == 0 {
				return 0, TapeEOR
			}
			data, err = tape.readPrevFrame()
			tape.recPos--
			tape.frame--
		}

	default:
		return 0, TapeTYPE
	}
	return data, err
}

// Write one frame to tape.
func (tape *TapeContext) WriteFrame(data byte) error {
	// Error if not attached.
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	// For P7B format for begin set IRG flag.
	if tape.format == TapeFmtP7B {
		data &= ^p7bIRG
		if tape.recPos == 0 {
			data |= p7bIRG
		}
	}
	tape.lrecl++
	tape.frame++
	tape.recPos++
	return tape.writeNextFrame(data)
}

// Finsh a record.
func (tape *TapeContext) FinishRecord() error {
	// Error if not attached.
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	// If there was tape mark, nothing more to do.
	if tape.mark {
		tape.mark = false
		return nil
	}

	var err error
	// Finish record
	switch tape.format {
	case TapeFmtTap, TapeFmtE11:
		err = tape.finishTAPfunc()
	case TapeFmtP7B:
		switch tape.mode {
		case funcRead, funcReadBack:
			for tape.lrecl != 2 {
				_, err = tape.ReadFrame()
				if errors.Is(err, TapeEOR) {
					return nil
				}
				if err != nil {
					break
				}
			}
		}
	case TapeFmtAWS:
		err = tape.finishAWSfunc()
	default:
		return TapeTYPE
	}
	tape.mode = funcNone
	return err
}

// Start rewind.
func (tape *TapeContext) StartRewind() error {
	// Error if not attached.
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	// If buffer dirty, flush it to file.
	if tape.dirty {
		_, _ = tape.file.Seek(tape.position, io.SeekStart)
		n, err := tape.file.Write(tape.buffer[:tape.bufLen])
		if err != nil {
			return err
		}
		if n != tape.bufLen {
			return errors.New("Write error on: " + tape.file.Name())
		}
		tape.dirty = false
	}
	tape.bufPos = 0
	tape.bufLen = 0
	return nil
}

// Rewind tape by number of frames.
func (tape *TapeContext) RewindFrames(frames int) bool {
	// If we hit beginning of tape set position to zero.
	if tape.frame < frames {
		tape.frame = 0
		tape.position = 0
		tape.mark = false
		tape.eot = false
		tape.bot = true
		return true
	}
	tape.frame -= frames
	return false
}

func NewTapeContext() *TapeContext {
	return &TapeContext{}
}

// Finish TAP operations.
func (tape *TapeContext) finishTAPfunc() error {
	switch tape.mode {
	case funcRead:
		// Make sure we read all of record.
		for tape.recPos < tape.lrecl {
			_, err := tape.readNextFrame()
			if err != nil {
				return err
			}
			tape.recPos++
		}

		// If Tap format and odd record length skip padding
		if tape.format == TapeFmtTap && (tape.lrecl&1) != 0 {
			_, err := tape.readNextFrame()
			if err != nil {
				return err
			}
		}

		recLen := [4]byte{}
		var err error
		for i := range 4 {
			recLen[i], err = tape.readNextFrame()
			if err != nil {
				return err
			}
		}
		lrecl := uint32(recLen[0]) | (uint32(recLen[1]) << 8) |
			(uint32(recLen[2]) << 16) | (uint32(recLen[3]) << 24)
		if lrecl != tape.lrecl {
			return TapeFORMAT
		}

	case funcWrite:
		// If TAP format, and odd record insert pad character
		if tape.format == TapeFmtTap && (tape.lrecl&1) != 0 {
			_ = tape.writeNextFrame(0)
		}

		lrecl := [4]byte{
			byte(tape.lrecl & 0xff),
			byte((tape.lrecl >> 8) & 0xff),
			byte((tape.lrecl >> 16) & 0xff),
			byte((tape.lrecl >> 24) & 0xff),
		}

		// Write ending and beginning record length
		for _, d := range lrecl {
			err := tape.writePrevByte(d)
			if err != nil {
				return err
			}

			err = tape.writeNextFrame(d)
			if err != nil {
				return err
			}
		}

	case funcReadBack:
		// Read rest of record if still data.
		for tape.recPos > 0 {
			_, err := tape.readPrevFrame()
			if err != nil {
				return err
			}
			tape.recPos--
		}

		// Read in header.
		recLen := [4]byte{}
		for i := 3; i >= 0; i-- {
			var err error
			recLen[i], err = tape.readPrevFrame()
			//		fmt.Printf(" F: %d %02x\n", i, recLen[i])
			if err != nil {
				return err
			}
		}

		// Make sure header matches
		lrecl := uint32(recLen[0]) | (uint32(recLen[1]) << 8) |
			(uint32(recLen[2]) << 16) | (uint32(recLen[3]) << 24)
		if lrecl != tape.lrecl {
			return TapeFORMAT
		}
	}
	return nil
}

// Finish TAP operations.
func (tape *TapeContext) finishAWSfunc() error {
	switch tape.mode {
	case funcRead:
		// Make sure we read all of record.
		for tape.recPos < tape.lrecl {
			_, err := tape.readNextFrame()
			if err != nil {
				return err
			}
			tape.recPos++
		}

		//  Read record header
		hdr := [6]byte{}
		var err error
		for i := range 6 {
			hdr[i], err = tape.readNextFrame()
			if err != nil {
				return err
			}
		}

		lrecl := (uint32(hdr[3]) << 8) | uint32(hdr[2])
		fmt.Printf("Header %02x %02x\n", hdr[4], hdr[5])
		if lrecl != tape.lrecl {
			return TapeFORMAT
		}

		// Check if tape mark.
		if hdr[4] == 0x4 {
			tape.mark = true
		}

	case funcWrite:
		// If TAP format, and odd record insert pad character
		if tape.format == TapeFmtTap && (tape.lrecl&1) != 0 {
			_ = tape.writeNextFrame(0)
		}

		lrecl := [4]byte{
			byte(tape.lrecl & 0xff),
			byte((tape.lrecl >> 8) & 0xff),
			byte((tape.lrecl >> 16) & 0xff),
			byte((tape.lrecl >> 24) & 0xff),
		}

		// Write ending and beginning record length
		for _, d := range lrecl {
			err := tape.writePrevByte(d)
			if err != nil {
				return err
			}

			err = tape.writeNextFrame(d)
			if err != nil {
				return err
			}
		}

	case funcReadBack:
		// Make sure we read all of record.
		for tape.recPos < tape.lrecl {
			_, err := tape.readNextFrame()
			if err != nil {
				return err
			}
			tape.recPos++
		}

		//  Read record header
		hdr := [6]byte{}
		var err error
		for i := range 6 {
			hdr[i], err = tape.readNextFrame()
			if err != nil {
				return err
			}
		}

		lrecl := (uint32(hdr[1]) << 8) | uint32(hdr[0])
		fmt.Printf("Header %02x %02x\n", hdr[4], hdr[5])
		if lrecl != tape.lrecl {
			return TapeFORMAT
		}

		// Check if tape mark.
		if hdr[4] == 0x4 {
			tape.mark = true
		}
	}
	return nil
}

// Read next frame from tape buffer.
func (tape *TapeContext) readNextFrame() (byte, error) {
	if tape.file == nil {
		return 0, errors.New("tape not attached")
	}
	// Check if at end of buffer
	err := tape.flushBuffer()
	if err != nil {
		return 0, err
	}
	err = tape.readBuffer()
	if err != nil {
		return 0, err
	}
	data := tape.buffer[tape.bufPos]
	tape.bufPos++
	return data, nil
}

// Peek at next frame from tape buffer.
func (tape *TapeContext) peekNextFrame() (byte, error) {
	if tape.file == nil {
		return 0, errors.New("tape not attached")
	}
	// Check if at end of buffer
	err := tape.flushBuffer()
	if err != nil {
		return 0, err
	}
	err = tape.readBuffer()
	if err != nil {
		return 0, err
	}
	data := tape.buffer[tape.bufPos]
	return data, nil
}

// Write character to tape.
func (tape *TapeContext) writeNextFrame(data byte) error {
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	// Check if at end of buffer
	if tape.bufPos >= len(tape.buffer) {
		// If buffer is dirty flush it to the file
		if tape.dirty {
			_, _ = tape.file.Seek(tape.position, io.SeekStart)
			n, err := tape.file.Write(tape.buffer[:])
			if err != nil {
				return err
			}
			if n != tape.bufLen {
				return errors.New("Write error on: " + tape.file.Name())
			}
			tape.position += int64(tape.bufLen)
			tape.dirty = false
		}
		tape.bufLen = 0
		tape.bufPos = 0
	}

	// Save data and advance position in buffer
	tape.buffer[tape.bufPos] = data
	tape.bufPos++
	tape.dirty = true

	// Adjust length of buffer if more data
	if tape.bufPos > tape.bufLen {
		tape.bufLen = tape.bufPos
	}
	return nil
}

// Write to previous tape byte.
func (tape *TapeContext) writePrevByte(data byte) error {
	if tape.file == nil {
		return errors.New("tape not attached")
	}

	pos := tape.startRec - tape.position
	// Check within buffer
	//	fmt.Printf("Write previous %d[%d], pos=%d, %02x\n", tape.bufPos, tape.bufLen, pos, data)
	if pos >= 0 && pos < int64(tape.bufLen) {
		tape.buffer[pos] = data
		// if pos > int64(tape.bufLen) {
		// 	tape.bufLen = int(pos)
		// }
		tape.dirty = true
	} else {
		_, _ = tape.file.Seek(tape.startRec, io.SeekStart)
		_, err := tape.file.Write([]byte{data})
		if err != nil {
			return err
		}
	}
	tape.startRec++
	return nil
}

// Flush a buffer and read in new one if needed.
func (tape *TapeContext) flushBuffer() error {
	if tape.bufPos < tape.bufLen {
		return nil
	}
	// If buffer is dirty flush it to the file
	if tape.dirty {
		_, _ = tape.file.Seek(tape.position, io.SeekStart)
		n, err := tape.file.Write(tape.buffer[:tape.bufLen])
		if err != nil {
			return err
		}
		if n != tape.bufLen {
			return errors.New("Write error on: " + tape.file.Name())
		}
		tape.position += int64(tape.bufLen)
		tape.bufLen = 0
		tape.dirty = false
	}

	return nil
}

// Read in buffer.
func (tape *TapeContext) readBuffer() error {
	if tape.bufPos < tape.bufLen {
		return nil
	}
	var err error
	// Advance tape by size of buffer
	tape.position += int64(tape.bufLen)
	_, _ = tape.file.Seek(tape.position, io.SeekStart)
	tape.bufLen, err = tape.file.Read(tape.buffer[:])
	tape.bufPos = 0
	if errors.Is(err, io.EOF) {
		tape.eot = true
	}
	return err
}

// Read previous frame from tape.
func (tape *TapeContext) readPrevFrame() (byte, error) {
	if tape.file == nil {
		return 0, errors.New("tape not attached")
	}

	// Check if not at beginning buffer or buffer not empty
	if tape.bufPos != 0 && tape.bufLen != 0 {
		tape.bufPos--
		data := tape.buffer[tape.bufPos]
		return data, nil
	}

	// If buffer is dirty flush it to the file
	if tape.dirty {
		_, _ = tape.file.Seek(tape.position, io.SeekStart)
		n, err := tape.file.Write(tape.buffer[:tape.bufLen])
		if err != nil {
			return 0, err
		}
		if n != tape.bufLen {
			return 0, errors.New("Write error on: " + tape.file.Name())
		}
		tape.dirty = false
	}

	// If at beging of tape, return BOT status.
	if tape.bot {
		fmt.Println("At BOT")
		return 0, TapeBOT
	}

	// If at beging of file, set BOT.
	if tape.position == 0 {
		data := tape.buffer[tape.bufPos]
		tape.bot = true
		tape.bufPos = 0
		tape.bufLen = 0
		return data, TapeBOT
	}

	// Backup current buffer first position
	// Check if moved before start of tape.
	opos := -1
	if int(tape.position) < len(tape.buffer) {
		opos = int(tape.position)
		tape.position = 0
	} else {
		tape.position -= int64(len(tape.buffer))
	}

	// Fill buffer.
	_, _ = tape.file.Seek(tape.position, io.SeekStart)
	n, err := tape.file.Read(tape.buffer[:])
	tape.bufLen = n
	if err != nil {
		return 0, err
	}

	if opos == -1 {
		tape.bufPos = tape.bufLen
	} else {
		tape.bufPos = 0
	}

	tape.eot = false
	tape.bufPos--
	data := tape.buffer[tape.bufPos]
	return data, nil
}
