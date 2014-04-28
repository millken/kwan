package webfilter

import (
	"github.com/millken/falcore"
	"strings"
	"strconv"
	"config"
	"net/http"
	"fmt"
)
type VhostFilter struct {
}
 
func (vh *VhostFilter) FilterRequest(req *falcore.Request) (res *http.Response) {
	host, _ := SplitHostPort(req.HttpRequest.Host, 80)
	dHost, dPort := SplitHostPort(req.ServerAddr, 80)
	//fmt.Printf("Routing %s::%s%s : %s => %s\n", req.HttpRequest.Host, req.HttpRequest.URL, req.RemoteAddr, req.ServerAddr)

	vhost, found := config.MatchingVhost(dHost, dPort, host)
	if found {
		//first try read cache
		cachefilter := NewCacheFilter()
		cachefilter.SetVhost(vhost)
		res = cachefilter.FilterRequest(req)
	}

	return 
}


func GetSourceIP(domain string, port int, vhost config.Vhost) (sHost string, sPort int) {
	sHost = ""
	sPort = 0
	domains := []string{
		config.WildcardOf(domain),
		domain,
	}
	for _, dom := range domains {
		for _, host := range vhost.Host {
			if host.Domain == dom {
				sHost = host.Ip
				if host.Port == 0 {
					sPort = port
				}else{
					sPort = host.Port
				}
			}
		}
	}
	return
}

// fixme: probably should use net.SplitHostPort
func SplitHostPort(hostPort string, defaultPort int) (string, int) {
	hostPort = strings.Replace(hostPort, "[::]", "0.0.0.0", -1)
	parts := strings.Split(hostPort, ":")
	upstreamHost := parts[0]
	upstreamPort := defaultPort
	if len(parts) > 1 {
		var err error
		upstreamPort, err = strconv.Atoi(parts[1])
		if err != nil {
			upstreamPort = defaultPort
			fmt.Printf("Error converting port to int for %s : %s",  upstreamHost, err)
		}
	}
	return upstreamHost, upstreamPort
}