package cloud_test

import (
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"testing"
)

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
