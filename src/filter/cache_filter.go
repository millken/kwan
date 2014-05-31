//base from https://github.com/cbinsights/godiskcache
package filter

import (
	"bytes"
	"config"
	"core"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/golang/groupcache/lru"
	"github.com/vmihailenco/msgpack"
	"io/ioutil"
	"logger"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"utils"
)

type GoDiskCache struct {
	mutex           sync.RWMutex
	cachePrefix     string
	memCache        *lru.Cache
	memMaxCacheSize int
} //struct

type Params struct {
	Directory string
	MemItems  int
} //struct

type DataWrapper struct {
	Ts   time.Time
	Data []byte
} // struct

var directory string = os.TempDir()

const DefaultTimeFormat = "2006-01-02 15:04:05.999999999"

func NewCacheFilter(p *Params) *GoDiskCache {
	var items int = 100000

	if len(p.Directory) > 0 {
		directory = p.Directory
		err := os.MkdirAll(directory, 0744)

		if err != nil {
			logger.Error(err)
		} //if
	} //if

	if p.MemItems != 0 {
		items = p.MemItems
	}

	dc := &GoDiskCache{}
	dc.cachePrefix = path.Join(directory, "godiskcache_")
	dc.mutex = sync.RWMutex{}
	dc.memCache = lru.New(items)
	dc.memMaxCacheSize = 1000000

	return dc
} //New

func NewGoDiskCacheParams(directory string, items int) *Params {
	return &Params{directory, items}
} //NewParams

func (dc *GoDiskCache) SetPrefix(prefix string) {
	dc.cachePrefix = path.Join(directory, prefix)
}

func (dc *GoDiskCache) SetMemMaxCacheSize(size int) {
	dc.memMaxCacheSize = size
}

func (dc *GoDiskCache) FilterRequest(request *core.Request) (res *http.Response) {
	req := request.HttpRequest
	vhost := request.Context["config"].(*config.Vhost)
	cancache, cacheRule := dc.checkCacheRule(req, vhost)
	logger.Fine("url = %s, cache = %v , rule = %v", req.URL, cancache, cacheRule)
	cacheKey := req.URL.String()
	if cancache {
		tmppos := strings.Index(cacheKey, "?")
		if cacheRule.IgnoreParam && tmppos > 0 {
			cacheKey = cacheKey[:tmppos]
		}
		dc.SetPrefix(vhost.Name)
		data, err := dc.Get(cacheKey, cacheRule.Time)
		if err != nil {
			request.HttpRequest.Header.Del("If-None-Match")
			request.HttpRequest.Header.Del("If-Modified-Since")
			//request.HttpRequest.Header.Set("Cache-Control", "no-cache, no-store")
			//request.HttpRequest.Header.Set("Pragma", "no-cache")
			logger.Warn(err)
		} else {
			return dc.serveFromCache(data, request)
		}
	}
	return nil
}
func (dc *GoDiskCache) FilterResponse(request *core.Request, res *http.Response) {
	if request.Status&STATUS_CACHE_HITS == STATUS_CACHE_HITS || request.Status&STATUS_DDOS == STATUS_DDOS {
		return
	}
	req := request.HttpRequest
	vhost := request.Context["config"].(*config.Vhost)
	cancache, cacheRule := dc.checkCacheRule(req, vhost)

	cacheKey := req.URL.String()
	if cancache {
		tmppos := strings.Index(cacheKey, "?")
		if cacheRule.IgnoreParam && tmppos > 0 {
			cacheKey = cacheKey[:tmppos]
		}
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			cached_response, err := utils.NewCachedResponse(res)
			if err != nil {
				logger.Warn("write cache error : %s", err.Error())
				return
			}
			if vhost.Limit.MaxCacheSize > 0 {
				dc.memMaxCacheSize = vhost.Limit.MaxCacheSize
			} else {
				dc.memMaxCacheSize = 1000000
			}
			dc.SetPrefix(vhost.Name)
			dc.cache(cacheKey, cached_response)
			res.Body = ioutil.NopCloser(bytes.NewBuffer(cached_response.Body))
			res.Header.Set("X-Cache", "Miss from "+config.GetHostname())
		}
	}
}

func (dc *GoDiskCache) Get(key string, lifetime int) ([]byte, error) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			logger.Error(rec)
		} //if
	}() //func

	// Take the reader lock
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	// check the in-memory cache first
	if val, ok := dc.memCache.Get(key); ok {
		dw := val.(DataWrapper)
		if int(time.Since(dw.Ts).Seconds()) < lifetime {
			return dw.Data, err
		}
	}

	//open the cache file
	if file, err := os.Open(dc.buildFileName(key)); err == nil {
		defer file.Close()
		//get stats about the file, need modified time
		if fi, err := file.Stat(); err == nil {
			//check that cache file is still valid
			if int(time.Since(fi.ModTime()).Seconds()) < lifetime {
				//try reading entire file
				if data, err := ioutil.ReadAll(file); err == nil {
					// update the cache with this value
					dc.memCache.Add(key, DataWrapper{Ts: fi.ModTime(),
						Data: data})

					return data, err
				} //if
			} //if
		} //if
	} //if

	return []byte{}, errors.New("cache not found")
} //Get

func (dc *GoDiskCache) Set(key string, data []byte) error {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			logger.Error(rec)
		} //if
	}() //func

	// Take the writer lock
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	//open the file
	if file, err := os.Create(dc.buildFileName(key)); err == nil {
		_, err = file.Write(data)

		// store it in the in-memory cache
		if fi, err := file.Stat(); err == nil {
			ts := fi.ModTime()
			dc.memCache.Add(key, DataWrapper{Ts: ts, Data: data})
		}

		_ = file.Close()
	} //if

	return err
} //func

func (dc *GoDiskCache) buildFileName(key string) string {
	//hash the byte slice and return the resulting string
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return dc.cachePrefix + hex.EncodeToString(hasher.Sum(nil))
} //buildFileName

func (dc *GoDiskCache) checkCacheRule(req *http.Request, cacher *config.Vhost) (cache bool, result config.Cache) {
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
	for _, cr := range cacher.Cache {
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
	/*
		if cache {
			fmt.Printf("hit cache Rule: %s => %v\n", url, result)
		}
	*/
	return
}
func (dc *GoDiskCache) cache(key string, cached_response *utils.CachedResponse) {
	// encode
	encoded, err := serializeResponse(cached_response)
	if err == nil {
		//fmt.Printf("proxy set cache ok %s %d\n", key, cacheRule.Time)
		if len(encoded) <= dc.memMaxCacheSize {
			go dc.Set(key, encoded)
		}
	} else {
		logger.Warn("proxy set cache failue :%s\n", err)
	}
}

func (dc *GoDiskCache) serveFromCache(data []byte, request *core.Request) (res *http.Response) {
	req := request.HttpRequest
	cached_response, err := deserializeResponse(data)
	if err != nil {
		logger.Warn("get CachedResponse : %s", err)
		return nil
	} else {
		res = core.ByteResponse(req, cached_response.StatusCode, cached_response.Headers, cached_response.Body)
		request.Status = request.Status | STATUS_CACHE_HITS
		res.Header.Set("X-Cache", "Hit from "+config.GetHostname())
	}
	if if_none_match := req.Header.Get("If-None-Match"); if_none_match != "" {
		if cached_response.Headers.Get("Etag") == if_none_match {
			res.StatusCode = 304
			res.Status = "304 Not Modified"
			res.Body.Close()
			res.Body = nil
			res.ContentLength = 0
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
				return
			}
		}
	}

	if req.Method == "HEAD" || req.Method == "OPTIONS" {
		res.Header.Set("Connection", "close")
		res.Body.Close()
		res.Body = nil
	} else {
		res.Body = ioutil.NopCloser(bytes.NewBuffer(cached_response.Body))
		//res.ContentLength = cached_response.ContentLength
	}
	return
}

func serializeResponse(res *utils.CachedResponse) (raw []byte, err error) {

	raw, err = msgpack.Marshal(res)

	return
}

func deserializeResponse(raw []byte) (res *utils.CachedResponse, err error) {

	err = msgpack.Unmarshal(raw, &res)

	return
}
