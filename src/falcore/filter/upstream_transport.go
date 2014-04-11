package filter

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fitstar/falcore"
)

// Holds all the information about how to lookup and
// connect to an upstream.
type UpstreamTransport struct {
	DNSCacheDuration time.Duration

	host string
	port int

	tcpaddr          *net.TCPAddr
	tcpaddrCacheTime time.Time

	transport *http.Transport
	timeout   time.Duration
}

// transport is optional.  We will override Dial
func NewUpstreamTransport(host string, port int, timeout time.Duration, transport *http.Transport) *UpstreamTransport {
	ut := &UpstreamTransport{
		host:      host,
		port:      port,
		timeout:   timeout,
		transport: transport,
	}
	ut.DNSCacheDuration = 15 * time.Minute

	if ut.transport == nil {
		ut.transport = &http.Transport{}
		ut.transport.MaxIdleConnsPerHost = 15
	}

	ut.transport.Dial = func(n, addr string) (c net.Conn, err error) {
		return ut.dial(n, addr)
	}

	return ut
}

func (t *UpstreamTransport) dial(n, a string) (c net.Conn, err error) {
	var addr *net.TCPAddr
	addr, err = t.lookupIp()

	falcore.Fine("Dialing connection to %v", addr)
	var ctcp *net.TCPConn
	ctcp, err = net.DialTCP("tcp4", nil, addr)
	if err != nil {
		falcore.Error("Dial Failed: %v", err)
		return
	}

	// FIXME: Go1 has a race that causes problems with timeouts
	// Recommend disabling until Go1.1
	if t.timeout > 0 {
		c = &timeoutConnWrapper{Conn: ctcp, timeout: t.timeout}
	} else {
		c = ctcp
	}

	return
}

func (t *UpstreamTransport) lookupIp() (addr *net.TCPAddr, err error) {
	// Cached tcpaddr
	if t.tcpaddr != nil && (t.DNSCacheDuration == 0 || t.tcpaddrCacheTime.Add(t.DNSCacheDuration).After(time.Now())) {
		return t.tcpaddr, nil
	}

	ips, err := net.LookupIP(t.host)
	var ip net.IP = nil

	if err != nil {
		return nil, err
	}

	// Find first IPv4 IP
	for i := range ips {
		ip = ips[i].To4()
		if ip != nil {
			break
		}
	}

	if ip != nil {
		t.tcpaddr = &net.TCPAddr{}
		t.tcpaddr.Port = t.port
		t.tcpaddr.IP = ip
		t.tcpaddrCacheTime = time.Now()
		addr = t.tcpaddr
	} else {
		errstr := fmt.Sprintf("Can't get IP addr for %v: %v", t.host, err)
		err = errors.New(errstr)
	}

	return
}

type timeoutConnWrapper struct {
	net.Conn
	timeout time.Duration
}

func (cw *timeoutConnWrapper) setDeadline() error {
	return cw.Conn.SetDeadline(time.Now().Add(cw.timeout))
}

func (cw *timeoutConnWrapper) Write(b []byte) (int, error) {
	if err := cw.setDeadline(); err != nil {
		return 0, err
	}
	return cw.Conn.Write(b)
}

func (cw *timeoutConnWrapper) Read(b []byte) (n int, err error) {
	if err := cw.setDeadline(); err != nil {
		return 0, err
	}
	return cw.Conn.Read(b)
}
