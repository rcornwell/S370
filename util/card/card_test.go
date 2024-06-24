/*
 * Card emulation test cases.
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
package card

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/rcornwell/S370/util/xlat"
)

// Create a card file with number of cards.
func createCardFile(file *os.File, cards int) {
	for i := range cards {
		fmt.Fprintf(file, "%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789\n", i)
	}
	file.Close()
}

var deck1 string
var deck2 string
var deck3 string
var deck4 string
var ctx *CardContext

func numToHol(v int) uint16 {
	//	h := uint16(0)
	h := uint16(0x200) >> v
	return h
}

func setupCardTest() error {
	f, err := os.CreateTemp("", "deck")
	if err != nil {
		return err
	}
	deck1 = f.Name()
	createCardFile(f, 10)
	f.Close()
	f, err = os.CreateTemp("", "deck")
	if err != nil {
		return err
	}
	deck2 = f.Name()
	createCardFile(f, 20)
	f.Close()
	f, err = os.CreateTemp("", "deck")
	if err != nil {
		return err
	}
	deck3 = f.Name()
	createCardFile(f, 30)
	f.Close()
	f, err = os.CreateTemp("", "deck")
	if err != nil {
		return err
	}
	deck4 = f.Name()
	createCardFile(f, 40)
	f.Close()
	return nil
}

func freeCtx() {
	if ctx == nil {
		return
	}
	if ctx.file != nil {
		ctx.file.Close()
	}
	ctx.deck = nil
	ctx = nil
}

func deleteTempFile() {
	os.Remove(deck1)
	os.Remove(deck2)
	os.Remove(deck3)
	os.Remove(deck4)
	freeCtx()
}

// Check that we can read a deck
func TestReadDeck(t *testing.T) {
	e := setupCardTest()
	defer deleteTempFile()
	if e != nil {
		t.Error(e)
		return
	}
	ctx = NewCardContext(MODE_TEXT)

	e = ctx.readDeck(deck1)
	if e != nil {
		t.Error(e)
		return
	}
	v := ctx.HopperSize()
	if v != 10 {
		t.Errorf("Read %d cards expected 10 cards", v)
	}
}

// Check that emoty_cards will remove all cards in hopper
func TestEmptyDeck(t *testing.T) {
	e := setupCardTest()
	defer deleteTempFile()
	if e != nil {
		t.Error(e)
		return
	}
	ctx = NewCardContext(MODE_TEXT)
	e = ctx.readDeck(deck1)
	if e != nil {
		t.Error(e)
		return
	}
	v := ctx.HopperSize()
	if v != 10 {
		t.Errorf("Read %d cards expected 10 cards", v)
	}
	ctx.EmptyDeck()

	v = ctx.HopperSize()
	if v != 0 {
		t.Errorf("EmptyDeck did not clear deck %d", v)
	}
}

// Check that we can stack read a deck
func TestReadDeckStack(t *testing.T) {
	e := setupCardTest()
	defer deleteTempFile()
	if e != nil {
		t.Error(e)
		return
	}
	ctx = NewCardContext(MODE_TEXT)

	e = ctx.readDeck(deck1)
	if e != nil {
		t.Error(e)
		return
	}
	v := ctx.HopperSize()
	if v != 10 {
		t.Errorf("Read %d cards expected 10 cards", v)
	}
	e = ctx.readDeck(deck2)
	if e != nil {
		t.Error(e)
		return
	}
	v = ctx.HopperSize()
	if v != 30 {
		t.Errorf("Read %d cards expected 30 cards", v)
	}
	e = ctx.readDeck(deck3)
	if e != nil {
		t.Error(e)
		return
	}
	v = ctx.HopperSize()
	if v != 60 {
		t.Errorf("Read %d cards expected 60 cards", v)
	}
	e = ctx.readDeck(deck4)
	if e != nil {
		t.Error(e)
		return
	}
	v = ctx.HopperSize()
	if v != 100 {
		t.Errorf("Read %d cards expected 100 cards", v)
	}
}

// Check that emoty_cards will remove all cards in hopper
func TestReadCard(t *testing.T) {
	e := setupCardTest()
	defer deleteTempFile()
	if e != nil {
		t.Error(e)
		return
	}
	ctx = NewCardContext(MODE_TEXT)
	e = ctx.readDeck(deck1)
	if e != nil {
		t.Error(e)
		return
	}
	v := ctx.HopperSize()
	if v != 10 {
		t.Errorf("Read %d cards expected 10 cards", v)
	}
	count := 0
	var err int
	for {
		_, err = ctx.ReadCard()
		if err != CARD_OK {
			break
		}
		count++
	}

	if err != CARD_EMPTY {
		t.Error("ReadCard did not return Empty Card")
	}

	if count != 10 {
		t.Errorf("ReadCard did not read 10 cards read: %d", v)
	}
	v = ctx.HopperSize()
	if v != 0 {
		t.Errorf("ReadCard hopper not empty %d", v)
	}
}

var testImage = [80]uint16{
	// Header
	0x000, 0x000, 0x000, 0x000, 0x000, 0x000,
	// A-Z
	0x900, 0x880, 0x840, 0x820, 0x810, 0x808, 0x804, 0x802, 0x801,
	0x500, 0x480, 0x440, 0x420, 0x410, 0x408, 0x404, 0x402, 0x401,
	0x280, 0x240, 0x220, 0x210, 0x208, 0x204, 0x202, 0x201,
	// 0-9
	0x200, 0x100, 0x080, 0x040, 0x020, 0x010, 0x008, 0x004, 0x002, 0x001,
	// A-Z
	0x900, 0x880, 0x840, 0x820, 0x810, 0x808, 0x804, 0x802, 0x801,
	0x500, 0x480, 0x440, 0x420, 0x410, 0x408, 0x404, 0x402, 0x401,
	0x280, 0x240, 0x220, 0x210, 0x208, 0x204, 0x202, 0x201,
	// 0-9
	0x200, 0x100, 0x080, 0x040, 0x020, 0x010, 0x008, 0x004, 0x002, 0x001,
}

// Verify that cards match hollerith values
func TestReadCardMatch(t *testing.T) {
	e := setupCardTest()
	defer deleteTempFile()
	if e != nil {
		t.Error(e)
		return
	}
	ctx = NewCardContext(MODE_TEXT)
	e = ctx.readDeck(deck1)
	if e != nil {
		t.Error(e)
		return
	}
	v := ctx.HopperSize()
	if v != 10 {
		t.Errorf("Read %d cards expected 10 cards", v)
	}
	var err int
	for i := range 10 {
		var c Card
		c, err = ctx.ReadCard()
		if err != CARD_OK {
			break
		}

		cor := [80]uint16{}

		for j := range 80 {
			cor[j] = testImage[j]
		}
		// Set count into testImage
		for j := range 5 {
			cor[j] = 0x200
		}
		cor[4] = numToHol(i % 10)
		cor[3] = numToHol(i / 10)

		for j := range 80 {
			if cor[j] != c.Image[j] {
				t.Errorf(" Card %d failed to match %d %03x != %03x", i, j, cor[j], c.Image[j])
				break
			}
		}
	}

	_, err = ctx.ReadCard()

	if err != CARD_EMPTY {
		t.Error("ReadCard did not return Empty Card")
	}

	v = ctx.HopperSize()
	if v != 0 {
		t.Errorf("ReadCard hopper not empty %d", v)
	}
}

// Test that blank cards creates requested number of blank cards
func TestReadBlankDeck(t *testing.T) {
	ctx = NewCardContext(MODE_TEXT)
	defer freeCtx()
	ctx.BlankDeck(10)

	v := ctx.HopperSize()
	if v != 10 {
		t.Errorf("Read %d cards expected 10 cards", v)
	}
	var err int
	for i := range 10 {
		var c Card
		c, err = ctx.ReadCard()
		if err != CARD_OK {
			break
		}

		for j := range 80 {
			if c.Image[j] != 0 {

				t.Errorf("Blank card not blank %d", i)
				break
			}
		}
	}
	_, err = ctx.ReadCard()

	if err != CARD_EMPTY {
		t.Error("ReadCard did not return Empty Card")
	}

	v = ctx.HopperSize()
	if v != 0 {
		t.Errorf("ReadCard hopper not empty %d", v)
	}
}

// Test punch of a blank deck
func TestPunchCardBlank(t *testing.T) {
	ctx = NewCardContext(MODE_TEXT)
	defer freeCtx()

	// Create blank card
	c := Card{}
	for i := range len(c.Image) {
		c.Image[i] = 0
	}
	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	err = ctx.Attach(name, MODE_TEXT, true, false)
	if err != nil {
		t.Error(err)
		return
	}

	for range 50 {
		ctx.PunchCard(c)
	}

	ctx.Detach()
	f, err = os.Open(name)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	count := 0
	for {
		line, e := reader.ReadString('\n')
		if e != nil {
			break
		}
		if line[0] != '\n' {
			t.Error("Non blank card found")
		} else {
			count++
		}
	}
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

func TestPunchCardDeck(t *testing.T) {
	ctx = NewCardContext(MODE_TEXT)
	defer freeCtx()

	// Create test card
	c := Card{}
	for i := range len(c.Image) {
		c.Image[i] = testImage[i]
	}
	// Set count into testImage
	for i := range 5 {
		c.Image[i] = 0x200
	}
	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	err = ctx.Attach(name, MODE_TEXT, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	for i := range 50 {
		c.Image[4] = numToHol(i % 10)
		c.Image[3] = numToHol(i / 10)
		ctx.PunchCard(c)
	}

	ctx.Detach()
	f, err = os.Open(name)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	count := 0
	for {
		line, e := reader.ReadString('\n')
		if e != nil {
			break
		}
		s := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789\n", count)
		if line != s {
			t.Errorf("Card %d did not match", count)
			t.Errorf("Correct: %s", s)
			t.Errorf("Got:     %s", line)
		}
		count++
	}
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

var ebcdicString = [80]uint8{
	0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0x40, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8, 0xc9, 0xd1,
	0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9,
	0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8, 0xf9, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6,
	0xc7, 0xc8, 0xc9, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xe2, 0xe3, 0xe4, 0xe5,
	0xe6, 0xe7, 0xe8, 0xe9, 0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8, 0xf9, 0x40, 0x40,
}

// Try to punch an EBCDIC deck
func TestPunchCardEBCDIC(t *testing.T) {
	ctx = NewCardContext(MODE_TEXT)
	defer freeCtx()

	// Create test card
	c := Card{}
	for i := range len(c.Image) {
		c.Image[i] = testImage[i]
	}
	// Set count into testImage
	for i := range 5 {
		c.Image[i] = 0x200
	}
	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}

	name := f.Name()

	f.Close()
	defer os.Remove(name)

	err = ctx.Attach(name, MODE_EBCDIC, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	for i := range 50 {
		c.Image[4] = numToHol(i % 10)
		c.Image[3] = numToHol(i / 10)
		ctx.PunchCard(c)
	}

	ctx.Detach()
	f, err = os.Open(name)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	count := 0
	for {
		var buf [80]uint8
		n, e := f.Read(buf[:])

		if e != nil || n != 80 {
			break
		}
		ebcdicString[3] = 0xf0 + uint8(count/10)
		ebcdicString[4] = 0xf0 + uint8(count%10)
		for j := range 80 {
			//			t.Logf("Pos: %d %02x", j, buf[j])
			if buf[j] != ebcdicString[j] {
				t.Errorf(" Card %d failed to match %d %03x != %03x", count, j, buf[j], ebcdicString[j])
			}
		}
		count++
	}
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to read an EBCDIC deck
func TestReadDeckEBCDIC(t *testing.T) {
	ctx = NewCardContext(MODE_TEXT)
	defer freeCtx()

	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	defer os.Remove(name)
	for i := 0; i < 50; i++ {
		// Create test card
		ebcdicString[3] = 0xf0 + uint8(i/10)
		ebcdicString[4] = 0xf0 + uint8(i%10)

		n, e := f.Write(ebcdicString[:])

		if e != nil || n != 80 {
			t.Error("Unable to create file")
			break
		}

	}
	f.Close()

	err = ctx.Attach(name, MODE_EBCDIC, false, false)
	if err != nil {
		t.Error(err)
		return
	}
	count := 0
	for {
		var c Card
		var e int
		c, e = ctx.ReadCard()
		if e == CARD_EMPTY {
			break
		}
		if e != CARD_OK {
			t.Error("Card not ok")
			break
		}

		cor := [80]uint16{}

		for j := range 80 {
			cor[j] = testImage[j]
		}
		// Set count into testImage
		for j := range 5 {
			cor[j] = 0x200
		}

		cor[3] = numToHol(count / 10)
		cor[4] = numToHol(count % 10)

		for j := range 80 {
			if cor[j] != c.Image[j] {
				t.Errorf(" Card %d failed to match %d %03x != %03x", count, j, cor[j], c.Image[j])
				break
			}
		}
		count++
	}
	ctx.Detach()
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to punch an binary deck
func TestPunchCardBinary(t *testing.T) {
	ctx = NewCardContext(MODE_BIN)
	defer freeCtx()

	// Create test card
	c := Card{}

	// Create test file
	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	err = ctx.Attach(name, MODE_BIN, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	// Copy test image to card
	for i := range len(c.Image) {
		c.Image[i] = testImage[i]
	}
	// Blank out sequence
	for i := range 5 {
		c.Image[i] = 0x200
		testImage[i] = 0x200
	}
	for i := range 50 {
		// Set count into testImage
		c.Image[4] = numToHol(i % 10)
		c.Image[3] = numToHol(i / 10)
		ctx.PunchCard(c)
	}

	ctx.Detach()
	f, err = os.Open(name)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	count := 0

	for {
		var buf [160]byte
		n, e := f.Read(buf[:])

		if e != nil || n != len(buf) {
			break
		}
		testImage[4] = uint16(0x200) >> (count % 10)
		testImage[3] = uint16(0x200) >> (count / 10)

		for j := range len(testImage) {
			l := byte((testImage[j] & 0x00f) << 4)
			h := byte((testImage[j] & 0xff0) >> 4)

			if l != buf[j*2] || h != buf[(j*2)+1] {
				t.Errorf(" Card %d failed to match %d %02x %02x != %03x", count, j, buf[j*2], buf[(j*2)+1], testImage[j])
			}
		}
		count++
	}
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to read an Binary deck
func TestReadDeckBinary(t *testing.T) {
	ctx = NewCardContext(MODE_TEXT)
	defer freeCtx()

	f, err := os.CreateTemp("", "deck")
	if err != nil {
		reader := bufio.NewReader(f)
		count := 0
		for {
			line, e := reader.ReadString('\n')
			if e != nil {
				break
			}
			s := fmt.Sprintf("%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789\n", count)
			if line != s {
				t.Errorf("Card %d did not match", count)
				t.Errorf("Correct: %s", s)
				t.Errorf("Got:     %s", line)
			}
			count++
		}
		t.Error(err)
		return
	}
	name := f.Name()
	defer os.Remove(name)
	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}
	for i := range 50 {
		var buf [160]byte

		// Create test card image
		testImage[4] = numToHol(i % 10)
		testImage[3] = numToHol(i / 10)

		for j := range 80 {
			buf[j*2] = byte((testImage[j] & 0x00f) << 4)
			buf[(j*2)+1] = byte((testImage[j] & 0xff0) >> 4)

		}

		n, e := f.Write(buf[:])

		if e != nil || n != len(buf) {
			t.Error("Unable to create file")
			break
		}

	}
	f.Close()

	err = ctx.Attach(name, MODE_BIN, false, false)
	if err != nil {
		t.Error(err)
		return
	}
	count := 0
	for {
		var c Card
		var e int
		c, e = ctx.ReadCard()
		if e == CARD_EMPTY {
			break
		}
		if e != CARD_OK {
			t.Error("Card not ok")
			break
		}

		testImage[4] = numToHol(count % 10)
		testImage[3] = numToHol(count / 10)

		for j := range 80 {
			if testImage[j] != c.Image[j] {
				t.Errorf(" Card %d failed to match %d %03x != %03x", count, j, c.Image[j], testImage[j])
			}
		}
		count++
	}
	ctx.Detach()
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to punch an CBN deck
func TestPunchCardCBN(t *testing.T) {
	ctx = NewCardContext(MODE_CBN)
	defer freeCtx()

	// Create test card
	c := Card{}

	// Create test file
	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	err = ctx.Attach(name, MODE_CBN, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	// Copy test image to card
	for i := range len(c.Image) {
		c.Image[i] = testImage[i]
	}
	// Blank out sequence
	for i := range 5 {
		c.Image[i] = 0x200
	}
	for i := range 50 {
		// Set count into testImage
		c.Image[4] = numToHol(i % 10)
		c.Image[3] = numToHol(i / 10)
		ctx.PunchCard(c)
	}

	ctx.Detach()
	f, err = os.Open(name)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	count := 0

	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}
	for {
		var buf [160]byte
		n, e := f.Read(buf[:])

		if e != nil || n != len(buf) {
			break
		}
		testImage[4] = numToHol(count % 10)
		testImage[3] = numToHol(count / 10)

		for j := range len(testImage) {
			// Check for start of record has record mark
			if j == 0 {
				if (buf[j] & 0x80) == 0 {
					t.Error("First character is not beginning of record")
				}
				buf[j] &= 0x7f
			}
			l := byte((testImage[j] >> 6) & 077)
			h := byte(testImage[j] & 077)
			// Or in correct parity
			l |= xlat.ParityTable[l&077] ^ 0100
			h |= xlat.ParityTable[h&077] ^ 0100

			if l != buf[j*2] || h != buf[(j*2)+1] {
				t.Errorf(" Card %d failed to match %d %02x %02x != %03x", count, j, buf[j*2], buf[(j*2)+1], testImage[j])
			}
		}
		count++
	}
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to read an CBN deck
func TestReadDeckCBN(t *testing.T) {
	ctx = NewCardContext(MODE_CBN)
	defer freeCtx()

	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	defer os.Remove(name)
	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}
	for i := range 50 {
		var buf [160]byte

		// Create test card image
		testImage[4] = numToHol(i % 10)
		testImage[3] = numToHol(i / 10)

		for j := range 80 {
			l := byte((testImage[j] >> 6) & 077)
			h := byte(testImage[j] & 077)
			buf[j*2] = (xlat.ParityTable[l&077] ^ 0100) | l
			buf[(j*2)+1] = (xlat.ParityTable[h&077] ^ 0100) | h
		}

		buf[0] |= 0200 // Record mark
		n, e := f.Write(buf[:])

		if e != nil || n != len(buf) {
			t.Error("Unable to create file")
			break
		}

	}
	f.Close()

	err = ctx.Attach(name, MODE_CBN, false, false)
	if err != nil {
		t.Error(err)
		return
	}
	count := 0
	for {
		var c Card
		var e int
		c, e = ctx.ReadCard()
		if e == CARD_EMPTY {
			break
		}
		if e != CARD_OK {
			t.Error("Card not ok")
			break
		}

		testImage[4] = uint16(0x200) >> (count % 10)
		testImage[3] = uint16(0x200) >> (count / 10)

		for j := range 80 {
			if testImage[j] != c.Image[j] {
				t.Errorf(" Card %d failed to match %d %03x != %03x", count, j, c.Image[j], testImage[j])
			}
		}
		count++
	}
	ctx.Detach()
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to punch an BCD deck
func TestPunchCardBCD(t *testing.T) {
	ctx = NewCardContext(MODE_BCD)
	defer freeCtx()

	// Create test card
	c := Card{}

	// Create test file
	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	err = ctx.Attach(name, MODE_BCD, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	// Copy test image to card
	for i := range len(c.Image) {
		c.Image[i] = testImage[i]
	}
	// Blank out sequence
	for i := range 5 {
		c.Image[i] = 0x200
	}
	for i := range 50 {
		// Set count into testImage
		c.Image[4] = numToHol(i % 10)
		c.Image[3] = numToHol(i / 10)
		ctx.PunchCard(c)
	}

	ctx.Detach()
	f, err = os.Open(name)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	count := 0

	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}
	for {
		var buf [80]byte
		n, e := f.Read(buf[:])

		if e != nil {
			break
		}

		// Fill out rest of buffer with zeros
		if n < len(buf) {
			for n++; n < len(buf); n++ {
				buf[n] = 0
			}
		}
		testImage[4] = numToHol(count % 10)
		testImage[3] = numToHol(count / 10)

		for j := range len(testImage) {
			// Check for start of record has record mark
			if j == 0 {
				if (buf[j] & 0x80) == 0 {
					t.Error("First character is not beginning of record")
				}
				buf[j] &= 0x7f
			} else if (buf[j] & 0x80) != 0 {
				// Got end of record, back us up in file
				_, _ = f.Seek(int64(j-len(testImage)), 1)
				// Fill remainder of buffer with zeros.
				for k := j; k < len(testImage); k++ {
					buf[k] = 0
				}
			}
			ch := HolToBcd(testImage[j])
			// Or in correct parity
			ch |= xlat.ParityTable[ch&077]

			if ch != buf[j] {
				t.Errorf(" Card %d failed to match %d %02x != %02x", count, j, buf[j], ch)
			}
		}
		count++
	}
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to read an BCD deck
func TestReadDeckBCD(t *testing.T) {
	ctx = NewCardContext(MODE_BCD)
	defer freeCtx()

	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	defer os.Remove(name)
	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}
	for i := range 50 {
		var buf [80]byte

		// Create test card image
		testImage[4] = numToHol(i % 10)
		testImage[3] = numToHol(i / 10)

		for j := range len(testImage) {
			ch := HolToBcd(testImage[j])
			if ch == 0 { // Translate space to space
				ch = 020
			}
			buf[j] = xlat.ParityTable[ch&077] | ch
		}

		buf[0] |= 0200 // Record mark
		n, e := f.Write(buf[:])

		if e != nil || n != len(buf) {
			t.Error("Unable to create file")
			break
		}

	}
	f.Close()

	err = ctx.Attach(name, MODE_BCD, false, false)
	if err != nil {
		t.Error(err)
		return
	}
	count := 0
	for {
		var c Card
		var e int
		c, e = ctx.ReadCard()
		if e == CARD_EMPTY {
			break
		}
		if e != CARD_OK {
			t.Error("Card not ok")
			break
		}

		testImage[4] = numToHol(count % 10)
		testImage[3] = numToHol(count / 10)

		for j := range 80 {
			if testImage[j] != c.Image[j] {
				t.Errorf(" Card %d failed to match %d %03x != %03x", count, j, c.Image[j], testImage[j])
			}
		}
		count++
	}
	ctx.Detach()
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to punch an octal deck
func TestPunchCardOctal(t *testing.T) {
	ctx = NewCardContext(MODE_OCTAL)
	defer freeCtx()

	// Create test card
	c := Card{}

	// Create test file
	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	err = ctx.Attach(name, MODE_OCTAL, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	// Copy test image to card
	for i := range len(c.Image) {
		c.Image[i] = testImage[i]
	}
	// Blank out sequence
	for i := range 5 {
		c.Image[i] = 0x200
	}
	for i := range 50 {
		// Set count into testImage
		c.Image[4] = numToHol(i % 10)
		c.Image[3] = numToHol(i / 10)
		ctx.PunchCard(c)
	}

	ctx.Detach()
	f, err = os.Open(name)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	count := 0

	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}
	reader := bufio.NewReader(f)

	for {
		line, e := reader.ReadString('\n')
		if e != nil {
			break
		}
		for i := range 5 {
			testImage[i] = 0x200
		}
		var buf [512]byte
		var l int

		// Create test card image
		testImage[4] = numToHol(count % 10)
		testImage[3] = numToHol(count / 10)
		buf[0] = '~'
		buf[1] = 'r'
		buf[2] = 'a'
		buf[3] = 'w'

		for l = 79; l >= 0; l-- {
			if testImage[l] != 0 {
				l++
				break
			}
		}
		p := 4
		for j := range l {
			col := testImage[j]
			buf[p] = byte((col>>9)&07) + '0'
			buf[p+1] = byte((col>>6)&07) + '0'
			buf[p+2] = byte((col>>3)&07) + '0'
			buf[p+3] = byte((col>>0)&07) + '0'
			p += 4
		}
		buf[p] = '\n'
		p++

		s := string(buf[0:p])
		if line != s {
			t.Errorf("Card %d did not match", count)
			t.Errorf("Correct: %s", s)
			t.Errorf("Got:     %s", line)
		}
		count++
	}
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

// Try to read an octal deck
func TestReadDeckOctal(t *testing.T) {
	ctx = NewCardContext(MODE_TEXT)
	defer freeCtx()

	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	defer os.Remove(name)
	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}
	for i := range 50 {
		var buf [512]byte

		// Create test card image
		testImage[4] = numToHol(i % 10)
		testImage[3] = numToHol(i / 10)
		buf[0] = '~'
		buf[1] = 'r'
		buf[2] = 'a'
		buf[3] = 'w'

		p := 4
		for j := range len(testImage) {
			col := testImage[j]
			buf[p] = byte((col>>9)&07) + '0'
			buf[p+1] = byte((col>>6)&07) + '0'
			buf[p+2] = byte((col>>3)&07) + '0'
			buf[p+3] = byte((col>>0)&07) + '0'
			p += 4
		}
		buf[p] = '\n'
		p++

		n, e := f.Write(buf[0:p])

		if e != nil || n != p {
			t.Error("Unable to create file")
			break
		}

	}
	f.Close()

	err = ctx.Attach(name, MODE_TEXT, false, false)
	if err != nil {
		t.Error(err)
		return
	}
	count := 0
	for {
		var c Card
		var e int
		c, e = ctx.ReadCard()
		if e == CARD_EMPTY {
			break
		}
		if e != CARD_OK {
			t.Error("Card not ok")
			break
		}

		testImage[4] = numToHol(count % 10)
		testImage[3] = numToHol(count / 10)

		for j := range 80 {
			if testImage[j] != c.Image[j] {
				t.Errorf(" Card %d failed to match %d %03x != %03x", count, j, c.Image[j], testImage[j])
			}
		}
		count++
	}
	ctx.Detach()
	if count != 50 {
		t.Errorf(" Wanted 50 cards got %d", count)
	}
}

func TestReadDeckAuto(t *testing.T) {
	ctx = NewCardContext(MODE_AUTO)
	defer freeCtx()

	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	defer os.Remove(name)
	f.Close()

	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}

	err = ctx.Attach(name, MODE_TEXT, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	var c Card
	// Copy test image to card
	for i := range len(c.Image) {
		c.Image[i] = testImage[i]
	}
	for i := range 60 {
		// Create test card image
		c.Image[4] = numToHol(i % 10)
		c.Image[3] = numToHol(i / 10)
		ctx.PunchCard(c)
		switch i {
		case 10:
			ctx.mode = MODE_BCD
		case 20:
			ctx.mode = MODE_BIN
		case 30:
			ctx.mode = MODE_OCTAL
		case 40:
			ctx.mode = MODE_CBN
		case 50:
			ctx.mode = MODE_TEXT
		}

	}
	ctx.Detach()

	err = ctx.Attach(name, MODE_AUTO, false, false)
	if err != nil {
		t.Error(err)
		return
	}
	count := 0
	var match [80]uint16

	for i := range len(testImage) {
		match[i] = testImage[i]
	}
	for {
		var c Card
		var e int
		c, e = ctx.ReadCard()
		if e == CARD_EMPTY {
			break
		}
		if e != CARD_OK {
			t.Error("Card not ok")
			break
		}

		match[4] = numToHol(count % 10)
		match[3] = numToHol(count / 10)

		for j := range 80 {
			// BCD translates space differntly
			if count >= 10 && count <= 20 {
				if c.Image[j] == 0x82 {
					c.Image[j] = 0
				}
			}
			if match[j] != c.Image[j] {
				t.Errorf(" Card %d failed to match %d %03x != %03x", count, j, c.Image[j], match[j])
			}
		}
		count++
	}
	ctx.Detach()
	if count != 60 {
		t.Errorf(" Wanted 60 cards got %d", count)
	}
}

// Test special cards and EOF
func TestReadDeckSpecial(t *testing.T) {
	ctx = NewCardContext(MODE_AUTO)
	defer freeCtx()

	f, err := os.CreateTemp("", "deck")
	if err != nil {
		t.Error(err)
		return
	}
	name := f.Name()
	defer os.Remove(name)
	fmt.Fprintf(f, "%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789\n", 0) // 1
	fmt.Fprint(f, "~eor\n")                                                                              // 2
	fmt.Fprint(f, "~eof\n")                                                                              // 3
	fmt.Fprint(f, "~eoi\n")                                                                              // 4
	fmt.Fprint(f, "~\n")                                                                                 // 5
	fmt.Fprintf(f, "%05d ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789\n", 0) // 6
	f.Close()

	err = ctx.Attach(name, MODE_AUTO, false, true)
	if err != nil {
		t.Error(err)
		return
	}

	// Blank out sequence
	for i := range 5 {
		testImage[i] = 0x200
	}
	var c Card
	var e int
	c, e = ctx.ReadCard()
	if e == CARD_EMPTY {
		t.Error("Card 1 empty")
		goto card2
	}
	if e != CARD_OK {
		t.Error("Card 1 not ok")
		goto card2
	}

	for j := range 80 {
		if testImage[j] != c.Image[j] {
			t.Errorf(" Card %d failed to match %d %03x != %03x", 1, j, c.Image[j], testImage[j])
		}
	}

card2:
	var match [80]uint16
	for i := range len(match) {
		match[i] = 0
	}
	c, e = ctx.ReadCard()
	if e == CARD_EMPTY {
		t.Error("Card 2 empty")
		goto card3
	}
	if e != CARD_OK {
		t.Error("Card 2 not ok")
		goto card3
	}
	match[0] = 07
	for j := range 80 {

		if match[j] != c.Image[j] {
			t.Errorf(" Card %d failed to match %d %03x != %03x", 2, j, c.Image[j], match[j])
		}
	}
card3:
	c, e = ctx.ReadCard()
	if e == CARD_EMPTY {
		t.Error("Card 3 empty")
		goto card4
	}
	if e != CARD_OK {
		t.Error("Card 3 not ok")
		goto card4
	}
	match[0] = 015
	for j := range 80 {

		if match[j] != c.Image[j] {
			t.Errorf(" Card %d failed to match %d %03x != %03x", 3, j, c.Image[j], match[j])
		}
	}
card4:
	c, e = ctx.ReadCard()
	if e == CARD_EMPTY {
		t.Error("Card 4 empty")
		goto card5
	}
	if e != CARD_OK {
		t.Error("Card 4 not ok")
		goto card5
	}
	match[0] = 017
	for j := range 80 {

		if match[j] != c.Image[j] {
			t.Errorf(" Card %d failed to match %d %03x != %03x", 4, j, c.Image[j], match[j])
		}
	}
card5:
	_, e = ctx.ReadCard()
	switch e {
	case CARD_EMPTY:
		t.Error("Card 5 empty")
	case CARD_EOF:
	case CARD_OK:
		t.Error("Card 5 not ok")
	case CARD_ERROR:
		t.Error("Card 5 in error")
	}

	c, e = ctx.ReadCard()
	if e == CARD_EMPTY {
		t.Error("Card 6 empty")
		goto card7
	}
	if e != CARD_OK {
		t.Error("Card 6 not ok")
		goto card7
	}

	for j := range 80 {
		if testImage[j] != c.Image[j] {
			t.Errorf(" Card %d failed to match %d %03x != %03x", 6, j, c.Image[j], testImage[j])
		}
	}
card7:
	_, e = ctx.ReadCard()
	switch e {
	case CARD_EMPTY:
		t.Error("Card 7 empty")
	case CARD_EOF:
	case CARD_OK:
		t.Error("Card 7 not ok")
	case CARD_ERROR:
		t.Error("Card 7 in error")
	}
	_, e = ctx.ReadCard()
	switch e {
	case CARD_EMPTY:
	case CARD_EOF:
		t.Error("Card 8 EOF")
	case CARD_OK:
		t.Error("Card 8 not ok")
	case CARD_ERROR:
		t.Error("Card 8 in error")
	}
	ctx.Detach()

}
