package netredialer

import (
	"io"
	"net"

	"github.com/cenkalti/redialer"
)

type netDialer struct {
	Network string
	Address string
}

func (d netDialer) Dial() (conn io.Closer, err error) {
	return net.Dial(d.Network, d.Address)
}

func (d netDialer) Addr() string {
	return d.Network + "://" + d.Address
}

type NetRedialer struct {
	*redialer.Redialer
}

func New(network, address string) *NetRedialer {
	d := &netDialer{network, address}
	return &NetRedialer{redialer.New(d)}
}

func (r *NetRedialer) Conn() <-chan net.Conn {
	ch := make(chan net.Conn, 1)
	go r.notifyConn(ch)
	return ch
}

func (r *NetRedialer) notifyConn(ch chan<- net.Conn) {
	rconn, ok := <-r.Redialer.Conn()
	if !ok {
		close(ch)
		return
	}
	ch <- rconn.Get().(net.Conn)
}

type conn struct {
	net.Conn
	rc *redialer.Conn
}

func (c *conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if err != nil {
		c.rc.SetClosed()
	}
	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if err != nil {
		c.rc.SetClosed()
	}
	return
}
