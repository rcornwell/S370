/*
 * S370 - Log debug data to a file
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

package debug

import (
	"fmt"
	"os"
	"strconv"

	config "github.com/rcornwell/S370/config/configparser"
)

var logFile *os.File

// Generic debug message.
func Debugf(module string, mask int, level int, format string, a ...interface{}) {
	if (mask & level) != 0 {
		fmt.Fprintf(logFile, module+": "+format+"\n", a...)
	}
}

// Device debug message.
func DebugDevf(devNum uint16, mask int, level int, format string, a ...interface{}) {
	if (mask & level) != 0 {
		dev := strconv.FormatUint(uint64(devNum), 16)
		fmt.Fprintf(logFile, dev+": "+format+"\n", a...)
	}
}

// Device debug message.
func DebugChanf(number int, mask int, level int, format string, a ...interface{}) {
	if (mask & level) != 0 {
		ch := strconv.FormatInt(int64(number), 10)
		fmt.Fprintf(logFile, "Channel "+ch+": "+format+"\n", a...)
	}
}

// register a device on initialize.
func init() {
	config.RegisterFile("DEBUGFILE", create)
}

// Create a card punch device.
func create(_ uint16, fileName string, _ []config.Option) error {
	if logFile != nil {
		return fmt.Errorf("Can't have more then one debug file, previous: %s", logFile.Name())
	}

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("unable to create debug file: %s", fileName)
	}

	logFile = file
	return nil
}
