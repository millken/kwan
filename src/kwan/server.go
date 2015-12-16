package main

import (
	"config"
	"core"
	"logger"
	"syscall"
	//"webfilter"
)

func setRlimit() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		logger.Warn("Unable to obtain rLimit", err)
	}
	if rLimit.Cur < rLimit.Max {
		rLimit.Max = 999999
		rLimit.Cur = 999999
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			logger.Warn("Unable to increase number of open files limit", err)
		}
	}
}

func listenServer() {
	for addr, bindnum := range config.GetListen() {
		logger.Info("start listen[%d] : %s", bindnum, addr)
		go listen(addr)
	}
	for addr, ssls := range config.GetSslListen() {
		logger.Info("start listen ssl : %s", addr)
		certs := make([]core.Certificates, 0)
		for _, ssl := range ssls {
			certs = append(certs, core.Certificates{
				CertFile: ssl.Certfile,
				KeyFile:  ssl.Keyfile,
			})
		}
		go listenSSL(addr, certs)
	}
}

func listen(addr string) {
	server := core.NewServer(addr, "http")
	if err := server.ListenAndServe(); err != nil {
		logger.Exitf("Could not start server[%s]: %s", addr, err)
	}
}

func listenSSL(addr string, certs []core.Certificates) {
	server := core.NewServer(addr, "https")

	if err := server.ListenAndServeTLSSNI(certs); err != nil {
		logger.Error("Could not start server: %s", err)
	}
}
