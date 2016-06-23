// Package amqpredialer provides a redialer for github.com/streadway/amqp.Conn.
package amqpredialer

import (
	"io"
	"sync"

	"github.com/cenk/redialer"
	"github.com/streadway/amqp"
)

type amqpDialer struct {
	uri     string
	address string
}

func (d amqpDialer) Dial() (conn io.Closer, err error) {
	return amqp.Dial(d.uri)
}

func (d amqpDialer) Addr() string {
	return d.address
}

type AMQPRedialer struct {
	redialer *redialer.Redialer
	channel  *amqpChannel
}

type amqpChannel struct {
	mu sync.RWMutex

	closed bool
	c      *amqp.Channel
}

func New(uri string) (*AMQPRedialer, error) {
	u, err := amqp.ParseURI(uri)
	if err != nil {
		return nil, err
	}
	u.Password = "XXX" // do not print in logs
	d := amqpDialer{
		uri:     uri,
		address: u.String(),
	}
	ar := &AMQPRedialer{
		redialer: redialer.New(d),
		channel:  &amqpChannel{closed: true},
	}

	return ar, nil
}

func (r *AMQPRedialer) Run() {
	r.redialer.Run()
}

func (r *AMQPRedialer) Close() error {
	return r.redialer.Close()
}

func (r *AMQPRedialer) Conn() <-chan *amqp.Connection {
	ch := make(chan *amqp.Connection, 1)
	go r.notifyConn(ch)
	return ch
}

func (r *AMQPRedialer) notifyConn(ch chan<- *amqp.Connection) {
	rconn, ok := <-r.redialer.Conn()
	if !ok {
		close(ch)
		return
	}
	aconn := rconn.Get().(*amqp.Connection)
	closed := make(chan *amqp.Error, 1)
	aconn.NotifyClose(closed)
	go func() {
		<-closed
		rconn.SetClosed()
	}()
	ch <- aconn
}

func (r *AMQPRedialer) Channel() <-chan *amqp.Channel {
	ch := make(chan *amqp.Channel, 1)
	go r.notifyChannel(ch)
	return ch
}

func (r *AMQPRedialer) notifyChannel(ch chan<- *amqp.Channel) {
	conn, ok := <-r.Conn()
	if !ok {
		close(ch)
		return
	}

	r.channel.mu.Lock()
	if r.channel.closed {
		achan, err := conn.Channel()
		if err != nil {
			close(ch)
			r.channel.mu.Unlock()
			return
		}
		r.channel.c = achan
		r.channel.closed = false
	}
	closed := make(chan *amqp.Error, 1)
	r.channel.c.NotifyClose(closed)
	r.channel.mu.Unlock()

	go func() {
		<-closed
		r.channel.mu.Lock()
		r.channel.closed = true
		r.channel.mu.Unlock()
	}()
	ch <- r.channel.c
}

func (r *AMQPRedialer) CloseChannel() error {
	r.channel.mu.Lock()
	defer r.channel.mu.Unlock()
	if !r.channel.closed {
		return r.channel.c.Close()
	}
	return nil
}

func (r *AMQPRedialer) CancelChannel(consumer string, noWait bool) error {
	r.channel.mu.Lock()
	defer r.channel.mu.Unlock()
	if !r.channel.closed {
		return r.channel.c.Cancel(consumer, noWait)
	}
	return nil
}
