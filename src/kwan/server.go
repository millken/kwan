package main

import (
	"config"
	"core"
	"logger"
)

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
