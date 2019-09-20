package http

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"gopkg.in/resty.v1"
	netUrl "net/url"
	"time"
)

const (
	GetRequest            = "GET"
	PostRequest           = "POST"
	metricRequest         = "HttpClientRequest"
	metricError           = "HttpClientError"
	metricResponseCode    = "HttpClientResponseCode"
	metricRequestDuration = "HttpRequestDuration"
)

//go:generate mockery -name Client
type Client interface {
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
	config         cfg.Config
}

func NewHttpClient(config cfg.Config, logger mon.Logger) Client {
	mo := mon.NewMetricDaemonWriter()

	return NewHttpClientWithInterfaces(config, logger, mo)
}

func NewHttpClientWithInterfaces(config cfg.Config, logger mon.Logger, mo mon.MetricWriter) Client {
	httpClient := resty.New()

	retryCount := config.GetInt("http_client_retry_count")
	timeout := config.GetDuration("http_client_request_timeout")

	httpClient.SetRetryCount(retryCount)
	httpClient.SetTimeout(time.Second * timeout)

	return &client{
		mo:             mo,
		logger:         logger,
		http:           httpClient,
		defaultHeaders: make(headers),
		config:         config,
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
		return nil, err
	}

	req.SetContext(ctx)
	req.SetHeaders(c.defaultHeaders)

	c.writeMetric(metricRequest, method, mon.UnitCount, 1.0)
	resp, err := req.Execute(method, url)

	// Unwrap the error so our callers can simply check if the request was canceled and
	// react accordingly. The caller can not check for this upfront as the request could
	// be canceled while we wait for an answer of the server, causing this error to get
	// thrown.
	err, canceled := isContextCanceled(ctx, err)

	metricName := fmt.Sprintf("%s%dXX", metricResponseCode, resp.StatusCode()/100)
	c.writeMetric(metricName, method, mon.UnitCount, 1.0)

	if err != nil {
		if !canceled {
			// Only log an error if the error was not caused by a canceled context
			// Otherwise a user might spam our error logs by just canceling a lot of requests
			// (or many users spam us because sometimes they cancel requests)
			c.writeMetric(metricError, method, mon.UnitCount, 1.0)
			logger.Error(err, "error while requesting url")
		}

		return nil, err
	}

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

	return response, err
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

func isContextCanceled(ctx context.Context, err error) (error, bool) {
	if err == nil {
		return err, false
	}

	// TODO: with go 1.13, we could check for errors.Is(err, context.Canceled),
	// doing all the unwrapping for us. For now we can not really safely and in
	// a nice way unwrap the error, so lets just check the context - if it is
	// canceled, we most likely also hit that error and can just pretend we did.
	if ctx.Err() == context.Canceled {
		return context.Canceled, true
	}

	return err, false
}
