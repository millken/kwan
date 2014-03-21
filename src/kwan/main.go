package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"config"
)

const (
	QUEUE_LENGTH      = 65535
	CONFIG_INTERVAL   = 20
	DEFAULT_USERAGENT = "abench tester"
	RC4_KEY           = "x6l4^j)2"
)
var VERSION string = "1.39"
var gitVersion string
var THREADS = 8
var _ENV map[string]string

func init() {
	if len(gitVersion) > 0 {
		VERSION = VERSION + "/" + gitVersion
	}

	getenvironment := func(data []string, getkeyval func(item string) (key, val string)) map[string]string {
		items := make(map[string]string)
		for _, item := range data {
			key, val := getkeyval(item)
			items[key] = val
		}
		return items
	}
	_ENV = getenvironment(os.Environ(), func(item string) (key, val string) {
		splits := strings.Split(item, "=")
		key = splits[0]
		val = strings.Join(splits[1:], "=")
		return
	})
}

/*
 *  线程不能开太多，官方建议在20以内，这里设为cpu*2
 */
func main() {
	numCpus := runtime.NumCPU()
	THREADS = numCpus * 1
	runtime.GOMAXPROCS(numCpus)
	fmt.Printf("Started: %d cores, %d threads， version: %s\n", numCpus, THREADS, VERSION)

	go config.Read()

	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)

	<-terminate
	fmt.Printf(" signal received, stopping\n")
}
