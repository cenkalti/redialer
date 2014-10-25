package amqpredialer

import (
	"io"

	"github.com/cenkalti/redialer"
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
	*redialer.Redialer
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
	return &AMQPRedialer{
		redialer.New(d),
	}, nil
}

func (r *AMQPRedialer) Conn() <-chan *amqp.Connection {
	ch := make(chan *amqp.Connection, 1)
	go r.notifyConn(ch)
	return ch
}

func (r *AMQPRedialer) notifyConn(ch chan<- *amqp.Connection) {
	rconn, ok := <-r.Redialer.Conn()
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
