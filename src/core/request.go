package core

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type Request struct {
	ID          string
	Status      int
	StartTime   time.Time
	EndTime     time.Time
	Overhead    time.Duration
	connection  net.Conn
	RemoteAddr  *net.TCPAddr
	ServerAddr  string
	HttpRequest *http.Request
	Context     map[string]interface{}
}

// Used internally to create and initialize a new request.
func newRequest(request *http.Request, conn net.Conn, startTime time.Time) *Request {
	fReq := new(Request)
	fReq.Context = make(map[string]interface{})
	fReq.HttpRequest = request
	fReq.StartTime = startTime
	fReq.connection = conn
	fReq.Status = 0
	if conn != nil {
		fReq.RemoteAddr = conn.RemoteAddr().(*net.TCPAddr)
	}

	var ut = fReq.StartTime.UnixNano()
	fReq.ID = fmt.Sprintf("%010x", (ut-(ut-(ut%1e12)))+int64(rand.Intn(999)))

	return fReq
}

func (fReq *Request) SetServerAddr(addr string) {
	fReq.ServerAddr = addr
}

func (fReq *Request) finishRequest() {
	fReq.EndTime = time.Now()
	fReq.Overhead = fReq.EndTime.Sub(fReq.StartTime)
}
