//base from https://github.com/cbinsights/godiskcache
package filter

import (
	"core"
	"crypto/sha256"
	"encoding/hex"
	"github.com/golang/groupcache/lru"
	"github.com/vmihailenco/msgpack"
	"io/ioutil"
	"logger"
	"os"
	"path"
	"sync"
	"time"
	"net/http"
	"config"
	"path/filepath"
	"regexp"	
	"strings"
	"utils"
	"bytes"
)


type GoDiskCache struct {
	mutex       sync.RWMutex
	cachePrefix string
	memCache    *lru.Cache
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

	return dc
} //New

func NewGoDiskCacheParams(directory string, items int) *Params {
	return &Params{directory, items}
} //NewParams

func (dc *GoDiskCache) SetPrefix(prefix string) {
	dc.cachePrefix = path.Join(directory, prefix)
}

func (dc *GoDiskCache) FilterResponse(request *core.Request, res *http.Response) {
	req := request.HttpRequest
	vhost := request.Context["config"].(*config.Vhost)
	cancache, cacheRule := dc.checkCacheRule(req, vhost)
	cacheKey := req.URL.String()

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			cached_response, err := utils.NewCachedResponse(res)
			if err != nil {
				logger.Warn("write cache error : %s", err.Error())
				return
			}
			go dc.cache(cacheKey, cached_response)
			res.Body = ioutil.NopCloser(bytes.NewBuffer(cached_response.Body))
		}	
	logger.Debug("%s => %v %v", req.URL, cancache, cacheRule)
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

	return []byte{}, err
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

func (dc *GoDiskCache) checkCacheRule(req *http.Request, cacher *config.Vhost)(cache bool, result config.Cache) {
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
		go dc.Set(key, encoded)
	} else {
		logger.Warn("proxy set cache failue :%s\n", err)
	}
}
func serializeResponse(res *utils.CachedResponse) (raw []byte, err error) {

	raw, err = msgpack.Marshal(res)

	return
}

func deserializeResponse(raw []byte) (res *utils.CachedResponse, err error) {

	err = msgpack.Unmarshal(raw, &res)

	return
}