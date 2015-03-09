package filter

import (
	"bytes"
	"core"
	"io"
	"logger"
	"net"
	"net/http"
	"sync"
	"time"
)

type passThruReadCloser struct {
	io.Reader
	io.Closer
}

// Proxies the request through another server.  Can be used in conjunction with
// the UpstreamPool to load balance traffic across several servers.
type Upstream struct {
	Transport *UpstreamTransport
	// Will ignore https on the incoming request and always upstream http
	ForceHttp bool
	// Ping URL Path-only for checking upness
	PingPath string
	// Throttling
	throttleC        *sync.Cond
	throttleMax      int64
	throttleInFlight int64
	throttleQueue    int64
}

func NewUpstream(transport *UpstreamTransport) *Upstream {
	u := new(Upstream)
	u.Transport = transport
	u.throttleC = sync.NewCond(new(sync.Mutex))
	return u
}

func (u *Upstream) FilterRequest(request *core.Request) (res *http.Response) {
	var err error
	req := request.HttpRequest

	// Throttle
	// Wait for an opening, then increment in flight counter
	u.throttleC.L.Lock()
	u.throttleQueue += 1
	for u.throttleMax > 0 && u.throttleInFlight >= u.throttleMax {
		u.throttleC.Wait()
	}
	u.throttleQueue -= 1
	u.throttleInFlight += 1
	u.throttleC.L.Unlock()
	// Decrement and signal when done
	defer func() {
		u.throttleC.L.Lock()
		u.throttleInFlight -= 1
		u.throttleC.Signal()
		u.throttleC.L.Unlock()
	}()

	// Force the upstream to use http
	if u.ForceHttp || req.URL.Scheme == "" {
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
	}
	before := time.Now()
	//req.Header.Set("Connection", "Keep-Alive")
	var upstrRes *http.Response
	upstrRes, err = u.Transport.transport.RoundTrip(req)
	//logger.Debug("%v", upstrRes)
	diff := time.Now().Sub(before).Seconds()
	if err == nil {
		// Copy response over to new record.  Remove connection noise.  Add some sanity.
		res = core.StringResponse(req, upstrRes.StatusCode, nil, "")
		if upstrRes.ContentLength > 0 {
			res.ContentLength = upstrRes.ContentLength
			res.Body = upstrRes.Body
		} else if res.ContentLength == -1 {
			res.Body = upstrRes.Body
			res.ContentLength = -1
			res.TransferEncoding = []string{"chunked"}
		} else {
			// Any bytes?
			var testBuf [1]byte
			n, _ := io.ReadFull(upstrRes.Body, testBuf[:])
			if n == 1 {
				// Yes there are.  Chunked it is.
				res.TransferEncoding = []string{"chunked"}
				res.ContentLength = -1
				rc := &passThruReadCloser{
					io.MultiReader(bytes.NewBuffer(testBuf[:]), upstrRes.Body),
					upstrRes.Body,
				}

				res.Body = rc
			} else {
				// There was an error reading the body
				upstrRes.Body.Close()
				res.ContentLength = 0
			}
		}
		// Copy over headers with a few exceptions
		res.Header = make(http.Header)
		for hn, hv := range upstrRes.Header {
			switch hn {
			case "Content-Length":
			case "Connection":
			case "Transfer-Encoding":
			default:
				res.Header[hn] = hv
			}
		}
	} else {
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			logger.Error("%s Upstream Timeout error: %v", request.ID, err)
			res = core.StringResponse(req, 504, nil, "Gateway Timeout\n")
			request.Status = 2 // Fail
		} else {
			logger.Error("%s Upstream error: %v", request.ID, err)
			res = core.StringResponse(req, 502, nil, "Bad Gateway\n")
			request.Status = 2 // Fail
		}
	}
	logger.Debug("%s [%s] [%s] %s s=%d Time=%.4f", request.ID, req.Method, u.Transport.host, req.URL, res.StatusCode, diff)
	return
}

// Set the maximum number of concurrent requests to send to upstream
// Set to 0 (the default) to disable throttling.
func (u *Upstream) SetMaxConcurrent(max int64) {
	u.throttleC.L.Lock()
	u.throttleMax = max
	u.throttleC.Broadcast()
	u.throttleC.L.Unlock()
}

// Returns the current maxconcurrent
func (u *Upstream) MaxConcurrent() int64 {
	u.throttleC.L.Lock()
	max := u.throttleMax
	u.throttleC.L.Unlock()
	return max
}

// Returns the number of requests waiting on throttling
func (u *Upstream) QueueLength() int64 {
	u.throttleC.L.Lock()
	ql := u.throttleQueue
	u.throttleC.L.Unlock()
	return ql
}

func (u *Upstream) ping(host string) (up bool) {
	// the url must be syntactically valid for this to work but the host will be ignored because we
	// are overriding the connection always
	request, err := http.NewRequest("GET", "http://"+host+"/", nil)
	request.Header.Set("User-Agent", "googleddos pinger")
	//request.Header.Set("Connection", "Keep-Alive") // not sure if this should be here for a ping
	if err != nil {
		logger.Error("Bad Ping request: %v", err)
		return false
	}
	res, err := u.Transport.transport.RoundTrip(request)

	if err != nil {
		logger.Error("Failed Ping to %v:%v: %v", u.Transport.host, u.Transport.port, err)
		return false
	} else {
		res.Body.Close()
	}
	if res.StatusCode == 200 {
		return true
	}
	logger.Error("Failed Ping to %v:%v: %v", u.Transport.host, u.Transport.port, res.Status)
	// bad status
	return false
}

