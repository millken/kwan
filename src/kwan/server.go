package main

import (
	"boot"
	"config"
	"fmt"
	"net/http"
	"strings"
)

type Source struct {
	Ip     string
	Port   int
	UseSsl bool
}

func startServer() {
	server := boot.NewApp()
	for port, _ := range config.GetListen() {
		fmt.Printf("start listen : %d\n", port)
		server.AddServer(&http.Server{Addr: fmt.Sprintf("%s%d", ":", port), Handler: serverHandler(port)})
	}
	if err := server.Run(); err != nil {
		fmt.Printf("server run error : %s", err.Error())
	}
}

func serverHandler(port int) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//fmt.Printf("Routing %s%s\n", r.Host, r.URL)
		domain := strings.Split(r.Host, ":")[0]
		vhost, found := config.MatchingVhost("0.0.0.0", port, domain)
		if found {
			src, _ := getSourceIP(domain, port, vhost)
			proxy := boot.NewProxy()
			proxy.AddBackend(src.Ip, 80)
			proxy.SetVhost(vhost)
			proxy.ServeHTTP(w, r)
		}

	})
	return mux
}

func getSourceIP(domain string, port int, vhost config.Vhost) (src Source, err error) {
	domains := []string{
		domain,
		config.WildcardOf(domain),
	}
	for _, dom := range domains {
		for _, host := range vhost.Host {
			if host.Domain == dom {
				return Source{host.Ip, host.Port, false}, nil
			}
		}
	}
	return
}
