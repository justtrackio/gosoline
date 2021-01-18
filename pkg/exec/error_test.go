package exec_test

import (
	"errors"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestIsMaxElapsedTimeError(t *testing.T) {
	err := exec.NewMaxElapsedTimeError(time.Second, time.Minute, errors.New("something went sideways"))

	assert.True(t, exec.IsMaxElapsedTimeError(err))
	assert.Equal(t, "can not retry as the elapsed time 1m0s is greater than the configured max of 1s: something went sideways", err.Error())
	assert.False(t, exec.IsMaxElapsedTimeError(err.Unwrap()))
	assert.EqualError(t, err.Unwrap(), "something went sideways")
}

func TestIsConnectionResetError(t *testing.T) {
	var err error
	var conn net.Conn
	var listener net.Listener
	var waitListen = make(chan struct{})
	var waitClose = make(chan struct{})

	go func() {
		var err error
		var conn net.Conn

		if listener, err = net.Listen("tcp", "localhost:0"); err != nil {
			assert.FailNow(t, err.Error(), "can not create listener")
			return
		}

		close(waitListen)

		if conn, err = listener.Accept(); err != nil {
			assert.FailNow(t, err.Error(), "can not accept connection")
			return
		}

		if err = conn.(*net.TCPConn).SetLinger(0); err != nil {
			assert.FailNow(t, err.Error(), "can not set linger value")
			return
		}

		<-waitClose

		if err = conn.Close(); err != nil {
			assert.FailNow(t, err.Error(), "can not close connection")
		}
	}()

	<-waitListen
	addr := listener.Addr().String()

	if conn, err = net.Dial("tcp", addr); err != nil {
		assert.FailNow(t, err.Error(), "can not connect")
		return
	}

	close(waitClose)

	// Block until close.
	buf := make([]byte, 1)
	_, err = conn.Read(buf)

	isConnErr := exec.IsConnectionError(err)
	assert.True(t, isConnErr, "error should be a connection error")
}
