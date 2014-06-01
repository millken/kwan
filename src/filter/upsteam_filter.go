package filter

import (
	"config"
	"core"
	"logger"
	"net/http"
	"time"
)

type UpstreamFilter map[string]*UpstreamPool

func NewUpstreamFilter() (f UpstreamFilter) {
	f = make(UpstreamFilter)
	return
}

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
	isUsePool := false
	if ups != "" {
		if _, ok := up[ups]; !ok {
			var upstreams []*UpstreamEntry
			upsgroup := config.GetUpstream()
			if _, ok = upsgroup[ups]; ok {
				for _, uv := range upsgroup[ups].Host {
					if uv.Port == 0 {
						sPort = port
					} else {
						sPort = uv.Port
					}
					upstreams = append(upstreams, &UpstreamEntry{
						NewUpstream(NewUpstreamTransport(uv.Ip, sPort, timeout, nil)),
						uv.Weight,
					})
				}
				up[ups] = NewUpstreamPool(host, upstreams)
				isUsePool = true
			}
		} else {
			isUsePool = true
		}
	}
	if isUsePool == true { //loadblanence
		logger.Info("use UpstreamPool")
		res = up[ups].FilterRequest(request)
	} else {
		proxyFilter := NewUpstream(NewUpstreamTransport(sHost, sPort, timeout, nil))
		res = proxyFilter.FilterRequest(request)
	}
	return res
}
