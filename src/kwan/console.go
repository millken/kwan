//http://blog.brandonc.me/2013/04/go-program-profiling.html
package main

import (
	"bufio"
	"cache"
	"config"
	"fmt"
	"logger"
	"net"
	"net/textproto"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"
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
			case "load":
				usage1 := &syscall.Rusage{}
				var lastUtime int64
				var lastStime int64
				counter := 0
				for {
					//http://man7.org/linux/man-pages/man3/vtimes.3.html
					syscall.Getrusage(syscall.RUSAGE_SELF, usage1)

					utime := usage1.Utime.Nano()
					stime := usage1.Stime.Nano()
					userCPUUtil := float64(utime-lastUtime) * 100 / float64(1)
					sysCPUUtil := float64(stime-lastStime) * 100 / float64(1)
					memUtil := usage1.Maxrss * 1024

					lastUtime = utime
					lastStime = stime
					if counter > 0 {
						conn.Write([]byte(fmt.Sprintf("cpu: %3.2f%% us  %3.2f%% sy, mem:%s \n", userCPUUtil, sysCPUUtil, toH(uint64(memUtil)))))
						break
					}
					counter += 1

					time.Sleep(1)
				}
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
		case "keys":
			res, err := cache.Do("keys", tokens[1])
			if err != nil {
				output := err.Error()
				conn.Write([]byte(output + "\n"))
			} else {
				for k, v := range res.(map[string]interface{}) {
					switch v.(type) {
					case int:
						conn.Write([]byte(fmt.Sprintf("%s\t%d\n", k, v.(int))))
					case string:
						conn.Write([]byte(k + "\t" + v.(string) + "\n"))

					}

				}
			}

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

// human readable format
func toH(bytes uint64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.2fK", float64(bytes)/1024)
	case bytes < 1024*1024*1024:
		return fmt.Sprintf("%.2fM", float64(bytes)/1024/1024)
	default:
		return fmt.Sprintf("%.2fG", float64(bytes)/1024/1024/1024)
	}
}
