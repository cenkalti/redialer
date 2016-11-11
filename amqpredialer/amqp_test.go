package amqpredialer_test

import (
	"log"
	"testing"
	"time"

	"github.com/cenkalti/redialer/amqpredialer"
)

func TestClose(t *testing.T) {
	addr := "amqp://127.0.0.1:0" // unreachable address
	r, err := amqpredialer.New(addr)
	if err != nil {
		log.Fatal(err)
	}
	finish := make(chan struct{})
	go func() {
		r.Run()
		close(finish)
	}()
	time.Sleep(time.Second)
	r.Close() // must not panic
	<-finish
}
