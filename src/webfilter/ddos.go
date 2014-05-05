package webfilter

import (
	"config"
	"github.com/millken/falcore"
	"net/http"
	"net"
	"net/url"
	"sync/atomic"
	"cache"
	"utils"
	"time"
	"sync"
)

const (
	DDOS_JS_FUNC = 1
	DDOS_FLASH = 2
	DDOS_CODE = 3
)
type DdosFilter map[string]*DdosFilterThrottler

type DdosFilterThrottler struct {
	Cache cache.Cache
	count  int64
	status bool
	check_ticker *time.Ticker
	valid_ticker *time.Ticker
	tickerM     *sync.RWMutex
}

// type check ,if no method ,compile error
var _ falcore.RequestFilter = new(DdosFilter)

func NewDdosFilter() (df DdosFilter) {
	df = make(map[string]*DdosFilterThrottler)
	return
}
func (df DdosFilter) FilterRequest(request *falcore.Request) *http.Response {
	req := request.HttpRequest
	vhost := request.Context["config"].(config.Vhost)

	if vhost.Ddos.Rtime == 0 || vhost.Ddos.Request == 0 {
		return nil
	}
	vhostname := vhost.Name

	//falcore.Debug("%s r=%d rt=%d m=%d st=%d", vhostname, vhost.Ddos.Request, vhost.Ddos.Rtime, vhost.Ddos.Mode, vhost.Ddos.Stime)

	if _, ok := df[vhostname]; !ok {
		df[vhostname] = new(DdosFilterThrottler)
		df[vhostname].Cache = cache.NewRedisCache(config.GetRedis().Addr, config.GetRedis().Password)
		df[vhostname].count = 0
		df[vhostname].status = false
		df[vhostname].check_ticker = time.NewTicker(time.Second * time.Duration(vhost.Ddos.Rtime))
		df[vhostname].valid_ticker = time.NewTicker(time.Second * time.Duration(vhost.Ddos.Stime))
		df[vhostname].tickerM = new(sync.RWMutex)
	}
	df[vhostname].tickerM.RLock()
	ct := df[vhostname].check_ticker
	vt := df[vhostname].valid_ticker
	df[vhostname].tickerM.RUnlock()

	if vt != nil &&  df[vhostname].status {
		RemoteAddr := request.RemoteAddr.String()
		ip, _, _ := net.SplitHostPort(RemoteAddr)
		ckey := "ddos:" + vhostname + ":" + ip
		cval, _ := df[vhostname].Cache.Get(ckey)
		if cval == "pass" {
			return nil
		}
		if cval == "" {
			cval = utils.RandomString(5)
			df[vhostname].Cache.SetEx(ckey, 5, cval)
		}
		isjoin := df.isJoinToWhitelist(req.URL, cval)
		if isjoin {
			df[vhostname].Cache.SetEx(ckey, 86400, "pass")
			return nil
		}
		response := df.getDdosBody(req.URL, cval, vhost.Ddos.Mode)
		return falcore.StringResponse(request.HttpRequest, 200, nil, response)
	}
	if ct != nil {
		atomic.AddInt64(&df[vhostname].count, 1)

		go func() {
			for {
				select {
				case <-ct.C:
					rps := atomic.LoadInt64(&df[vhostname].count)
					atomic.StoreInt64(&df[vhostname].count, 0)
					//falcore.Debug("%s RPS: %d", vhostname, atomic.LoadInt64(&df[vhostname].count))
					if rps >= vhost.Ddos.Request {
						df[vhostname].status = true
					}					
				case <-vt.C:
					df[vhostname].status = false
				}
			}
			//atomic.AddInt64(&df[vhostname].count, -1)
		}()
	}
	return nil
}

func (df DdosFilter) getDdosBody(uri *url.URL, key string, mode int32) (body string) {
	q := uri.Query()
	q.Set("_xko", key)
	uri.RawQuery = q.Encode()
	switch mode {
		case DDOS_JS_FUNC :
			body = `<html><script>window.top.location = "`+utils.ToUnicode(uri.RequestURI())+`";</script></html>`
		default :
			body = "the site was been attacked!"
	}
	return 
}

func (df DdosFilter) isJoinToWhitelist(uri *url.URL, key string) bool {
	q := uri.Query()
	qkey := q.Get("_xko")
	if qkey != "" && qkey == key {
		return true
	}
	return false
}