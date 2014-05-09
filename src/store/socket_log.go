package store

import (
	"net"
	"time"
)

type SocketHandler struct {
	c        net.Conn
	protocol string
	addr     string
}

func NewSocketHandler(protocol string, addr string) (*SocketHandler) {
	s := new(SocketHandler)

	s.protocol = protocol
	s.addr = addr

	return s
}

func (h *SocketHandler) Write(p string) ( err error) {
	if err = h.connect(); err != nil {
		return 
	}

	buf := []byte(p)
	_, err = h.c.Write(buf)
	if err != nil {
		h.c = nil
	}
	return
}

func (h *SocketHandler) Close() error {
	if h.c != nil {
		h.c.Close()
	}
	return nil
}

func (h *SocketHandler) connect() error {
	if h.c != nil {
		return nil
	}

	var err error
	h.c, err = net.DialTimeout(h.protocol, h.addr, 5*time.Second)
	if err != nil {
		return err
	}
	t := time.Unix(0,0)
	h.c.SetWriteDeadline(t)
	h.c.SetDeadline(t)

	return nil
}
