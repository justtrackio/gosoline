package exec_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"testing"
)

func TestIsRequestCanceled(t *testing.T) {
	for name, test := range map[string]struct {
		err        error
		isCanceled bool
	}{
		"none": {
			err:        nil,
			isCanceled: false,
		},
		"other error": {
			err:        io.EOF,
			isCanceled: false,
		},
		"format error": {
			err:        fmt.Errorf("error: %d", 42),
			isCanceled: false,
		},
		"simple": {
			err:        context.Canceled,
			isCanceled: true,
		},
		"simple wrapped": {
			err:        fmt.Errorf("error %w", context.Canceled),
			isCanceled: true,
		},
		"exec": {
			err:        exec.RequestCanceledError,
			isCanceled: true,
		},
		"exec wrapped": {
			err:        fmt.Errorf("error %w", exec.RequestCanceledError),
			isCanceled: true,
		},
		"multierror empty": {
			err:        multierror.Append(nil),
			isCanceled: false,
		},
		"multierror single": {
			err:        multierror.Append(nil, context.Canceled),
			isCanceled: true,
		},
		"multierror single wrapped": {
			err:        multierror.Append(nil, fmt.Errorf("error %w", context.Canceled)),
			isCanceled: true,
		},
		"multierror multiple wrapped": {
			err:        multierror.Append(nil, fmt.Errorf("error %w", context.Canceled), fmt.Errorf("error %w", exec.RequestCanceledError)),
			isCanceled: true,
		},
		"multierror mixed": {
			err:        multierror.Append(nil, context.Canceled, io.EOF),
			isCanceled: false,
		},
	} {
		test := test
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.isCanceled, exec.IsRequestCanceled(test.err))
		})
	}
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
