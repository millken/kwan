package boot

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/ParsePlatform/go.grace"
	"log"
	"net"
	"net/http"
	"os"
)

// An App contains one or more servers and associated configuration.
type App struct {
	Servers   []*http.Server
	listeners []grace.Listener
	errors    chan error
}

func NewApp() (a *App) {
	return &App{}
}

func (a *App) AddServer(s *http.Server) {
	a.Servers = append(a.Servers, s)
}

func (a *App) ServeHttp(addr string) {
	if addr == "" {
		return
	}
	ln := a.listen(addr)
	log.Printf("Listening http on [%s]", addr)
	a.serve(ln)
}

func (a *App) listen(addr string) net.Listener {
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		log.Printf("Cannot listen [%s]: [%s]", addr, err)
	}
	return ln
}

func (a *App) serve(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				log.Printf("Cannot accept connections due temporary network error: [%s]", err)
				//time.Sleep(time.Second)
				continue
			}
			log.Fatal("Cannot accept connections due permanent error: [%s]", err)
		}
		go a.handleConnection(conn)
	}
}

func ip4ToUint32(ip4 net.IP) uint32 {
return (uint32(ip4[0]) << 24) | (uint32(ip4[1]) << 16) | (uint32(ip4[2]) << 8) | uint32(ip4[3])
}

func (a *App) handleConnection(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().(*net.TCPAddr).IP.To4()

	r := bufio.NewReaderSize(conn, 1024)
	w := bufio.NewWriterSize(conn, 4096)
	clientAddrStr := clientAddr.String()
	for {
		req, err := http.ReadRequest(r)
		if err != nil {
			if err != io.EOF {
				logMessage("Error when reading http request from ip=%s: [%s]", clientAddr, err)
			}
			return
		}
		req.RemoteAddr = clientAddrStr
		ok := handleRequest(req, w)
		w.Flush()
		if !ok || !req.ProtoAtLeast(1, 1) || req.Header.Get("Connection") == "close" {
			return
		}
	}
}

// Listen will inherit or create new listeners. Returns a bool indicating if we
// inherited listeners. This return value is useful in order to decide if we
// should instruct the parent process to terminate.
func (a *App) Listen() (bool, error) {
	var err error
	a.errors = make(chan error, len(a.Servers))
	a.listeners, err = grace.Inherit()
	if err == nil {
		if len(a.Servers) != len(a.listeners) {
			return true, errors.New("unexpected listeners count")
		}
		return true, nil
	} else if err == grace.ErrNotInheriting {
		if a.listeners, err = a.newListeners(); err != nil {
			return false, err
		}
		return false, nil
	}
	return false, fmt.Errorf("failed graceful handoff: %s", err)
}

// Creates new listeners (as in not inheriting) for all the configured Servers.
func (a *App) newListeners() ([]grace.Listener, error) {
	listeners := make([]grace.Listener, len(a.Servers))
	for index, server := range a.Servers {
		addr, err := net.ResolveTCPAddr("tcp", server.Addr)
		if err != nil {
			return nil, fmt.Errorf("net.ResolveTCPAddr %s: %s", server.Addr, err)
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("net.ListenTCP %s: %s", server.Addr, err)
		}
		listeners[index] = grace.NewListener(l)
	}
	return listeners, nil
}

// Serve the configured servers, but do not block. You must call Wait at some
// point to ensure correctly waiting for graceful termination.
func (a *App) Serve() {
	for i, l := range a.listeners {
		go func(i int, l net.Listener) {
			server := a.Servers[i]

			// Wrap the listener for TLS support if necessary.
			if server.TLSConfig != nil {
				l = tls.NewListener(l, server.TLSConfig)
			}

			err := server.Serve(l)
			// The underlying Accept() will return grace.ErrAlreadyClosed
			// when a signal to do the same is returned, which we are okay with.
			if err != nil && err != grace.ErrAlreadyClosed {
				a.errors <- fmt.Errorf("http.Serve: %s", err)
			}
		}(i, l)
	}
}

// Wait for the serving goroutines to finish.
func (a *App) Wait() error {
	waiterr := make(chan error)
	go func() { waiterr <- grace.Wait(a.listeners) }()
	select {
	case err := <-waiterr:
		return err
	case err := <-a.errors:
		return err
	}
}

func (a *App) Run() error {
	inherited, err := a.Listen()
	if err != nil {
		return err
	}

	a.Serve()

	// Close the parent if we inherited and it wasn't init that started us.
	if inherited && os.Getppid() != 1 {
		if err := grace.CloseParent(); err != nil {
			return fmt.Errorf("failed to close parent: %s", err)
		}
	}

	err = a.Wait()

	return err
}
