package main
import (
	"core"
	"net/http"
	"filter"
	"config"
)

func filterManager() {
	//var filter2 helloFilter
	core.AddRequestFilter(&filter.VhostRouter{"http"})
	var statusfilter filter.StatusFilter	
	core.AddRequestFilter(statusfilter)

	ddosfilter := filter.NewDdosFilter()
	core.AddRequestFilter(ddosfilter)

	cachefilter := filter.NewCacheFilter(filter.NewGoDiskCacheParams(config.GetCacheSetting().Path, config.GetCacheSetting().HotItem))
	core.AddRequestFilter(cachefilter)
	core.AddResponseFilter(cachefilter)

	var upstreamfilter filter.UpstreamFilter
	core.AddRequestFilter(upstreamfilter)
}
type helloFilter int

func (f helloFilter) FilterRequest(req *core.Request) *http.Response {
	return core.StringResponse(req.HttpRequest, 200, nil, "hello world!\n")
}