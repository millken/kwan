package filter

import (
	"github.com/fitstar/falcore"
	"io/ioutil"
	// "math"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"
)

var upstreamTimeoutTestData = []struct {
	Name       string
	Time       time.Duration
	BodyTime   time.Duration
	StatusCode int
	BodyLen    int64
}{
	{
		"fast",
		100 * time.Millisecond,
		0,
		200,
		1,
	},
	{
		"slow start",
		2 * time.Second,
		0,
		504,
		0, // ignored
	},
	{
		"slow body",
		100 * time.Millisecond,
		time.Millisecond,
		200,
		2048,
	},
}

func TestUpstreamTimeout(t *testing.T) {
	// Start a test server
	sleepPipe := falcore.NewPipeline()
	sleepPipe.Upstream.PushBack(falcore.NewRequestFilter(func(req *falcore.Request) *http.Response {
		tt, _ := strconv.Atoi(req.HttpRequest.URL.Query().Get("time"))
		b, _ := strconv.Atoi(req.HttpRequest.URL.Query().Get("body"))
		bl, _ := strconv.Atoi(req.HttpRequest.URL.Query().Get("bl"))

		time.Sleep(time.Duration(tt))
		pr, pw := io.Pipe()
		go func() {
			buf := make([]byte, 1024)
			for i := 0; i < bl; i++ {
				<-time.After(time.Duration(b))
				pw.Write(buf)
			}
			pw.Close()
		}()
		return falcore.SimpleResponse(req.HttpRequest, 200, nil, int64(bl*1024), pr)
	}))
	sleepSrv := falcore.NewServer(0, sleepPipe)
	go func() {
		sleepSrv.ListenAndServe()
	}()
	<-sleepSrv.AcceptReady

	// Build Upstream
	up := NewUpstream(NewUpstreamTransport("localhost", sleepSrv.Port(), time.Second, nil))

	for _, test := range upstreamTimeoutTestData {
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost/test?time=%v&body=%v&bl=%v", int64(test.Time), int64(test.BodyTime), test.BodyLen), nil)
		_, res := falcore.TestWithRequest(req, up, nil)
		if res.StatusCode != test.StatusCode {
			t.Errorf("%v Expected status %v Got %v", test.Name, test.StatusCode, res.StatusCode)
		}

		if res.StatusCode == 200 {
			i, _ := io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
			if i != (test.BodyLen * 1024) {
				t.Errorf("%v Expected body len %v Got %v", test.Name, (test.BodyLen * 1024), i)
			}
		}
	}
}
