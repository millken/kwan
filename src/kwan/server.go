package main

import (
	"config"
	"fmt"
	"net/http"
	"github.com/fitstar/falcore"
	"webfilter"
)


func startServer() {
	for addr, bindnum := range config.GetListen() {
		fmt.Printf("start listen[%d] : %s\n", bindnum, addr)
		go listen(addr)
	}
}

func listen(addr string) {
	pipeline := falcore.NewPipeline()

	// upstream
	//pipeline.Upstream.PushBack(helloFilter)
	pipeline.Upstream.PushBack(&webfilter.VhostFilter{})	
	server := falcore.NewServer(addr, pipeline)
	//server.CompletionCallback = reqCB
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Could not start server:", err)
	}
}

var helloFilter = falcore.NewRequestFilter(func(req *falcore.Request) *http.Response {
	return falcore.StringResponse(req.HttpRequest, 200, nil, "hello world!")
})
var reqCB = func(req *falcore.Request, res *http.Response) {
	req.Trace(res) // Prints detailed stats about the request to the log
}

