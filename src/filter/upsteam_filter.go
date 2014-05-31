package filter

import (
	"config"
	"core"
	"net/http"
	"time"
)

type UpstreamFilter int

func (up UpstreamFilter) FilterRequest(request *core.Request) (res *http.Response) {
	vhost := request.Context["config"].(*config.Vhost)
	req := request.HttpRequest
	sHost, sPort := "", 80
	_, dPort := SplitHostPort(request.ServerAddr, 80)
	host, port := SplitHostPort(req.Host, dPort)
	sHost, sPort = GetSourceIP(host, port, vhost)
	timeout := time.Duration(30) * time.Second
	if vhost.Limit.Timeout > 0 {
		timeout = time.Duration(vhost.Limit.Timeout) * time.Second
	}
	proxyFilter := NewUpstream(NewUpstreamTransport(sHost, sPort, timeout, nil))
	res = proxyFilter.FilterRequest(request)
	return res
}
