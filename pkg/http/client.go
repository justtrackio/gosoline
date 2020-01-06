package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"gopkg.in/resty.v1"
	netUrl "net/url"
	"time"
)

const (
	DeleteRequest         = "DELETE"
	GetRequest            = "GET"
	PostRequest           = "POST"
	metricRequest         = "HttpClientRequest"
	metricError           = "HttpClientError"
	metricResponseCode    = "HttpClientResponseCode"
	metricRequestDuration = "HttpRequestDuration"
)

//go:generate mockery -name Client
type Client interface {
	Delete(ctx context.Context, request *Request) (*Response, error)
	Get(ctx context.Context, request *Request) (*Response, error)
	Post(ctx context.Context, request *Request) (*Response, error)
	SetTimeout(timeout time.Duration)
	SetUserAgent(ua string)
	NewRequest() *Request
	NewJsonRequest() *Request
	NewXmlRequest() *Request
}

type Response struct {
	Body            []byte
	StatusCode      int
	RequestDuration time.Duration
}

type headers map[string]string

type client struct {
	mo             mon.MetricWriter
	logger         mon.Logger
	http           *resty.Client
	defaultHeaders headers
}

type Settings struct {
	RetryCount int
	Timeout    time.Duration
}

func NewHttpClient(config cfg.Config, logger mon.Logger) Client {
	mo := mon.NewMetricDaemonWriter()

	retryCount := config.GetInt("http_client_retry_count")
	timeout := config.GetDuration("http_client_request_timeout")

	return NewHttpClientWithInterfaces(logger, mo, &Settings{
		RetryCount: retryCount,
		Timeout:    timeout,
	})
}

func NewHttpClientWithInterfaces(logger mon.Logger, mo mon.MetricWriter, settings *Settings) Client {
	httpClient := resty.New()

	httpClient.SetRetryCount(settings.RetryCount)
	httpClient.SetTimeout(settings.Timeout)

	return &client{
		mo:             mo,
		logger:         logger,
		http:           httpClient,
		defaultHeaders: make(headers),
	}
}

func (c *client) NewRequest() *Request {
	return &Request{
		resty:       c.http.NewRequest(),
		url:         &netUrl.URL{},
		queryParams: netUrl.Values{},
	}
}

func (c *client) NewJsonRequest() *Request {
	return c.NewRequest().WithHeader(HdrAccept, ContentTypeApplicationJson)
}

func (c *client) NewXmlRequest() *Request {
	return c.NewRequest().WithHeader(HdrAccept, ContentTypeApplicationXml)
}

func (c *client) SetTimeout(timeout time.Duration) {
	c.http.SetTimeout(timeout)
}

func (c *client) SetUserAgent(ua string) {
	c.defaultHeaders[HdrUserAgent] = ua
}

func (c *client) Delete(ctx context.Context, request *Request) (*Response, error) {
	return c.do(ctx, DeleteRequest, request)
}

func (c *client) Get(ctx context.Context, request *Request) (*Response, error) {
	return c.do(ctx, GetRequest, request)
}

func (c *client) Post(ctx context.Context, request *Request) (*Response, error) {
	return c.do(ctx, PostRequest, request)
}

func (c *client) do(ctx context.Context, method string, request *Request) (*Response, error) {
	req, url, err := request.build()
	logger := c.logger.WithContext(ctx).WithFields(mon.Fields{
		"url":    url,
		"method": method,
	})

	if err != nil {
		logger.Error(err, "failed to assemble request")
		return nil, fmt.Errorf("failed to assemble request: %w", err)
	}

	req.SetContext(ctx)
	req.SetHeaders(c.defaultHeaders)

	c.writeMetric(metricRequest, method, mon.UnitCount, 1.0)
	resp, err := req.Execute(method, url)

	if errors.Is(err, context.Canceled) {
		return nil, err
	}

	// Only log an error if the error was not caused by a canceled context
	// Otherwise a user might spam our error logs by just canceling a lot of requests
	// (or many users spam us because sometimes they cancel requests)
	if err != nil {
		c.writeMetric(metricError, method, mon.UnitCount, 1.0)
		return nil, fmt.Errorf("failed to perform %s request to %s: %w", request.resty.Method, request.url.String(), err)
	}

	metricName := fmt.Sprintf("%s%dXX", metricResponseCode, resp.StatusCode()/100)
	c.writeMetric(metricName, method, mon.UnitCount, 1.0)

	response := &Response{
		Body:            resp.Body(),
		StatusCode:      resp.StatusCode(),
		RequestDuration: resp.Time(),
	}

	// Only log the duration if we did not get an error.
	// If we get an error, we might not actually have send anything,
	// so the duration will be very low. If we get back an error (e.g., status 500),
	// we log the duration as this is just a valid http response.
	requestDurationMs := float64(resp.Time() / time.Millisecond)
	c.writeMetric(metricRequestDuration, method, mon.UnitMilliseconds, requestDurationMs)

	return response, nil
}

func (c *client) writeMetric(metricName string, method string, unit string, value float64) {
	c.mo.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: mon.MetricDimensions{
			"Method": method,
		},
		Unit:  unit,
		Value: value,
	})
}
