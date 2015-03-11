// Package redialer provides a generic redialer for connection-like types in Go.
// It is useful when you need to access a connection from multiple goroutines.
// It helps to keep the reconnection logic in a single goroutine and provide protected access to the connection.
//
// See netredialer subpackage for usage example.
package redialer

import (
	"io"
	"log"
	"sync"
	"time"
)

// Redialer keeps connections connected.
type Redialer struct {
	dialer Dialer
	conn   io.Closer
	closed bool
	m      sync.Mutex
	cond   sync.Cond
}

type Dialer interface {
	Addr() string // used in logs
	Dial() (conn io.Closer, err error)
}

type Conn struct {
	redialer      *Redialer
	connectedConn io.Closer
}

// Get returns the connected connection.
// You have to convert the returned value to the type your Dial function returns.
func (c *Conn) Get() interface{} {
	return c.connectedConn
}

// SetClosed tells Redialer that the connection is closed.
// You have to call this function after your code detected the connection is disconnected.
func (c *Conn) SetClosed() {
	c.redialer.connClosed(c)
}

func New(d Dialer) *Redialer {
	r := &Redialer{
		dialer: d,
	}
	r.cond.L = &r.m
	return r
}

// Conn sends the connected connection on the returned channel.
// Only one Conn will be sent to the channel.
// If the Redialer is closed, the channel is closed.
func (r *Redialer) Conn() <-chan *Conn {
	ch := make(chan *Conn, 1)
	go r.notifyConn(ch)
	return ch
}

func (r *Redialer) notifyConn(ch chan<- *Conn) {
	r.m.Lock()
	defer r.m.Unlock()
	for r.conn == nil && !r.closed {
		r.cond.Wait()
	}
	if r.closed {
		close(ch)
		return
	}
	ch <- &Conn{
		redialer:      r,
		connectedConn: r.conn,
	}
}

// Close stops the Redialer and closes the connection if it is open.
func (r *Redialer) Close() error {
	r.m.Lock()
	defer r.m.Unlock()
	defer r.cond.Broadcast()
	r.closed = true
	if r.conn == nil {
		return nil
	}
	return r.conn.Close()
}

func (r *Redialer) Run() {
	for {
		r.m.Lock()
		for r.conn != nil && !r.closed {
			r.cond.Wait()
		}
		if r.closed {
			if r.conn != nil {
				r.conn.Close()
			}
			r.m.Unlock()
			break
		}
		r.m.Unlock()

		var conn io.Closer
		for {
			r.m.Lock()
			closed := r.closed
			r.m.Unlock()
			if closed {
				break
			}

			log.Println("connecting to", r.dialer.Addr())
			var err error
			conn, err = r.dialer.Dial()
			if err != nil {
				conn = nil // implementation may return non-nil value on error
				log.Println("cannot connect to", r.dialer.Addr(), "err:", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		log.Println("connected to", r.dialer.Addr())
		r.m.Lock()
		r.conn = conn
		r.m.Unlock()
		r.cond.Broadcast()
	}
}

func (r *Redialer) connClosed(conn *Conn) {
	r.m.Lock()
	defer r.m.Unlock()
	if conn.connectedConn == r.conn {
		r.conn = nil
		log.Println("disconnected from", r.dialer.Addr())
		r.cond.Broadcast()
	}
}
