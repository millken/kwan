package main
 
import (
"fmt"
"github.com/fitstar/falcore"
"github.com/fitstar/falcore/filter"
"net/http"
"flag"
"time"
)
 
var (
port = flag.Int("port", 8000, "the port to listen on")
)
 
// Change the 3 to 0 to disable timeout and everything works properly
var timeout time.Duration = 3*time.Second
var proxyFilter = filter.NewUpstream(filter.NewUpstreamTransport("112.125.239.95", 80, timeout, nil))
 
func main() {
flag.Parse()
pipeline := falcore.NewPipeline()
pipeline.Upstream.PushBack(&UrlFilter{})
pipeline.Upstream.PushBack(proxyFilter)
server := falcore.NewServer(*port, pipeline)
if err := server.ListenAndServe(); err != nil {
fmt.Println("Could not start server:", err)
}
return
}
 
type UrlFilter struct {
}
 
func (f *UrlFilter) FilterRequest(req *falcore.Request) *http.Response {
req.HttpRequest.Host = "www.wiz.cn"
req.HttpRequest.URL.Host = req.HttpRequest.Host
return nil
}
