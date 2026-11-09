package dynamodb_test

import (
	"context"
	"errors"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	awsDynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	smithyHttp "github.com/aws/smithy-go/transport/http"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientDrainsClosedRequestBody(t *testing.T) {
	var requestBody io.ReadCloser
	responseBody := `{"Table":{"TableName":"test"}}`
	httpClient := smithyHttp.ClientDoFunc(func(request *http.Request) (*http.Response, error) {
		requestBody = request.Body
		body, err := io.ReadAll(requestBody)
		require.NoError(t, err)
		assert.JSONEq(t, `{"TableName":"test"}`, string(body))

		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"X-Amz-Crc32": {strconv.FormatUint(uint64(crc32.ChecksumIEEE([]byte(responseBody))), 10)},
			},
			Body: io.NopCloser(strings.NewReader(responseBody)),
		}, nil
	})
	client := newClient(t, func(config *dynamodb.ClientConfig) {
		config.LoadOptions = append(config.LoadOptions, awsCfg.WithHTTPClient(httpClient))
	})

	output, err := client.DescribeTable(t.Context(), &awsDynamodb.DescribeTableInput{TableName: aws.String("test")})
	require.NoError(t, err)
	require.NotNil(t, output.Table)
	assert.Equal(t, "test", aws.ToString(output.Table.TableName))

	// net/http can finish draining the request after Smithy has closed its body.
	written, err := io.Copy(io.Discard, requestBody)
	assert.NoError(t, err)
	assert.Zero(t, written)
}

func TestNewClientPreservesHttpRequest(t *testing.T) {
	for _, name := range []string{"nil body", "empty body", "body"} {
		t.Run(name, func(t *testing.T) {
			body := &trackedBody{Reader: strings.NewReader("request")}
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://localhost", http.NoBody)
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/json")
			switch name {
			case "nil body":
				request.Body = nil
			case "body":
				request.Body = body
				request.ContentLength = int64(body.Len())
			}
			originalBody := request.Body
			response := &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			httpClient := smithyHttp.ClientDoFunc(func(sent *http.Request) (*http.Response, error) {
				assert.Same(t, request.Context(), sent.Context())
				assert.Equal(t, request.Method, sent.Method)
				assert.Equal(t, request.URL, sent.URL)
				assert.Equal(t, request.Header, sent.Header)
				assert.Equal(t, request.ContentLength, sent.ContentLength)
				if name == "body" {
					data, err := io.ReadAll(sent.Body)
					require.NoError(t, err)
					assert.Equal(t, "request", string(data))
					require.NoError(t, sent.Body.Close())
					assert.True(t, body.closed)
				} else {
					assert.Equal(t, originalBody, sent.Body)
				}

				return response, nil
			})
			client := newClient(t, func(config *dynamodb.ClientConfig) {
				config.LoadOptions = append(config.LoadOptions, awsCfg.WithHTTPClient(httpClient))
			})

			actual, err := client.Options().HTTPClient.Do(request)
			require.NoError(t, err)
			assert.Same(t, response, actual)
			assert.Equal(t, originalBody, request.Body)
		})
	}
}

func TestNewClientPreservesHttpError(t *testing.T) {
	expected := errors.New("request failed")
	httpClient := smithyHttp.ClientDoFunc(func(*http.Request) (*http.Response, error) {
		return nil, expected
	})
	client := newClient(t, func(config *dynamodb.ClientConfig) {
		config.LoadOptions = append(config.LoadOptions, awsCfg.WithHTTPClient(httpClient))
	})
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://localhost", strings.NewReader("request"))
	require.NoError(t, err)

	response, err := client.Options().HTTPClient.Do(request)
	assert.Same(t, expected, err)
	assert.Nil(t, response)
}

func TestNewClientPreservesHttpTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	t.Cleanup(server.Close)
	client := newClient(t, func(config *dynamodb.ClientConfig) {
		config.Settings.HttpClient.Timeout = 50 * time.Millisecond
	})
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	response, err := client.Options().HTTPClient.Do(request)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, response)
}

type trackedBody struct {
	*strings.Reader
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true

	return nil
}

func newClient(t *testing.T, options ...dynamodb.ClientOption) *awsDynamodb.Client {
	t.Helper()
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"cloud.aws.defaults.credentials.access_key_id":     "test",
		"cloud.aws.defaults.credentials.secret_access_key": "test",
	})))
	client, err := dynamodb.NewClient(t.Context(), config, log.NewLogger(), "default", options...)
	require.NoError(t, err)

	return client
}
