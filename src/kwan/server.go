package main

import (
	"config"
	"fmt"
	"github.com/millken/falcore"
	"webfilter"
)

func startServer() {
	for addr, bindnum := range config.GetListen() {
		fmt.Printf("start listen[%d] : %s\n", bindnum, addr)
		go listen(addr)
	}
	for addr, ssls := range config.GetSslListen() {
		fmt.Printf("start ssl listen : %s\n", addr)
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
		fmt.Println("Could not start server:", err)
	}
}

func ssllisten(addr string, certs []falcore.Certificates) {
	spipeline := makepipeline("https")
	server := falcore.NewServer(addr, spipeline)

	if err := server.ListenAndServeTLSSNI(certs); err != nil {
		fmt.Println("Could not start server:", err)
	}
}

func makepipeline(scheme string) *falcore.Pipeline {
	pipeline := falcore.NewPipeline()

	// upstream
	//pipeline.Upstream.PushBack(helloFilter)
	pipeline.Upstream.PushBack(&webfilter.VhostRouter{scheme})
	
	var statusfilter webfilter.StatusFilter
	pipeline.Upstream.PushBack(statusfilter)

	ddosfilter := webfilter.NewDdosFilter()
	pipeline.Upstream.PushBack(ddosfilter)

	cachefilter := webfilter.NewCacheFilter()
	pipeline.Upstream.PushBack(cachefilter)
	//server.CompletionCallback = reqCB	
	return pipeline
}