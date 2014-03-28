package boot
import (
    "net"
    "net/url"
    "net/http"
    "strings"
    "io"
    "fmt"
)


type Proxy struct {
  backends []*url.URL
  // The transport used to perform proxy requests.
  // If nil, http.DefaultTransport is used.
  // http.RoundTripper is an interface
  Transport http.RoundTripper
  // Per-host cache key prefixes
  Prefixes map[string]int
}

func NewProxy() (proxy *Proxy) {

  proxy = &Proxy{
    Prefixes: make(map[string]int),
  }
  return
}

func (p *Proxy) AddBackend(ip string, port int) {

	host := fmt.Sprintf("http://%s:%d", ip, port)
    url, _ := url.Parse(host)

    p.backends = append(p.backends, url)
  
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
  
    backendResp, err := p.proxy(rw, req)

    if err != nil {
      // log.Printf("http: proxy error: %v", err)
     http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
      return
    }
    //http://golang.org/src/pkg/net/http/httputil/reverseproxy.go , do not put under rw.WriteHeader
    copyHeader(rw.Header(), backendResp.Header)

    rw.WriteHeader(backendResp.StatusCode)

    //rate limiting
    gLink := NewLink(1024 /* kbps */)
    io.Copy(rw, gLink.NewLinkReader(backendResp.Body))
    defer backendResp.Body.Close()

}

func (p *Proxy) director(req *http.Request, backend *url.URL) {
  targetQuery := backend.RawQuery

  req.URL.Scheme = backend.Scheme
  req.URL.Host = backend.Host
  req.URL.Path = singleJoiningSlash(backend.Path, req.URL.Path)
  if targetQuery == "" || req.URL.RawQuery == "" {
    req.URL.RawQuery = targetQuery + req.URL.RawQuery
  } else {
    req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
  }
}

func (p *Proxy) proxy(rw http.ResponseWriter, req *http.Request) (*http.Response, error) {
	backend := p.backends[0] // do round-robin here later
	/* Add forward
	-----------------------------*/
	transport := p.Transport
	if transport == nil {
	transport = http.DefaultTransport
	}

	outreq := new(http.Request)
	*outreq = *req // includes shallow copies of maps, but okay

	p.director(outreq, backend)
	outreq.Method = req.Method
	outreq.Proto = "HTTP/1.1"
	outreq.ProtoMajor = 1
	outreq.ProtoMinor = 1
	outreq.Close = false

	// log.Println("Proxy", outreq.URL.Host, outreq.URL.Path, outreq.URL.RawQuery)

	// Remove the connection header to the backend.  We want a
	// persistent connection, regardless of what the client sent
	// to us.  This is modifying the same underlying map from req
	// (shallow copied above) so we only copy it if necessary.
	if outreq.Header.Get("Connection") != "" {
	outreq.Header = make(http.Header)
	copyHeaderForBackend(outreq.Header, req.Header)
	outreq.Header.Del("Connection")
	}

	if clientIp, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
	outreq.Header.Set("X-Forwarded-For", clientIp)
	}

	backendResp, err := transport.RoundTrip(outreq)

	return backendResp, err
}

func singleJoiningSlash(a, b string) string {
  aslash := strings.HasSuffix(a, "/")
  bslash := strings.HasPrefix(b, "/")
  switch {
  case aslash && bslash:
    return a + b[1:]
  case !aslash && !bslash:
    return a + "/" + b
  }
  return a + b
}

func copyHeader(dst, src http.Header) {
  for k, vv := range src {
    for _, v := range vv {
      dst.Add(k, v)
    }
  }
}
// We don not want the backend to respond with 304
// Because then we don't have a response body to cache!
func copyHeaderForBackend(dst, src http.Header) {
  for k, vv := range src {
    for _, v := range vv {
      if k != "If-Modified-Since" && k != "If-None-Match" {
        dst.Add(k, v)
      }
    }
  }
}