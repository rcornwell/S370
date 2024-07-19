/*
 * S370 - Command reader.
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

package reader

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/peterh/liner"
	"github.com/rcornwell/S370/command/parser"
	"github.com/rcornwell/S370/emu/core"
)

func ConsoleReader(core *core.Core) {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)
	line.SetCompleter(func(line string) []string {
		return parser.CompleteCmd(line)
	})

	for {
		command, err := line.Prompt("S370> ")
		if err == nil {
			line.AppendHistory(command)
			quit, err := parser.ProcessCommand(command, core)
			if err != nil {
				fmt.Println("Error: " + err.Error())
			}
			if quit {
				return
			}
			continue
		}

		if errors.Is(err, liner.ErrPromptAborted) {
			return
		} else {
			slog.Error("error reading line: " + err.Error())
		}
	}
}
