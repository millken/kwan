package webfilter

import (
	"github.com/fitstar/falcore"
	"github.com/fitstar/falcore/filter"
	"github.com/vmihailenco/msgpack"
	"config"
	"net/http"
	"store"
	"structs"
	"path/filepath"
	"regexp"
	"io/ioutil"
	"bytes"
	"strings"	
	"fmt"
	"time"
)


type CacheFilter struct {
	Store     store.Store
	Vhost    config.Vhost
}

const DefaultTimeFormat = "2006-01-02 15:04:05.999999999"

func NewCacheFilter()(cf *CacheFilter) {
 	cf = &CacheFilter{
 		Store:    store.NewCache2goStore(),
 	}
 	return
}

func (c *CacheFilter) SetVhost(vhost config.Vhost) {
	c.Vhost = vhost
}

func (c *CacheFilter) SetStore(s store.Store) {
	c.Store = s
}

func (c *CacheFilter) checkCacheRule(req *http.Request) (cache bool, result config.Cache) {
	cache = false
	result = config.Cache{}

    uncacheable_headers := []string{
        "Proxy-Authenticate",
        "Proxy-Authorization",
        "TE",
        "Trailers",
        "Upgrade",
    }
    for _, uncacheable_header := range uncacheable_headers {
    	if req.Header.Get(uncacheable_header) != "" {
    		return
    	}
    }

	uri := req.URL
	url := uri.String()
	fileext := strings.Replace(filepath.Ext(uri.Path), ".", "", 1)
	re := regexp.MustCompile("no-cache|no-store|private") //Pragma?
	for _, cr := range c.Vhost.Cache {
		if cr.Url != "" && strings.Index(url, cr.Url) != -1 {
			if cr.Static || (!cr.Static && re.FindStringIndex(req.Header.Get("Cache-Control")) == nil) {
				cache = true
				result = cr
			}

		}
		if fileext != "" && strings.Index(cr.FileExt, fileext) != -1 {
			if cr.Static || (!cr.Static && re.FindStringIndex(req.Header.Get("Cache-Control")) == nil) {
				cache = true
				result = cr
			}
		}
	}
	if cache {
		fmt.Printf("hit cache Rule: %s => %v\n", url, result)
	}
	return
}

func (c *CacheFilter) cacheKey(req *http.Request) string {
	key := req.URL.String()
	s := []string{"caching", req.Method, req.Host, key}
	return strings.Join(s, ":")
}


func (c *CacheFilter) cache(key string, cacheRule config.Cache, cached_response *structs.CachedResponse) {
	// encode
	encoded, err := serializeResponse(cached_response)
	if err == nil {
		//fmt.Printf("proxy set cache ok %s %d\n", key, cacheRule.Time)
		c.Store.Set(key, cacheRule.Time, encoded)
	} else {
		fmt.Printf("proxy set cache failue :%s\n", err)
	}
}

func (c *CacheFilter) serveFromCache(data []byte, request *falcore.Request) (res *http.Response) {
	req := request.HttpRequest
	request.CurrentStage.Status = 1 // Skipped (default)
	cached_response, _ := deserializeResponse(data)
	res = falcore.ByteResponse(req, cached_response.StatusCode, cached_response.Headers, cached_response.Body)
	if if_none_match := req.Header.Get("If-None-Match"); if_none_match != "" {
		if cached_response.Headers.Get("Etag") == if_none_match {
			res.StatusCode = 304
			res.Status = "304 Not Modified"
			res.Body.Close()
			res.Body = nil
			res.ContentLength = 0
			request.CurrentStage.Status = 0 // Success
			return 
		}
	}			

	if req.Header.Get("If-Modified-Since") != "" && cached_response.Headers.Get("Last-Modified") != "" {
		t1, err1 := time.Parse(DefaultTimeFormat, req.Header.Get("If-Modified-Since"))
		t2, err2 := time.Parse(DefaultTimeFormat, cached_response.Headers.Get("Last-Modified"))
		if err1 != nil && err2 != nil {
			if t2.Unix() <= t1.Unix() {
				res.StatusCode = 304
				res.Status = "304 Not Modified"
				res.Body.Close()
				res.Body = nil
				res.ContentLength = 0
				request.CurrentStage.Status = 0 // Success
				return
			}
		}
	}
	

	if req.Method == "HEAD" || req.Method == "OPTIONS" {
		res.Header.Set("Connection", "close")
		res.Body.Close()
		res.Body = nil
		request.CurrentStage.Status = 0 // Success
	} else {
			res.Body = ioutil.NopCloser(bytes.NewBuffer(cached_response.Body))
			res.ContentLength = cached_response.ContentLength
			request.CurrentStage.Status = 0 // Success		
	}
	return
}
func (c *CacheFilter) FilterRequest(request *falcore.Request) (res *http.Response) {
	req := request.HttpRequest
	sHost, sPort := "", 80
	_, dPort := SplitHostPort(request.ServerAddr, 80)
	host, port := SplitHostPort(req.Host, dPort)

	if strings.Index("GET|POST", req.Method)  == -1 {
		res = falcore.StringResponse(req, 405, nil, "Method Not Allowed")
		request.CurrentStage.Status = 0
		return
	}
	cancache, cacheRule := c.checkCacheRule(req)
	cacheKey := c.cacheKey(req)
	sHost, sPort = GetSourceIP(host, port, c.Vhost)
	if sPort != 443 && req.URL.Scheme == "https" {
		request.HttpRequest.URL.Scheme = "http"
	}
	//falcore.Debug("source : %s:%d\n", sHost, sPort)
	var timeout time.Duration = 3 * time.Second

	request.HttpRequest.URL.Host = host
	if cancache {
		data, err := c.Store.Get(cacheKey)
		if err == nil { // cache hit. Serve it.
			res = c.serveFromCache(data, request)
			res.Header.Set("X-Cache", "Hit from " + config.GetHostname())
		} else { // Cache miss. Proxy and cache.
			//res = falcore.ByteResponse(req, 100, nil, []byte{})
			proxyFilter := filter.NewUpstream(filter.NewUpstreamTransport(sHost, sPort, timeout, nil))
			request.HttpRequest.Header.Del("If-None-Match")
			request.HttpRequest.Header.Del("If-Modified-Since")
			request.HttpRequest.Header.Set("Cache-Control", "no-cache, no-store")
			request.HttpRequest.Header.Set("Pragma", "no-cache")


			res = proxyFilter.FilterRequest(request)
			if res.StatusCode >= 200 && res.StatusCode < 300 {
				cached_response := structs.NewCachedResponse(res)
				go c.cache(cacheKey, cacheRule, cached_response)
				res.Body = ioutil.NopCloser(bytes.NewBuffer(cached_response.Body))
			}
			res.Header.Set("X-Cache", "Miss from " + config.GetHostname())
		}
	}else{
		proxyFilter := filter.NewUpstream(filter.NewUpstreamTransport(sHost, sPort, timeout, nil))
		res = proxyFilter.FilterRequest(request)			
	}

	return 
}

func (c *CacheFilter) FilterResponse(request *falcore.Request, res *http.Response) {
	//req := request.HttpRequest
	
}

func serializeResponse(res *structs.CachedResponse) (raw []byte, err error) {

	raw, err = msgpack.Marshal(res)

	return
}

func deserializeResponse(raw []byte) (res *structs.CachedResponse, err error) {

	err = msgpack.Unmarshal(raw, &res)

	return
}