package webfilter

import (
	"config"
	"github.com/millken/falcore"
	"net"
	"net/http"
	"strconv"
	"strings"
	"utils"
	"logger"
	"time"
	"fmt"
)

type StatusFilter int

func (s StatusFilter) FilterRequest(request *falcore.Request) *http.Response {
	req := request.HttpRequest
	vhost := request.Context["config"].(config.Vhost)
	if vhost.Status == 1 {
		return falcore.StringResponse(request.HttpRequest, 200, nil, "the site was paused!\n")
	}
	RemoteAddr := request.RemoteAddr.String()
	ip, _, _ := net.SplitHostPort(RemoteAddr)
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
	}else if request.Context["blacklist"] == true {
		return falcore.StringResponse(request.HttpRequest, 403, nil, "you has been blocked!\n")
	}
	return nil
}

func (s StatusFilter) FilterResponse(request *falcore.Request, res *http.Response) {
	req := request.HttpRequest
    clientIP := request.RemoteAddr.String()

    if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
        clientIP = clientIP[:colon]
    }	
	log := logger.NewLogger()
	log.AddFilter("stdout", logger.DEBUG, logger.NewConsoleLogWriter())
	logRecord :=  clientIP +" - - ["+time.Now().Format("02/Jan/2006 03:04:05")+" +8000] \""+req.Method+" "+req.RequestURI+" "+req.Proto+"\" " + fmt.Sprintf("%d %d ", res.StatusCode,res.ContentLength) + req.Referer() + " "+req.UserAgent()
	log.Info(logRecord)	
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
