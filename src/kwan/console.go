//http://blog.brandonc.me/2013/04/go-program-profiling.html
package main

import (
	"bufio"
	"config"
	"fmt"
	"logger"
	"net"
	"net/textproto"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"runtime"
)

var cpuProfile *os.File

func StartConsole() {
	address := config.GetConsoleAddr()
	if address == "" {
		address = "0.0.0.0:59100"
	}
	logger.Info("start console server: %s", address)

	listener, err := net.Listen("tcp", address)

	if err != nil {
		logger.Warn("Error creating TCP socket : %s", address, err.Error())
	}

	// Safe to close the listener after error checking
	defer listener.Close()

	// Loop, accept, push work into a goroutine
	for {
		conn, err := listener.Accept()

		if err != nil {
			logger.Warn("Connection error from accept : %s", err.Error())
		}

		// TODO: Use a pool of Goroutines
		go ConsoleRawHandler(conn)
	}
}

func ConsoleRawHandler(conn net.Conn) {

	reader := bufio.NewReader(conn)
	// TODO extract this into an ASCIIProtocolHandler
	protocol := textproto.NewReader(reader)

	defer conn.Close()

	// Loop and read, parsing for commands along the way
	for {

		line, err := protocol.ReadLine()

		if err != nil {
			logger.Warn("Error reading from client", err.Error())
			return
		}

		logger.Debug("got line: %s", line)

		tokens := strings.Split(line, " ")
		command := tokens[0]

		switch command {
			//http://1234n.com/?post/wgskfs 
		case "lookup":
			subcommand := tokens[1]
			switch subcommand {
			case "heap":
				p := pprof.Lookup("heap")
				p.WriteTo(conn, 2)
			case "threadcreate":
				p := pprof.Lookup("threadcreate")
				p.WriteTo(conn, 2)
			case "block":
				p := pprof.Lookup("block")
				p.WriteTo(conn, 2)
			}

		case "startcpuprof":
			if cpuProfile == nil {
				filename := "cpu-" + strconv.Itoa(os.Getpid()) + ".pprof"
				if f, err := os.Create(filename); err != nil {
					conn.Write([]byte(fmt.Sprintf("start cpu profile failed: %v", err)))
				} else {
					conn.Write([]byte("start cpu profile\n"))
					pprof.StartCPUProfile(f)
					cpuProfile = f
				}
			}

		case "stopcpuprof":
			if cpuProfile != nil {
				pprof.StopCPUProfile()
				cpuProfile.Close()
				cpuProfile = nil
				conn.Write([]byte("stop cpu profile\n"))
			}
		case "getmemprof":
			filename := "mem-" + strconv.Itoa(os.Getpid()) + ".pprof"
           if f, err := os.Create(filename); err != nil {
               conn.Write([]byte(fmt.Sprintf("record memory profile failed: %v", err)))
           } else {
               runtime.GC()
               pprof.WriteHeapProfile(f)
               f.Close()
               conn.Write([]byte("record memory profile\n"))
           }
        case "quit":
        	return
        case "get":

			conn.Write([]byte("END\r\n"))
		case "set":
			if len(tokens) != 4 {
				conn.Write([]byte("Error"))
				return
			}
			conn.Write([]byte("set\r\n"))        	
		}


	}
}
