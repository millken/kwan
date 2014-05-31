package filter

import (
	"config"
	"core"
	"fmt"
	"logger"
	"net/http"
	"strconv"
	"strings"
)

type VhostRouter struct {
	Scheme string
}

func (vr *VhostRouter) FilterRequest(request *core.Request) *http.Response {
	request.HttpRequest.URL.Scheme = vr.Scheme
	request.HttpRequest.URL.Host = request.HttpRequest.Host
	host, _ := SplitHostPort(request.HttpRequest.Host, 80)
	dHost, dPort := SplitHostPort(request.ServerAddr, 80)
	logger.Finest("Routing %s: %s => %s", request.HttpRequest.URL,
		request.RemoteAddr, request.ServerAddr)

	vhost, found := config.MatchingVhost(dHost, dPort, host)
	if found {
		request.Context["config"] = &vhost
	} else {
		request.Context["config"] = &config.Vhost{}
		//可以直接返回不存在
		return core.StringResponse(request.HttpRequest, 404, nil, "<h1>Not found</h1>\n")

	}

	return nil
}

func GetSourceIP(domain string, port int, vhost *config.Vhost) (sHost string, sPort int) {
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
				} else {
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
			fmt.Printf("Error converting port to int for %s : %s", upstreamHost, err)
		}
	}
	return upstreamHost, upstreamPort
}
