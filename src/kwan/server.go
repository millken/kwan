package main

import (
	"boot"
	"net/http"
	"fmt"
	"config"
	"os"
)

func StartServer() {
	server := boot.NewApp()
	for port, _ := range config.GetListen() {
		fmt.Printf("start listen : %d\n", port)
		server.AddServer(&http.Server{Addr: fmt.Sprintf("%s%d", ":", port), Handler: ServerHandler(port)})
	}
	if err := server.Run(); err != nil {
		fmt.Printf("server run error : %s", err.Error())
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