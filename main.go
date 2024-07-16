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
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	getopt "github.com/pborman/getopt/v2"
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
	//	optDeck := getopt.StringLong("deck", 'd', "", "Deck to load")
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
	Logger = slog.New(logger.NewHandler(file, &slog.HandlerOptions{Level: programLevel, AddSource: false}))
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

	// Wait for a SIGINT or SIGTERM signal to gracefully shut down the server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	msg := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		// Receive commands from stdin
		for {
			input, _ := reader.ReadString('\n')
			msg <- input
		}
	}()

loop:
	for {
		select {
		case <-sigChan:
			fmt.Println("Got quit signal")
			break loop
		case <-msg:
			fmt.Printf("IPL device: %03x\n", core.IPLDevice())
			masterChannel <- master.Packet{DevNum: core.IPLDevice(), Msg: master.IPLdevice}
		}
	}

	Logger.Info("Shutting down CPU")
	cpu.Stop()
	Logger.Info("Shutting down server...")
	telnet.Stop()
	Logger.Info("Servers stopped.")
}
