package main
import (
	"core"
	"net/http"
	"filter"
)

func filterManager() {
	//var filter2 helloFilter
	core.AddRequestFilter(&filter.VhostRouter{"http"})
	var statusfilter filter.StatusFilter	
	core.AddRequestFilter(statusfilter)

	ddosfilter := filter.NewDdosFilter()
	core.AddRequestFilter(ddosfilter)

	var upstreamfilter filter.UpstreamFilter
	core.AddRequestFilter(upstreamfilter)

	cachefilter := filter.NewCacheFilter(filter.NewGoDiskCacheParams("./cacher/", 100))
	core.AddResponseFilter(cachefilter)
}
type helloFilter int

func (f helloFilter) FilterRequest(req *core.Request) *http.Response {
	return core.StringResponse(req.HttpRequest, 200, nil, "hello world!\n")
}