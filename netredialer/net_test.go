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

func Example() {
	r := netredialer.New("tcp", "localhost:5000")
	go r.Run()

	// Each goroutine uses the same NetRedialer.
	worker := func(id int) {
		for {
			// Wait until the connection is established.
			conn := <-r.Conn()

			// Use the connection.
			_, err := conn.Write([]byte("message from " + strconv.Itoa(id)))
			if err != nil {
				// Don't forget to call Close, it signals Redialer to reconnect.
				conn.Close()
			}
		}
	}

	for i := 0; i < 5; i++ {
		go worker(i)
	}

	select {}
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
