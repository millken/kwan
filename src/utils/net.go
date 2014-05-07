package utils

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

func IpStringToI32(a string) uint32 {
	return IpToI32(net.ParseIP(a))
}

func IpToI32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func I32ToIP(a uint32) net.IP {
	return net.IPv4(byte(a>>24), byte(a>>16), byte(a>>8), byte(a))
}

func BuildCommonLogLine(req *http.Request, res *http.Response) string {
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
