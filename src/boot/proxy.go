package boot

import (
	"config"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"store"
	"strings"
	"structs"
	"sync"
	"time"
	//"strconv"
)

var mutex sync.Mutex

type Proxy struct {
	backends []*url.URL
	// The transport used to perform proxy requests.
	// If nil, http.DefaultTransport is used.
	// http.RoundTripper is an interface
	Transport http.RoundTripper
	Store     store.Store
	// Per-host cache key prefixes
	Prefixes map[string]int
	Vhost    config.Vhost
}

func NewProxy() (proxy *Proxy) {

	proxy = &Proxy{
		Prefixes: make(map[string]int),
		Store:    store.NewCache2goStore(),
	}
	return
}

func (p *Proxy) AddBackend(ip string, port int) {

	host := fmt.Sprintf("http://%s:%d", ip, port)
	url, _ := url.Parse(host)

	p.backends = append(p.backends, url)

}

func (p *Proxy) SetStore(s store.Store) {
	p.Store = s
}
func (p *Proxy) SetVhost(vhost config.Vhost) {
	p.Vhost = vhost
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	cancache, _ := p.checkCacheRule(req)
	cacheKey := p.cacheKey(req)
	if cancache {
		data, err := p.Store.Get(cacheKey)
		if err == nil { // cache hit. Serve it.
			rw.Header().Add("Cache", "hit")
			p.serveFromCache(data, rw, req)
		} else { // Cache miss. Proxy and cache.
			fmt.Printf("cache miss: %s\n", err)
			rw.Header().Add("Cache", "miss")
			p.proxyAndCache(cacheKey, rw, req)
		}
	} else {
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

}

func (p *Proxy) proxyAndCache(cacheKey string, rw http.ResponseWriter, req *http.Request) {

	backendResp, err := p.proxy(rw, req)

	if err != nil {
		// log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// log.Println("http: response code: ", backendResp.StatusCode)
	/* Cache in Redis
	------------------------------------*/
	cached_response := structs.NewCachedResponse(backendResp)
	/* Copy headers
	------------------------------------*/
	copyHeader(rw.Header(), backendResp.Header)

	/* Only cache successful response
	------------------------------------*/
	var body []byte
	body = cached_response.Body

	if backendResp.StatusCode >= 200 && backendResp.StatusCode < 300 {
		fmt.Println("SUCCESS")
		if req.Method == "HEAD" || req.Method == "OPTIONS" {
			// log.Println("NO BODY", req.Method)
			rw.Write([]byte{})
			req.Body.Close()
		} else if req.Header.Get("If-Modified-Since") != "" && req.Header.Get("If-None-Match") != "" {
			// if client is sending if-modified or if-non-match
			// we assume that they already have a copy of the body
			// log.Println("Client has copy. Send 304")
			rw.WriteHeader(http.StatusNotModified)
			body = []byte{}
		} else {
			/* Copy status code
			   ------------------------------------*/
			rw.WriteHeader(backendResp.StatusCode)
		}

		rw.Write(body)
		go p.cache(cacheKey, cached_response)
	} else { // error. Copy body.
		fmt.Println("Error Status", backendResp.StatusCode)
		rw.WriteHeader(backendResp.StatusCode)
		io.Copy(rw, backendResp.Body)
	}

}

func (p *Proxy) cache(key string, cached_response *structs.CachedResponse) {
	// encode
	fmt.Println("CACHE NOW", cached_response.Headers)
	encoded, _ := serializeResponse(cached_response)
	p.Store.Set(key, 60, encoded)
}

func (p *Proxy) serveFromCache(data []byte, rw http.ResponseWriter, req *http.Request) {
	cached_response, _ := deserializeResponse(data)
	/* Copy headers
	------------------------------------*/
	copyHeader(rw.Header(), cached_response.Headers)

	if req.Method == "HEAD" || req.Method == "OPTIONS" {
		rw.Header().Set("Connection", "close")
		rw.Write([]byte{})
	} else {
		rw.Write(cached_response.Body)
	}
}

func (p *Proxy) checkCacheRule(req *http.Request) (cache bool, result config.Cache) {
	cache = false
	result = config.Cache{}
	uri := req.URL
	url := uri.String()
	fileext := strings.Replace(filepath.Ext(uri.Path), ".", "", 1)
	for _, cr := range p.Vhost.Cache {
		fmt.Printf("cache rule[%s]: %s\n", fileext, cr.FileExt)
		if cr.Url != "" && strings.Index(url, cr.Url) != -1 {
			cache = true
			result = cr
		}
		if cr.FileExt != "" && strings.Index(cr.FileExt, fileext) != -1 {
			cache = true
			result = cr
		}
	}
	return
}

// This needs a mutex or channel/goroutine
func (p *Proxy) initializePrefix(hostName string, fn func(int)) (prefix int) {
	mutex.Lock()
	prefix = p.Prefixes[hostName]
	if prefix == 0 { // start prefix as unix timestamp
		ts := time.Now().Unix()
		prefix = int(ts)
		p.Prefixes[hostName] = prefix
	}
	fn(prefix)
	mutex.Unlock()
	return
}

func (p *Proxy) cacheKey(req *http.Request) string {
	key := req.URL.String()
	//prefix := p.initializePrefix(req.Host, func(int){})
	//stprefix := strconv.Itoa(prefix)
	s := []string{"caching", req.Method, req.Host, key}
	return strings.Join(s, ":")
}
func serializeResponse(res *structs.CachedResponse) (raw []byte, err error) {

	raw, err = msgpack.Marshal(res)

	return
}

func deserializeResponse(raw []byte) (res *structs.CachedResponse, err error) {

	err = msgpack.Unmarshal(raw, &res)

	return
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
