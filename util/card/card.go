/*
 * Generic Card read/punch routines for simulators.
 *
 * Copyright (c) 2021-2024, Richard Cornwell
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
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/rcornwell/S370/util/xlat"
)

const (
	MODE_AUTO int = iota + 1
	MODE_TEXT
	MODE_EBCDIC
	MODE_BIN
	MODE_OCTAL
	MODE_BCD
	MODE_CBN
)

const (
	CARD_OK int = iota + 1
	CARD_EOF
	CARD_EMPTY
	CARD_ERROR
)

const (
	TYPE_029 int = iota + 1
	TYPE_026
	TYPE_ASCII
)

const (
	flag_eof uint16 = 0x1000
	flag_err uint16 = 0x2000
	flag_dat uint16 = 0x8000
)

type Card struct {
	Image [80]uint16 // Image of individual cards
}

type CardContext struct {
	file        *os.File // file handle
	mode        int      // Current input/output mode
	hopperCards int      // Number of cards in hopper
	hopperPos   int      // Position in hopper
	eofPending  bool     // Next return should be EOF
	table       int      // Translation table to use
	deck        []Card   // Card images
}

type cardBuffer struct {
	buffer [8192 + 500]uint8 // Buffer data
	len    int               // Amount of data in buffer
}

var emptyCard Card

/* Generic Card read/punch routines for simulators.

   Input formats are accepted in a variaty of formats:
        Standard ASCII: one record per line.
                returns are ignored.
                tabs are expanded to modules 8 characters.
                ~ in first column is treated as a EOF.

        Binary Card format:
                Each record 160 characters.
                First characters 6789----
                Second character 21012345
                                 111
                Top 4 bits of second character are 0.
                It is unlikely that any other format could
                look like this.

    ASCII mode recognizes some additional forms of input which allows the
    intermixing of binary cards with text cards.

    Lines beginning with ~raw are taken as a number of 4 digit octal values
    with represent each column of the card from 12 row down to 9 row. If there
    is not enough octal numbers to span a full card the remainder of the
    card will not be punched.

    Also ~eor, will generate a 7/8/9 punch card. An ~eof will gernerate a
    6/7/9 punch card, and a ~eoi will generate a 6/7/8/9 punch.

    A single line of ~ will set the EOF flag when that card is read.

    For autodetection of card format, there can be no parity errors.
    All undeterminate formats are treated as ASCII.

    Auto output format is ASCII if card has only printable characters
    or card format binary.
*/

// Return if attached to a file
func (ctx *CardContext) Attached() bool {
	return ctx.file != nil
}

// Attach a context to a file
func (ctx *CardContext) Attach(fileName string, mode int, write bool, eof bool) error {
	var err error

	ctx.file = nil
	ctx.mode = mode
	if write {
		var file *os.File
		file, err = os.Create(fileName)
		if err != nil {
			return err
		}
		ctx.file = file
		ctx.deck = []Card{}
	} else {
		err = ctx.readDeck(fileName)
		if eof {
			ctx.SetEOF()
		}
	}
	return err
}

// Detach from file
func (ctx *CardContext) Detach() {
	if ctx.file != nil {
		ctx.file.Close()
		ctx.file = nil
	}
	ctx.hopperCards = 0
	ctx.hopperPos = 0
}

// Initialize back translation tables
func init() {
	for i := range 4096 {
		holToEbcdicTable[i] = 0x100
		holToAsciiTable26[i] = 0xff
		holToAsciiTable29[i] = 0xff
	}

	// Initialize back translation from Hollerith to Ebcdic
	for i, t := range ebcdicToHolTable {
		if holToEbcdicTable[t] != 0x100 {
			s := fmt.Sprintf("Translation error EBCDIC %02x is %03x and %03x", i, t, holToEbcdicTable[t])
			panic(s)
		} else {
			holToEbcdicTable[t] = uint16(i)
		}
	}

	// Initialize back translation from Hollerith to ascii
	for i, t := range asciiToHol29 {
		if (t & 0xf000) == 0 {
			holToAsciiTable29[t] = uint8(i)
		}
	}
	for i, t := range asciiToHol26 {
		if (t & 0xf000) == 0 {
			holToAsciiTable26[t] = uint8(i)
		}
	}
}

func (ctx CardContext) HopperSize() int {
	return ctx.hopperCards - ctx.hopperPos
}

func (ctx CardContext) StackSize() int {
	return ctx.hopperCards
}

func (ctx *CardContext) CardEOF() bool {
	if ctx.hopperCards == 0 || ctx.hopperPos >= ctx.hopperCards {
		return true
	}
	c := ctx.deck[ctx.hopperPos]
	return (c.Image[0] & flag_eof) != 0
}

func (ctx *CardContext) FileName() string {
	if ctx.file == nil {
		return ""
	}
	return ctx.file.Name()
}

// Set end of file flag on last card in deck
func (ctx *CardContext) SetEOF() {
	if ctx.hopperCards != 0 {
		ctx.deck[ctx.hopperCards-1].Image[0] |= flag_eof
	}
}

func (ctx *CardContext) ReadCard() (Card, int) {

	if ctx.eofPending {
		ctx.eofPending = false
		return emptyCard, CARD_EOF
	}
	if ctx.hopperPos >= ctx.hopperCards {
		return emptyCard, CARD_EMPTY
	}
	c := ctx.deck[ctx.hopperPos]
	ctx.hopperPos++

	if (c.Image[0] & flag_eof) != 0 {
		if (c.Image[0] & flag_dat) != 0 {
			ctx.eofPending = true
			c.Image[0] &= 07777
			return c, CARD_OK
		}
		return emptyCard, CARD_EOF
	}
	if (c.Image[0] & flag_err) != 0 {
		return emptyCard, CARD_ERROR
	}
	c.Image[0] &= 07777
	return c, CARD_OK
}

// Helper function to look for special cards
func cmpCard(buf *cardBuffer, s string) bool {
	if buf.buffer[0] != '~' {
		return false
	}

	word := []byte(buf.buffer[1 : len(s)+1])
	for i, v := range s {
		if bytes.ToUpper(word)[i] != byte(v) {
			return false
		}
	}
	return true
}

// Card punch routine

//	Modifiers have been checked by the caller
//	C modifier is recognized (column binary is implemented)
//
// Convert word record into column image
// Check output type, if auto or text, try and convert record to bcd first
// If failed and text report error and dump what we have
// Else if binary or not convertable, dump as image
func (ctx *CardContext) PunchCard(img Card) int {
	mode := ctx.mode

	// Fix mode if in auto mode
	if mode == MODE_AUTO {
		mode = MODE_TEXT
		ok := true
		// Try to convert each column to ascii
		for i := range 80 {
			var ch uint8
			c := img.Image[i]
			switch ctx.table {
			case TYPE_029, TYPE_ASCII:
				ch = holToAsciiTable29[c]
			case TYPE_026:
				ch = holToAsciiTable26[c]
			}
			if ch == 0xff {
				ok = false
				break
			}
		}
		if !ok {
			mode = MODE_OCTAL
		}
	}

	outBuffer := make([]byte, 0, 330)
	out := outBuffer[0:0]
	switch mode {
	default:
		fallthrough
	case MODE_TEXT:
		// Scan each column
		// Try to convert each column to ascii
		for i := range 80 {
			var ch uint8
			c := img.Image[i]
			switch ctx.table {
			case TYPE_029, TYPE_ASCII:
				ch = holToAsciiTable29[c]
			case TYPE_026:
				ch = holToAsciiTable26[c]
			}
			if ch == 0xff {
				ch = '?'
			}
			out = out[0 : len(out)+1]
			out[i] = ch
		}
		var l int
		for l = 79; l >= 0; l-- {
			if out[l] != ' ' {
				break
			}
		}
		out = out[0 : l+2]
		out[l+1] = '\n'
	case MODE_OCTAL:
		out = out[0:321]

		var i int
		for i = 79; i >= 0; i-- {
			if img.Image[i] != 0 {
				i++
				break
			}
		}
		// Check if special card
		out[0] = '~'
		o := 4
		if i == 0 {
			out[1] = 'e'
			out[2] = 'o'
			if img.Image[0] == 07 {
				out[3] = 'r'
				goto fin
			}
			if img.Image[0] == 015 {
				out[3] = 'f'
				goto fin
			}
			if img.Image[0] == 017 {
				out[3] = 'i'
				goto fin
			}
		}
		out[1] = 'r'
		out[2] = 'a'
		out[3] = 'w'

		for p := range i {
			out[o] = byte((img.Image[p]>>9)&7) + '0'
			out[o+1] = byte((img.Image[p]>>6)&7) + '0'
			out[o+2] = byte((img.Image[p]>>3)&7) + '0'
			out[o+3] = byte((img.Image[p]>>0)&7) + '0'
			o += 4
		}
	fin:
		out[o] = '\n'
		o++
		out = out[0:o]
	case MODE_BIN:
		out = out[0:160]
		for i := range 80 {
			out[i*2] = byte((img.Image[i] & 0x00f) << 4)
			out[(i*2)+1] = byte((img.Image[i] & 0xff0) >> 4)
		}
	case MODE_CBN:
		out = out[0:160]
		for i := range 80 {
			out[i*2] = byte(img.Image[i]>>6) & 077
			out[(i*2)+1] = byte(img.Image[i] & 077)
		}
		// Set parity
		for i := range 160 {
			out[i] |= 0100 ^ xlat.ParityTable[out[i]]
		}
		out[0] |= 0200
	case MODE_BCD:
		out = out[0:80]
		for i := range 80 {
			out[i] = HolToBcd(img.Image[i])
			if out[i] != 0x7f {
				out[i] |= xlat.ParityTable[out[i]]
			} else {
				out[i] = 077
			}
		}
		out[0] |= 0200
	case MODE_EBCDIC:
		out = out[0:80]
		for i := range 80 {
			out[i] = byte(HolToEbcdic(img.Image[i]))
		}
	}
	ctx.hopperPos++
	_, _ = ctx.file.Write(out)
	return CARD_OK
}

// Empty hopper of cards
func (ctx *CardContext) EmptyDeck() {
	ctx.deck = ctx.deck[0:0]
	ctx.hopperPos = 0
	ctx.hopperCards = 0
}

func (ctx *CardContext) BlankDeck(n int) {
	var c Card
	for i := range 80 {
		c.Image[i] = 0
	}
	for range n {
		ctx.deck = append(ctx.deck, c)
		ctx.hopperCards++
	}
}

func (ctx *CardContext) SetTable(table int) {
	ctx.table = table
}

func NewCardContext(mode int) *CardContext {
	ctx := new(CardContext)
	ctx.mode = mode
	ctx.table = TYPE_029
	return ctx
}

// Read file into hopper
func (ctx *CardContext) readDeck(fileName string) error {
	var buffer cardBuffer
	var eof bool = false

	buffer.len = 0
	buffer.buffer[0] = 0 // Initialize buffer to empty
	ctx.file = nil
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	ctx.file = file
	ctx.deck = ctx.deck[ctx.hopperPos:ctx.hopperCards]
	ctx.hopperPos = 0
	ctx.hopperCards = len(ctx.deck)

	// Slurp up file
	for {
		if buffer.len < 500 && !eof {
			var n int
			n, err = ctx.file.Read(buffer.buffer[buffer.len:len(buffer.buffer)])
			if err != nil {
				if err == io.EOF {
					err = nil
					eof = true
				} else {
					break
				}
			} else {
				buffer.len += n
			}
		}
		ctx.parseCard(&buffer)
		if buffer.len == 0 {
			break
		}
	}
	ctx.file.Close()
	ctx.file = nil
	return err
}

// Convert head of buffer to a card image.
func (ctx *CardContext) parseCard(buf *cardBuffer) {
	var ch byte
	var mode int = MODE_TEXT
	c := Card{}
	var p int = 0
	var col int = 0
	if ctx.mode == MODE_AUTO {
		// Check to see if binary card in it
		var t uint16 = 0
		var i int
		for i = 0; i < 160 && i < buf.len; i += 2 {
			t |= uint16(buf.buffer[i] & 0xff)
		}
		// Check if every other char < 16 & full buffer
		if (t&0xf) == 0 && i == 160 {
			mode = MODE_BIN // Possibly binary
		}
		// Check if maybe BCD or CBN
		if (buf.buffer[0] & 0x80) != 0 {
			var odd, even int
			// Check all chars for correct parity
			for i = 0; i < buf.len && i < 160; i++ {
				ch = buf.buffer[i] & 0177
				// Try matching parity
				if xlat.ParityTable[ch&077] == (ch & 0100) {
					even++
				}
				if xlat.ParityTable[ch&077] != (ch & 0100) {
					odd++
				}
				// Check if we hit end of record.
				if even == 0 && odd == 160 {
					mode = MODE_CBN
					break
				}
				if odd == 0 && even == 80 {
					mode = MODE_BCD
					break
				}
				if (buf.buffer[i+1] & 0x80) != 0 {
					break
				}
			}

		}
		if ctx.mode != MODE_AUTO && ctx.mode != mode {
			c.Image[0] = flag_err
			goto card_done
		}
	} else {
		mode = ctx.mode
	}

	// scan card.
	switch mode {
	case MODE_TEXT:
		// Check for special codes
		if buf.buffer[0] == '~' {
			var f bool = true
			col = 1
			for i := 1; col < 80 && f && i < buf.len; i++ {
				ch = buf.buffer[i]
				switch ch {
				case '\n', '\r':
					col = 80
				case ' ': // Ignore space
				case '\t':
					col = (col | 7) + 1
				default:
					f = false
				}
			}
			if f {
				c.Image[0] = flag_eof
				goto endCard
			}
			if cmpCard(buf, "RAW") {
				var j int = 0
				col = 0
				for i := 4; col < 80 && i < buf.len; i++ {
					ch = buf.buffer[i]
					if ch >= '0' && ch <= '7' {
						c.Image[col] = (c.Image[col] << 3) | (uint16(ch - '0'))
						j++
						p++
					} else if ch == '\n' || ch == '\r' {
						break
					} else {
						c.Image[0] = flag_err
					}
					if j == 4 {
						col++
						j = 0
					}
				}
			} else if cmpCard(buf, "EOR") {
				c.Image[0] = 07 // 7/8/9 punch
				p = 4
			} else if cmpCard(buf, "EOF") {
				c.Image[0] = 015 // 6/7/9 punch
				p = 4
			} else if cmpCard(buf, "EOI") {
				c.Image[0] = 017 // 6/7/8/9 punch
				p = 4

			}
			goto endCard
		}
		// Convert text line into card image
		for p = 0; col < 80 && p < buf.len; p++ {
			ch = buf.buffer[p]
			switch ch {
			case '\r': // ignore
			case '\t': // Skip to multiple of 8
				col = (col | 7) + 1
			case '\n':
				col = 80
				p--
			default:
				var t uint16

				switch ctx.table {
				case TYPE_029:
					t = asciiToHol29[ch]
				case TYPE_026:
					t = asciiToHol26[ch]
				case TYPE_ASCII:
					t = asciiToHolEbcdic[ch]
				}
				if (t & 0xf000) != 0 {
					t = 0xfff
				}
				c.Image[col] = t & 0xfff
				col++
			}
		}
	endCard:
		// Scan to end of line, ignore anything after last column
		for buf.buffer[p] != '\n' && buf.buffer[p] != '\r' && p < buf.len {
			p++
		}
		if buf.buffer[p] == '\r' {
			p++
		}
		if buf.buffer[p] == '\n' {
			p++
		}
	case MODE_BIN:
		t := uint16(0)
		if buf.len < 160 {
			c.Image[0] = flag_err
			goto card_done
		}
		// Move data to buffer
		for p = 0; p < 160; {
			t |= uint16(buf.buffer[p] & 0xff)
			c.Image[col] = uint16((buf.buffer[p] >> 4) & 0xf)
			c.Image[col] |= uint16(buf.buffer[p+1]&0xff) << 4
			col++
			p += 2
		}
		if (t & 0xf) != 0 {
			c.Image[0] = flag_err
		}
	case MODE_CBN:
		// Check if first character is a tape mark
		if buf.buffer[0] == 0217 && (buf.len == 1 || (buf.buffer[1]&0200) != 0) {
			p = 1
			c.Image[0] = flag_eof
			break
		}

		// Clear record mark
		buf.buffer[0] &= 0x7f

		// Convert card and check for errors
		col = 0
		error := false
		for p = 0; p < buf.len && col < 80; {
			ch = buf.buffer[p]
			if (ch & 0x80) != 0 {
				break
			}
			ch &= 077
			if xlat.ParityTable[ch] == (buf.buffer[p] & 0100) {
				error = true
			}
			c.Image[col] = uint16(ch) << 6
			p++
			ch = buf.buffer[p]
			if (ch & 0x80) != 0 {
				break
			}
			ch &= 077
			if xlat.ParityTable[ch] == (buf.buffer[p] & 0100) {
				error = true
			}
			c.Image[col] |= uint16(ch)
			col++
			p++
		}

		if ctx.mode != MODE_AUTO {
			if p < buf.len && col >= 80 && (buf.buffer[p]&0x80) == 0 {
				error = true
				for (buf.buffer[p] & 0x80) == 0 {
					if p > buf.len {
						break
					}
					p++
				}
			}
		}
		if error {
			c.Image[0] = flag_err
		}
	case MODE_BCD:
		// Check if first character is a tape mark
		if buf.buffer[0] == 0217 && (buf.len == 1 || (buf.buffer[1]&0200) != 0) {
			p = 1
			c.Image[0] = flag_eof
			break
		}

		buf.buffer[0] &= 0x7f
		// Convert text line into card image
		col = 0
		error := false
		for p = 0; col < 80 && p < buf.len; p++ {
			ch = buf.buffer[p]
			if (ch & 0x80) != 0 {
				break
			}
			ch &= 077
			if xlat.ParityTable[ch] != (buf.buffer[p] & 0100) {
				error = true
			}
			c.Image[col] = BcdToHol(ch)
			col++
		}

		// Record over length of card, skip until next
		if ctx.mode != MODE_AUTO {
			if p < buf.len && col >= 80 && (buf.buffer[p]&0x80) == 0 {
				error = true
				for (buf.buffer[p] & 0x80) != 0 {
					if p > buf.len {
						break
					}
					p++
				}
			}
		}

		// Clear out remainder of record if needed
		for col < 80 {
			c.Image[col] = 0
			col++
		}

		// Set error flage if something wrong
		if error {
			c.Image[0] = flag_err
		}
	case MODE_EBCDIC:
		// Move data to buffere
		for p = 0; p < 80 && p < buf.len; p++ {
			c.Image[p] = ebcdicToHolTable[buf.buffer[p]]
		}
		if buf.len < 80 {
			c.Image[0] |= 0xfff
		}
	}

card_done:
	if (c.Image[0] & (flag_eof | flag_err)) == 0 {
		c.Image[0] |= flag_dat
	}
	ctx.deck = append(ctx.deck, c)
	ctx.hopperCards++
	l := buf.len - p
	j := p
	for i := 0; i < l; i++ {
		buf.buffer[i] = buf.buffer[j]
		j++
	}
	buf.len -= p
}
