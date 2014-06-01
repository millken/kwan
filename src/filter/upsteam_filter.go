package filter

import (
	"config"
	"core"
	"net/http"
	"time"
	"strings"
	"strconv"
	"logger"
)

type UpstreamFilter int


func (up UpstreamFilter) FilterRequest(request *core.Request) (res *http.Response) {
	vhost := request.Context["config"].(*config.Vhost)
	req := request.HttpRequest
	sHost, sPort, ups := "", 80, ""
	_, dPort := SplitHostPort(request.ServerAddr, 80)
	host, port := SplitHostPort(req.Host, dPort)
	sHost, sPort, ups = GetSourceIP(host, port, vhost)
	timeout := time.Duration(3) * time.Second
	if vhost.Limit.Timeout > 0 {
		timeout = time.Duration(vhost.Limit.Timeout) * time.Second
	}
	if strings.Index(ups, ",") > 0 { //loadblanence
		var upstreams []*UpstreamEntry
		for _, u := range strings.Split(ups, ",") {
			parts := strings.Split(u, ":")
			if len(parts) == 3 {
			sHost = parts[0]
			if sPort, _ = strconv.Atoi(parts[1]); sPort == 0 {
				sPort = port
			}
			weight, err := strconv.Atoi(parts[2])
			if err != nil {
				weight = 5
			}
			logger.Debug("%s:%d:%d", sHost, sPort, weight)
			upstreams = append(upstreams, &UpstreamEntry{
				NewUpstream(NewUpstreamTransport(sHost, sPort, timeout, nil)),
				weight,
				})
		}
		}
		ups_pool  := NewUpstreamPool(host, upstreams)
		res = ups_pool.FilterRequest(request)
	}else{
		proxyFilter := NewUpstream(NewUpstreamTransport(sHost, sPort, timeout, nil))
		res = proxyFilter.FilterRequest(request)
	}
	return res
}
