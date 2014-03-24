package main

import (
	"github.com/ParsePlatform/go.grace/gracehttp"
	"net/http"
	"fmt"
	"config"
	"os"
)

func StartServer() {

	for port, _ := range config.GetListen() {
		fmt.Printf("start listen : %d\n", port)
		go gracehttp.Serve( &http.Server{Addr: fmt.Sprintf("%s%d", ":", port), Handler: ServerHandler(port)})

	}
	
}

func ServerHandler(port int) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Routing %s%s\n", r.Host, r.URL)
		fmt.Fprintf(
			w,
			"started at  from pid %d.\n",
			os.Getpid(),
		)
	})
	return mux
}