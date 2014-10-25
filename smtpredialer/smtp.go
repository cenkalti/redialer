package smtpredialer

import (
	"io"
	"net/smtp"

	"github.com/cenkalti/redialer"
)

type smtpDialer struct {
	Address string
}

func (d smtpDialer) Dial() (conn io.Closer, err error) {
	return smtp.Dial(d.Address)
}

func (d smtpDialer) Addr() string {
	return d.Address
}

type SMTPRedialer struct {
	*redialer.Redialer
}

func New(address string) *SMTPRedialer {
	d := &smtpDialer{address}
	return &SMTPRedialer{redialer.New(d)}
}

func (r *SMTPRedialer) Conn() <-chan *Client {
	ch := make(chan *Client, 1)
	go r.notifyConn(ch)
	return ch
}

func (r *SMTPRedialer) notifyConn(ch chan<- *Client) {
	rconn, ok := <-r.Redialer.Conn()
	if !ok {
		close(ch)
		return
	}
	client := rconn.Get().(*smtp.Client)
	ch <- &Client{client, rconn}
}

type Client struct {
	*smtp.Client
	rconn *redialer.Conn
}

func (c *Client) Close() error {
	err := c.Client.Close()
	c.rconn.SetClosed()
	return err
}
