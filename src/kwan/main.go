package main

import (
	"config"
	"logger"
	"os"
	"os/signal"
	"runtime"
	"strings"
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

func main() {
	numCpus := runtime.NumCPU()
	THREADS = numCpus * 1
	runtime.GOMAXPROCS(numCpus)
	logger.Info("Started: %d cores, %d threads， version: %s", numCpus, THREADS, VERSION)

	config.Read()
	startServer()
	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)

	<-terminate
	logger.Info("signal received, stopping")
}
