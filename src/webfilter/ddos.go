package webfilter

import (
	"cache"
	"config"
	"github.com/millken/falcore"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
	"utils"
)

const (
	DDOS_JS_REDIRECT = 10
	DDOS_JS_TIMEOUT  = 11
	DDOS_FLASH       = 2
	DDOS_CODE        = 3
)

type DdosFilter map[string]*DdosFilterThrottler

type DdosFilterThrottler struct {
	Cache        cache.Cache
	count        int64
	status       bool
	check_ticker *time.Ticker
	valid_ticker *time.Ticker
	tickerM      *sync.RWMutex
}

// type check ,if no method ,compile error
var _ falcore.RequestFilter = new(DdosFilter)

func NewDdosFilter() (df DdosFilter) {
	df = make(DdosFilter)
	return
}
func (df DdosFilter) FilterRequest(request *falcore.Request) *http.Response {
	req := request.HttpRequest
	vhost := request.Context["config"].(*config.Vhost)

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

	if vt != nil && df[vhostname].status {
		RemoteAddr := request.RemoteAddr.String()
		ip, _, _ := net.SplitHostPort(RemoteAddr)
		ckey := "ddos:" + vhostname + ":" + ip
		cval, _ := df[vhostname].Cache.Get(ckey)
		if cval == "pass" {
			return nil
		}
		if cval == "" {
			cval = utils.RandomString(utils.RandomInt(3, 7))
			df[vhostname].Cache.SetEx(ckey, 5, cval)
		}
		isjoin := df.isJoinToWhitelist(req.URL, cval)
		ikey := "ccbl:" + vhostname + ":" + ip
		if isjoin {
			df[vhostname].Cache.Do("Del", ikey)
			df[vhostname].Cache.SetEx(ckey, 86400, "pass")
			return nil
		} else {
			df[vhostname].Cache.Do("INCR", ikey)
		}
		response := df.getDdosBody(req.URL.String(), cval, vhost.Ddos.Mode)
		return falcore.StringResponse(request.HttpRequest, 200, nil, response)
	}
	if ct != nil && request.Context["whitelist"] == false {
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

func (df DdosFilter) getDdosBody(link string, key string, mode int32) (body string) {
	uri, _ := url.Parse(link)
	q := uri.Query()
	q.Set("_l1O0", key)
	uri.RawQuery = q.Encode()
	switch mode {
	case DDOS_JS_REDIRECT:
		body = `<html><script>window.top.location = "` + uri.RequestURI() + `";</script></html>`
	case DDOS_JS_TIMEOUT:
		r := rand.Intn(7) + 1
		rkey := utils.RandomString(r)
		a := utils.Base64_encode(utils.Crypt(uri.RequestURI(), rkey))
		body = `<html><script>
		var B={_keyStr:"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=",d:function(input){var output="";var chr1,chr2,chr3;var enc1,enc2,enc3,enc4;var i=0;input=input.replace(/[^A-Za-z0-9\+\/\=]/g,"");while(i<input.length){enc1=this._keyStr.indexOf(input.charAt(i++));enc2=this._keyStr.indexOf(input.charAt(i++));enc3=this._keyStr.indexOf(input.charAt(i++));enc4=this._keyStr.indexOf(input.charAt(i++));chr1=(enc1<<2)|(enc2>>4);chr2=((enc2&15)<<4)|(enc3>>2);chr3=((enc3&3)<<6)|enc4;output=output+String.fromCharCode(chr1);if(enc3!=64){output=output+String.fromCharCode(chr2)};if(enc4!=64){output=output+String.fromCharCode(chr3)}};output=B._d(output);return output},_d:function(utftext){var string="";var i=0;var c=c1=c2=0;while(i<utftext.length){c=utftext.charCodeAt(i);if(c<128){string+=String.fromCharCode(c);i++}else if((c>191)&&(c<224)){c2=utftext.charCodeAt(i+1);string+=String.fromCharCode(((c&31)<<6)|(c2&63));i+=2}else{c2=utftext.charCodeAt(i+1);c3=utftext.charCodeAt(i+2);string+=String.fromCharCode(((c&15)<<12)|((c2&63)<<6)|(c3&63));i+=3}};return string}}
		var A=B.d("` + a + `"),C="` + rkey + `",O="";
		for(var i=0;i<A.length;i++){var k = C.charCodeAt(i%C.length);O += String.fromCharCode(A.charCodeAt(i)^k);}
		window.setTimeout(function(){window.top.location= O;}, 600);
		</script></html>`
	default:
		body = "the site was been attacked!"
	}
	return
}

func (df DdosFilter) isJoinToWhitelist(uri *url.URL, key string) bool {
	q := uri.Query()
	qkey := q.Get("_l1O0")
	if qkey != "" && qkey == key {
		return true
	}
	return false
}
