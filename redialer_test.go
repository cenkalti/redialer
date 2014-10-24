package redialer_test

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/cenkalti/redialer"
)

const host = "localhost"

var l net.Listener

func init() {
	var err error
	l, err = net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}
			go func(conn net.Conn) {
				for {
					if _, err := io.Copy(ioutil.Discard, conn); err != nil {
						return
					}
				}
			}(conn)
			go func(conn net.Conn) {
				for {
					if _, err := conn.Write([]byte{1}); err != nil {
						return
					}
				}
			}(conn)
		}
	}()
}

type tcpDialer struct{}

func (d *tcpDialer) Dial() (conn io.Closer, err error) {
	return net.Dial("tcp", d.Addr())
}

func (d *tcpDialer) Addr() string {
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	return net.JoinHostPort(host, port)
}

func Test(t *testing.T) {
	d := &tcpDialer{}
	r := redialer.New(d)
	go r.Run()

	var conn *redialer.Conn
	var netConn net.Conn

	select {
	case conn = <-r.Conn():
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	netConn = conn.Get().(net.Conn)
	testRead(t, netConn)

	netConn.Close()
	conn.SetClosed()

	select {
	case conn = <-r.Conn():
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	netConn = conn.Get().(net.Conn)
	testRead(t, netConn)
}

func testRead(t *testing.T, conn net.Conn) {
	r := bufio.NewReader(conn)
	b, err := r.ReadByte()
	if err != nil {
		t.Fatal(err)
	}
	if b != 1 {
		t.Fatalf("wrong data: %d", b)
	}
}
