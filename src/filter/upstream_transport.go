package filter

import (
	"crypto/tls"
	"errors"
	"fmt"
	"logger"
	"net"
	"net/http"
	"time"
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
		ut.transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		ut.transport.ResponseHeaderTimeout = time.Second * 1
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

	logger.Fine("Dialing connection to %v", addr)
	var ctcp net.Conn
	ctcp, err = net.DialTimeout("tcp4", addr.String(), time.Second*2)
	if err != nil {
		logger.Error("Dial Failed: %v", err)
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

//https://gist.github.com/seantalts/11266762

type TimeoutTransport struct {
	http.Transport
	RoundTripTimeout time.Duration
}

type respAndErr struct {
	resp *http.Response
	err  error
}

type netTimeoutError struct {
	error
}

func (ne netTimeoutError) Timeout() bool { return true }

// If you don't set RoundTrip on TimeoutTransport, this will always timeout at 0
func (t *TimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	timeout := time.After(t.RoundTripTimeout)
	resp := make(chan respAndErr, 1)

	go func() {
		r, e := t.Transport.RoundTrip(req)
		resp <- respAndErr{
			resp: r,
			err:  e,
		}
	}()

	select {
	case <-timeout: // A round trip timeout has occurred.
		t.Transport.CancelRequest(req)
		return nil, netTimeoutError{
			error: fmt.Errorf("timed out after %s", t.RoundTripTimeout),
		}
	case r := <-resp: // Success!
		return r.resp, r.err
	}
}
