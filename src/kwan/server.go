package main

import (
	"config"
	"logger"
	"github.com/millken/falcore"
	"webfilter"
)

func startServer() {
	for addr, bindnum := range config.GetListen() {
		logger.Info("start listen[%d] : %s", bindnum, addr)
		go listen(addr)
	}
	for addr, ssls := range config.GetSslListen() {
		logger.Info("start ssl listen : %s", addr)
		certs := make([]falcore.Certificates, 0)
		for _, ssl := range ssls {
			certs = append(certs, falcore.Certificates{
					CertFile: ssl.Certfile,
					KeyFile:  ssl.Keyfile,
				})
		}
		go ssllisten(addr, certs)
	}	
}

func listen(addr string) {
	pipeline := makepipeline("http")
	server := falcore.NewServer(addr, pipeline)
	if err := server.ListenAndServe(); err != nil {
		logger.Error("Could not start server: %s", err)
	}
}

func ssllisten(addr string, certs []falcore.Certificates) {
	spipeline := makepipeline("https")
	server := falcore.NewServer(addr, spipeline)

	if err := server.ListenAndServeTLSSNI(certs); err != nil {
		logger.Error("Could not start server: %s", err)
	}
}

func makepipeline(scheme string) *falcore.Pipeline {
	pipeline := falcore.NewPipeline()

	// upstream
	//pipeline.Upstream.PushBack(helloFilter)
	pipeline.Upstream.PushBack(&webfilter.VhostRouter{scheme})
	
	var statusfilter webfilter.StatusFilter
	pipeline.Upstream.PushBack(statusfilter)
	pipeline.Downstream.PushBack(webfilter.NewCommonLogger())

	ddosfilter := webfilter.NewDdosFilter()
	pipeline.Upstream.PushBack(ddosfilter)

	cachefilter := webfilter.NewCacheFilter()
	pipeline.Upstream.PushBack(cachefilter)
	//server.CompletionCallback = reqCB	
	return pipeline
}