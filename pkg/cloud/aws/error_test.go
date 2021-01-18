package aws_test

import (
	"fmt"
	cloudAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
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

func TestInvalidStatusError(t *testing.T) {
	for name, test := range map[string]struct {
		err                  error
		isInvalidStatusError bool
		errorType            exec.ErrorType
	}{
		"invalid status": {
			err: &cloudAws.InvalidStatusError{
				Status: 400,
			},
			isInvalidStatusError: true,
			errorType:            exec.ErrorTypeRetryable,
		},
		"canceled": {
			err:                  exec.RequestCanceledError,
			isInvalidStatusError: false,
			errorType:            exec.ErrorTypeUnknown,
		},
		"nil": {
			err:                  nil,
			isInvalidStatusError: false,
			errorType:            exec.ErrorTypeUnknown,
		},
	} {
		test := test
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.isInvalidStatusError, cloudAws.IsInvalidStatusError(test.err))
			assert.Equal(t, test.errorType, cloudAws.CheckInvalidStatusError(nil, test.err))
		})
	}
}

func TestIsRequestCanceled(t *testing.T) {
	for name, test := range map[string]struct {
		err        error
		isCanceled bool
	}{
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
	} {
		test := test
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.isCanceled, exec.IsRequestCanceled(test.err))
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

	isClosedErr := exec.IsUsedClosedConnectionError(err)

	assert.True(t, isClosedErr, "error: %v", err)
}
