package core

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"io"
	"logger"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
	"runtime"
)

type Server struct {
	Addr         string
	listener     net.Listener
	listenerFile *os.File
}

type Certificates struct {
	CertFile string
	KeyFile  string
}

var (
	perIpConnTracker = createPerIpConnTracker()
)

func NewServer(addr string) *Server {
	s := new(Server)
	s.Addr = addr
	return s
}

func (srv *Server) socketListen() error {
	var la *net.TCPAddr
	var err error
	if la, err = net.ResolveTCPAddr("tcp", srv.Addr); err != nil {
		return err
	}

	var l *net.TCPListener
	if l, err = net.ListenTCP("tcp", la); err != nil {
		return err
	}
	srv.listener = l

	return srv.setupNonBlockingListener(err, l)
}

func (srv *Server) ListenAndServe() error {
	if srv.Addr == "" {
		srv.Addr = ":http"
	}
	if srv.listener == nil {
		if err := srv.socketListen(); err != nil {
			return err
		}
	}
	return srv.serve()
}

func (srv *Server) ListenAndServeTLSSNI(certs []Certificates) error {
	if srv.Addr == "" {
		srv.Addr = ":https"
	}
	config := &tls.Config{
		Rand:       rand.Reader,
		Time:       time.Now,
		NextProtos: []string{"http/1.1"},
	}

	var err error
	config.Certificates = make([]tls.Certificate, len(certs))
	for i, v := range certs {
		config.Certificates[i], err = tls.LoadX509KeyPair(v.CertFile, v.KeyFile)
		if err != nil {
			return err
		}
	}
	config.BuildNameToCertificate()

	if srv.listener == nil {
		if err := srv.socketListen(); err != nil {
			return err
		}
	}

	srv.listener = tls.NewListener(srv.listener, config)

	return srv.serve()
}

func (srv *Server) setupNonBlockingListener(err error, l *net.TCPListener) error {
	if srv.listenerFile, err = l.File(); err != nil {
		return err
	}
	fd := int(srv.listenerFile.Fd())
	if e := syscall.SetNonblock(fd, true); e != nil {
		return e
	}
	return nil
}

func (srv *Server) serve() error {

	for {
		var c net.Conn
		var err error
		if l, ok := srv.listener.(*net.TCPListener); ok {
			l.SetDeadline(time.Now().Add(3 * time.Second))
		}
		c, err = srv.listener.Accept()
		if err != nil {
			if ope, ok := err.(*net.OpError); ok {
				if !(ope.Timeout() && ope.Temporary()) {
					logger.Error("SERVER Accept Error: %v", ope)
				}
			} else {
				logger.Error("SERVER Accept Error: %v", err)
			}			
		} else {
			go srv.handler(c)
		}
	}
	return nil
}

func (srv *Server) handler(c net.Conn) {
	var startTime time.Time

	defer c.Close()
	clientAddr := c.RemoteAddr().(*net.TCPAddr).IP.To4()
	ipUint32 := Ip4ToUint32(clientAddr)
	if perIpConnTracker.RegisterIp(ipUint32) > 50 {
		logger.Debug("Too many concurrent connections (more than %d) from ip=%s. Denying new connection from the ip\n%v\n", 20, clientAddr, perIpConnTracker.GetIpConn())
		perIpConnTracker.UnregisterIp(ipUint32)
		return
	}
	defer perIpConnTracker.UnregisterIp(ipUint32)

	r := bufio.NewReaderSize(c, 1024)
	w := bufio.NewWriterSize(c, 4096)
	clientAddrStr := clientAddr.String()
	for {
		req, err := http.ReadRequest(r)
		if err != nil {
			if err != io.EOF {
				logger.Error("Error when reading http request from ip=%s: [%s]", clientAddrStr, err)
			}
			return
		}
		startTime = time.Now()
		request := newRequest(req, c, startTime)
		request.SetServerAddr(srv.listener.Addr().String())

		var res = srv.handlerExecute(request)

		err = srv.handlerWriteResponse(request, res, c, w)
		if err != nil {
			logger.Error("ERROR writing response: <%T %v>", err, err)
		}
	}
}

func (srv *Server) handlerExecute(request *Request) *http.Response {

	var res *http.Response
	// execute the pipeline
	if res = pipeline.execute(request); res == nil {
		res = StringResponse(request.HttpRequest, 404, nil, "Not Found")
	}
	// cleanup
	request.HttpRequest.Body.Close()
	return res
}

func (srv *Server) handlerWriteResponse(request *Request, res *http.Response, c net.Conn, bw *bufio.Writer) error {
	// cleanup
	defer func() {
		request.finishRequest()
		srv.requestFinished(request, res)
		if res.Body != nil {
			res.Body.Close()
		}
	}()

	// Cycle nodelay flag on socket
	// Note: defers for FILO so this will happen before the write
	// 		 phase is complete, which is what we want.
	if nodelay := srv.setNoDelay(c, false); nodelay {
		defer srv.setNoDelay(c, true)
	}

	var err error
	// Write response
	if err = res.Write(bw); err != nil {
		return err
	}

	// Flush any remaining buffer
	err = bw.Flush()

	return err
}

func (srv *Server) requestFinished(request *Request, res *http.Response) {
	//if srv.CompletionCallback != nil {
		// Don't block the connecion for this
		//go srv.CompletionCallback(request, res)
	//}
}


// Used NoDelay (Nagle's algorithm) where available
func (srv *Server) setNoDelay(c net.Conn, noDelay bool) bool {
	switch runtime.GOOS {
	case "linux", "freebsd", "darwin":
		if tcpC, ok := c.(*net.TCPConn); ok {
			if noDelay {
				// Disable TCP_CORK/TCP_NOPUSH
				tcpC.SetNoDelay(true)
				// For TCP_NOPUSH, we need to force flush
				c.Write([]byte{})
			} else {
				// Re-enable TCP_CORK/TCP_NOPUSH
				tcpC.SetNoDelay(false)
			}
		}
		return true
	default:
		return false
	}
}
