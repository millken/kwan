package main

import (
	"config"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	QUEUE_LENGTH    = 65535
	CONFIG_INTERVAL = 20
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

	config.Read()
	startServer()
	time.Sleep(1e9)
}
