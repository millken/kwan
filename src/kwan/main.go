package main

import (
	"config"
	"logger"
	"os"
	"os/signal"
	"runtime"
	"strings"
)

var VERSION string = "0.11"
var gitVersion string
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
	threads := numCpus * 1
	runtime.GOMAXPROCS(numCpus)
	logger.Global = logger.NewDefaultLogger(logger.FINEST)
	logger.Info("Started: %d cores, %d threads， version: %s", numCpus, threads, VERSION)

	setRlimit()
	config.Read()
	go filterManager()
	listenServer()
	startConsole()
	sigChan := make(chan os.Signal, 3)

	signal.Notify(sigChan, os.Interrupt, os.Kill)

	<-sigChan
}
