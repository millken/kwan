package webfilter

import (
	"config"
	"github.com/millken/falcore"
	"net/http"
	"sync/atomic"
	"time"
	"sync"
)

type DdosFilter map[string]*DdosFilterThrottler

type DdosFilterThrottler struct {
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
	vhost := request.Context["config"].(config.Vhost)

	if vhost.Ddos.Rtime == 0 || vhost.Ddos.Request == 0 {
		return nil
	}
	vhostname := vhost.Name

	//falcore.Debug("%s r=%d rt=%d m=%d st=%d", vhostname, vhost.Ddos.Request, vhost.Ddos.Rtime, vhost.Ddos.Mode, vhost.Ddos.Stime)

	if _, ok := df[vhostname]; !ok {
		df[vhostname] = new(DdosFilterThrottler)
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
		return falcore.StringResponse(request.HttpRequest, 200, nil, "the site was been attacked!\n")
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
