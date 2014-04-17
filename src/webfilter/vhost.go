package webfilter

import (
	"github.com/fitstar/falcore"
	"github.com/fitstar/falcore/filter"
	"config"
	"net/http"
	//"fmt"
)
type VhostFilter struct {
}
 
func (vh *VhostFilter) FilterRequest(req *falcore.Request) (res *http.Response) {
	host, _ := filter.SplitHostPort(req.HttpRequest.Host, 80)
	dHost, dPort := filter.SplitHostPort(req.ServerAddr, 80)
	//fmt.Printf("Routing %s%s : %s => %s\n", req.HttpRequest.Host, req.HttpRequest.URL, req.RemoteAddr, req.ServerAddr)

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
		domain,
		config.WildcardOf(domain),
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
