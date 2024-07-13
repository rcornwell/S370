/*
 * S370 - telnet server, listener.
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

package telnet

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rcornwell/S370/emu/master"
)

type Server struct {
	wg         sync.WaitGroup
	listener   net.Listener
	shutdown   chan struct{}
	connection chan net.Conn
	master     chan master.Packet
}

// Open new listener.
func newServer(address string) (*Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on address %s: %w", address, err)
	}

	return &Server{
		listener:   listener,
		shutdown:   make(chan struct{}),
		connection: make(chan net.Conn),
	}, nil
}

// Accept a connection.
func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdown:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}
			s.connection <- conn
		}
	}
}

// Start processing for a new connection.
func (s *Server) handleConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdown:
			return
		case conn := <-s.connection:
			host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
			if err != nil {
				panic(err)
			}
			if host == "::" {
				host = "localhost"
			}
			fmt.Printf("Connection from %s:%s\n", host, port)
			go handleClient(conn, s.master)
		}
	}
}

// Start a new server.
func Start(master chan master.Packet) error {
	for portNum, p := range ports {
		s, err := newServer(":" + portNum)
		if err != nil {
			return err
		}
		p.server = s
		host, port, err := net.SplitHostPort(s.listener.Addr().String())
		if err != nil {
			panic(err)
		}
		if host == "::" {
			host = "localhost"
		}
		fmt.Printf("Server started on %s:%s\n", host, port)

		s.wg.Add(2)
		s.master = master
		go s.acceptConnections()
		go s.handleConnections()
	}
	return nil
}

// Stop a running servers.
func Stop() {
	for portNum, p := range ports {

		fmt.Println("Shutdown port: ", portNum)
		s := p.server
		close(s.shutdown)
		s.listener.Close()

		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			fmt.Println("Timed out waiting for connections to finish.")
			return
		}
	}
}
