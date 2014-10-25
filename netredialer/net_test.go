package netredialer_test

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/cenkalti/redialer/netredialer"
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

func Test(t *testing.T) {
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	address := net.JoinHostPort(host, port)
	r := netredialer.New("tcp", address)
	go r.Run()

	var conn net.Conn

	select {
	case conn = <-r.Conn():
		testRead(t, conn)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	conn.Close()

	select {
	case conn = <-r.Conn():
		testRead(t, conn)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
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
