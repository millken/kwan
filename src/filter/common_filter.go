package filter

import (
	"config"
	"core"
	"logger"
	"net"
	"net/http"
	"store"
	"strconv"
	"strings"
	"utils"
	"time"
	"fmt"
)

type StatusFilter int

type CommonLogger map[string]store.Log

func NewCommonLogger() (df CommonLogger) {
	df = make(CommonLogger)
	return
}

func (s StatusFilter) FilterRequest(request *core.Request) *http.Response {
	req := request.HttpRequest
	vhost := request.Context["config"].(*config.Vhost)
	if vhost.Status == 1 {
		return core.StringResponse(request.HttpRequest, 200, nil, "the site was paused!\n")
	}
	RemoteAddr := request.RemoteAddr.String()
	ip, _, _ := net.SplitHostPort(RemoteAddr)

	//request filter
	//http2https
	for _, host := range vhost.Request.Http2https {
		if req.URL.Scheme == "http" && req.Host == host {
			req.URL.Scheme = "https"
			req.URL.Host = req.Host
			return core.RedirectResponse(req, req.URL.String())
		}
	}
	request.Context["blacklist"] = false
	//blacklist
BLACKLIST:
	for {
		for _, ips := range vhost.BlackList.Ip {
			if checkIp(ip, ips) {
				request.Context["blacklist"] = true
				break BLACKLIST
			}
		}
		for _, urls := range vhost.BlackList.Url {
			if checkUrl(req.URL.RequestURI(), urls) {
				request.Context["blacklist"] = true
				break BLACKLIST
			}
		}
		for _, useragent := range vhost.BlackList.UserAgent {
			if strings.Contains(req.UserAgent(), useragent) {
				request.Context["blacklist"] = true
				break BLACKLIST
			}
		}
		break BLACKLIST
	}
	//whitelist
	request.Context["whitelist"] = false
WHITELIST:
	for {
		for _, ips := range vhost.WhiteList.Ip {
			if checkIp(ip, ips) {
				request.Context["whitelist"] = true
				break WHITELIST
			}
		}
		for _, urls := range vhost.WhiteList.Url {
			if checkUrl(req.URL.RequestURI(), urls) {
				request.Context["whitelist"] = true
				break WHITELIST
			}
		}
		for _, useragent := range vhost.WhiteList.UserAgent {
			if strings.Contains(req.UserAgent(), useragent) {
				request.Context["whitelist"] = true
				break WHITELIST
			}
		}
		break WHITELIST
	}
	if request.Context["whitelist"] == true {
		request.Context["blacklist"] = false
	} else if request.Context["blacklist"] == true {
		return core.StringResponse(request.HttpRequest, 403, nil, "you has been blocked!\n")
	}
	return nil
}

func (c CommonLogger) FilterResponse(request *core.Request, res *http.Response) {
	//logger.Finest("CommonLogger start")
	res.Header.Set("Server", config.GetServername())
	go func() {
		req := request.HttpRequest
		req.RemoteAddr = request.RemoteAddr.String()
		vhost := request.Context["config"].(*config.Vhost)
		vhostname := vhost.Name
		vl := vhost.Log
		if vl.Status == true {
			switch vl.Type {
			case "tcp", "udp":
				if _, ok := c[vhostname]; !ok {

					c[vhostname] = store.NewSocketHandler(vl.Type, vl.Addr)
				}
			case "file":
				if _, ok := c[vhostname]; !ok {

					nclw := store.NewCommonLogWriter(vl.Addr, vl.RotateDaily)
					if nclw != nil {
					nclw.SetRotateDaily(vl.RotateDaily)
					c[vhostname] = nclw
					}
				}
			}
			if c[vhostname] != nil {
			err := c[vhostname].Write(c.buildCommonLogLine(request, res))
			if err != nil {
				logger.Warn(err)
			}
		}
		}
	}()

}

func (c CommonLogger) buildCommonLogLine(request *core.Request, res *http.Response) string {
	req := request.HttpRequest
	username := "-"
	if req.URL.User != nil {
		if name := req.URL.User.Username(); name != "" {
			username = name
		}
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		host = req.RemoteAddr
	}

	ts := time.Now()
	return fmt.Sprintf("%s - %s [%s] \"%s %s %s\" %d %d %s \"%s\" \"%s\"\n",
		host,
		username,
		ts.Format("02/Jan/2006:15:04:05 -0700"),
		req.Method,
		req.URL.RequestURI(),
		req.Proto,
		res.StatusCode,
		res.ContentLength,
		req.Host,
		req.Referer(),
		req.UserAgent(),
	)
}
func checkIp(ip, ips string) bool {
	ipint32 := utils.IpStringToI32(ip)
	ips = strings.Trim(ips, "\r\n ")
	if ips == "" {
		return false
	}
	if strings.Index(ips, "/") != -1 {
		if _, _, err := net.ParseCIDR(ips); err == nil {
			cidr := strings.Split(ip, "/")
			addr32 := utils.IpStringToI32(cidr[0])
			mask32, _ := strconv.ParseUint(cidr[1], 10, 8)
			ip_start := addr32 & (0xFFFFFFFF << (32 - mask32))
			ip_end := addr32 | ^(0xFFFFFFFF << (32 - mask32))
			if ipint32 >= ip_start && ipint32 <= ip_end {
				return true
			}
		}
	} else {
		if ip1 := net.ParseIP(ips); ip1 != nil {
			ipsint32 := utils.IpToI32(ip1)
			if ipint32 == ipsint32 {
				return true
			}
		}
	}
	return false
}

func checkUrl(url, urls string) bool {
	urls = strings.Trim(urls, "\r\n ")
	if urls == "" {
		return false
	}
	return strings.Contains(url, urls)
}
