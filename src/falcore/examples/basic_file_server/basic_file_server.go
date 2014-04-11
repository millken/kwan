package main

import (
	"flag"
	"fmt"
	"github.com/fitstar/falcore"
	"github.com/fitstar/falcore/filter"
)

// Command line options
var (
	port = flag.Int("port", 8000, "the port to listen on")
	path = flag.String("base", "./test", "the path to serve files from")
)

func main() {
	// parse command line options
	flag.Parse()

	// setup pipeline
	pipeline := falcore.NewPipeline()

	// upstream filters

	// Serve files
	pipeline.Upstream.PushBack(&filter.FileFilter{
		BasePath:       *path,
		DirectoryIndex: "index.html", // Serve index.html for root requests
	})

	// downstream
	pipeline.Downstream.PushBack(filter.NewCompressionFilter(nil))

	// setup server
	server := falcore.NewServer(*port, pipeline)

	// start the server
	// this is normally blocking forever unless you send lifecycle commands
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Could not start server:", err)
	}
}
