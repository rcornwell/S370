/*
 * S370 - Main process.
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

package main

import (
	"log/slog"
	"os"

	getopt "github.com/pborman/getopt/v2"
	reader "github.com/rcornwell/S370/command/reader"
	config "github.com/rcornwell/S370/config/configparser"
	core "github.com/rcornwell/S370/emu/core"
	master "github.com/rcornwell/S370/emu/master"
	syschannel "github.com/rcornwell/S370/emu/sys_channel"
	telnet "github.com/rcornwell/S370/telnet"
	logger "github.com/rcornwell/S370/util/logger"

	_ "github.com/rcornwell/S370/config/debugconfig"
	_ "github.com/rcornwell/S370/emu/models"
)

var Logger *slog.Logger

func main() {
	optConfig := getopt.StringLong("config", 'c', "S370.cfg", "Configuration file")
	optLogFile := getopt.StringLong("log", 'l', "", "Log file")
	optDebug := getopt.BoolLong("debug", 'd', "Log debug to console")
	optHelp := getopt.BoolLong("help", 'h', "Help")
	getopt.Parse()

	if *optHelp {
		getopt.Usage()
		os.Exit(0)
	}

	var file *os.File
	if optLogFile != nil {
		file, _ = os.Create(*optLogFile)
	}
	programLevel := new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)
	Logger := slog.New(logger.NewHandler(file, &slog.HandlerOptions{Level: programLevel, AddSource: false}, optDebug))
	slog.SetDefault(Logger)

	Logger.Info("S370 Started")
	if optConfig == nil {
		Logger.Error("Please specify a configuration file")
		os.Exit(0)
	}

	_, err := os.Stat(*optConfig)
	if os.IsNotExist(err) {
		Logger.Error("Configuration file ", *optConfig, " can't be found")
		os.Exit(0)
	}

	syschannel.InitializeChannels()
	err = config.LoadConfigFile(*optConfig)
	if err != nil {
		Logger.Error(err.Error())
		os.Exit(0)
	}

	masterChannel := make(chan master.Packet)

	// Create new routine to run CPU.
	cpu := core.NewCPU(masterChannel)

	// Configure I/O devices.
	syschannel.ResetChannels()

	// Start telnet servers.
	err = telnet.Start(masterChannel)
	if err != nil {
		Logger.Error(err.Error())
		os.Exit(1)
	}

	// Start main emulator.
	go cpu.Start()

	msg := make(chan string, 1)
	go func() {
		reader.ConsoleReader(cpu)
		msg <- ""
	}()

	// Wait on shutdown option
	<-msg

	cpu.Stop()
	telnet.Stop()
	Logger.Info("Servers stopped.")
}
