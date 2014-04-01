package boot

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/ParsePlatform/go.grace"
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
