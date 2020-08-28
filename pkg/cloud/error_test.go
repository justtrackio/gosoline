package cloud_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/redis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"testing"
)

type awsErr struct {
	error     string
	code      string
	message   string
	origError error
}

func (a awsErr) Error() string {
	return a.error
}

func (a awsErr) Code() string {
	return a.code
}

func (a awsErr) Message() string {
	return a.message
}

func (a awsErr) OrigErr() error {
	return a.origError
}

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
			err:        redis.Nil,
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
		"cloud": {
			err:        cloud.RequestCanceledError,
			isCanceled: true,
		},
		"cloud wrapped": {
			err:        fmt.Errorf("error %w", cloud.RequestCanceledError),
			isCanceled: true,
		},
		"aws": {
			err: awsErr{
				code: request.CanceledErrorCode,
			},
			isCanceled: true,
		},
		"aws wrapped": {
			err: fmt.Errorf("error %w", awsErr{
				code: request.CanceledErrorCode,
			}),
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
			err:        multierror.Append(nil, fmt.Errorf("error %w", context.Canceled), fmt.Errorf("error %w", cloud.RequestCanceledError)),
			isCanceled: true,
		},
		"multierror mixed": {
			err:        multierror.Append(nil, context.Canceled, redis.Nil),
			isCanceled: false,
		},
	} {
		test := test
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.isCanceled, cloud.IsRequestCanceled(test.err))
		})
	}
}

func TestIsUsedClosedConnectionError(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = ln.Close()
	}()

	cfg := &aws.Config{
		Region:      aws.String(endpoints.EuCentral1RegionID),
		Endpoint:    aws.String(ln.Addr().String()),
		Credentials: credentials.NewStaticCredentials("test", "a", "b"),
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				DialTLS: func(network, addr string) (net.Conn, error) {
					conn, err := net.Dial(ln.Addr().Network(), ln.Addr().String())

					if err != nil {
						return nil, err
					}

					// close the connection to reproduce the error
					defer func() {
						_ = conn.Close()
					}()

					return conn, err
				},
			},
		},
	}

	sess := session.Must(session.NewSession(cfg))

	client := kinesis.New(sess)
	_, err = client.ListStreams(&kinesis.ListStreamsInput{})

	isClosedErr := cloud.IsUsedClosedConnectionError(err)

	assert.True(t, isClosedErr, "error: %v", err)
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

	isConnErr := cloud.IsConnectionError(err)
	assert.True(t, isConnErr, "error should be a connection error")
}
