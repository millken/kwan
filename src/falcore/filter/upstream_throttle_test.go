package filter

import (
	"fmt"
	"github.com/fitstar/falcore"
	"net/http"
	// "strconv"
	"testing"
)

const REQ_COUNT = 100

func TestUpstreamThrottle(t *testing.T) {
	// Build a thing for
	var started = make(chan chan bool, REQ_COUNT+1)

	// Start a test server
	sleepPipe := falcore.NewPipeline()
	sleepPipe.Upstream.PushBack(falcore.NewRequestFilter(func(req *falcore.Request) *http.Response {
		// get chan
		// j, _ := strconv.Atoi(req.HttpRequest.URL.Query().Get("j"))
		// fmt.Println(req.HttpRequest.URL, j)
		c := make(chan bool)
		started <- c
		// wait on chan
		<-c
		// fmt.Println("DONE")
		return falcore.StringResponse(req.HttpRequest, 200, nil, "OK")
	}))
	sleepSrv := falcore.NewServer(0, sleepPipe)
	defer sleepSrv.StopAccepting()
	go func() {
		sleepSrv.ListenAndServe()
	}()
	<-sleepSrv.AcceptReady

	// Build Upstream
	up := NewUpstream(NewUpstreamTransport("localhost", sleepSrv.Port(), 0, nil))
	// pipe := falcore.NewPipeline()
	// pipe.Upstream.PushBack(up)

	resCh := make(chan *http.Response, REQ_COUNT)
	var i int64 = 1
	for ; i < 12; i++ {
		// fmt.Println("Testing with limit", i)

		up.SetMaxConcurrent(i)

		for j := 0; j < REQ_COUNT; j++ {
			var jj = j
			go func() {
				// fmt.Println("STARTING")
				req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost/foo?j=%v", jj), nil)
				_, res := falcore.TestWithRequest(req, up, nil)
				res.Body.Close()
				resCh <- res
				// fmt.Println("OK")
			}()
		}
		for j := 0; j < REQ_COUNT; j++ {
			// make sure we haven't gone over the limit
			// fmt.Println(i, len(started))
			if r := int64(len(started)); r > i {
				t.Errorf("%v: Over the limit: %v", i, r)
			}

			// send a finish signal
			(<-started) <- true

			// collect the result
			res := <-resCh
			if res.StatusCode != 200 {
				t.Fatalf("Error: %v", res)
			}
		}
	}

}
