package http

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"gopkg.in/resty.v1"
	"net/http"
	"strconv"
	"time"
)

const GetRequest = "GET"
const PostRequest = "POST"
const metricRequest = "HttpClientRequest"
const metricRequestFailure = "HttpClientRequestFailure"
const metricRequestDuration = "HttpRequestDuration"

//go:generate mockery -name Client
type Client interface {
	Get(ctx context.Context, request *Request) (*Response, error)
	Post(ctx context.Context, request *Request) (*Response, error)
	SetTimeout(timeout time.Duration)
	SetUserAgent(ua string)
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
	logger := c.logger.
		WithContext(ctx).
		WithFields(mon.Fields{
			"url":    url,
			"method": method,
		})

	if err != nil {
		logger.Error(err, "failed to assemble request")

		return nil, err
	}

	req.SetContext(ctx)
	req.SetHeaders(c.defaultHeaders)

	logger.Debug("Requesting URL")

	c.writeMetric(metricRequest, method, mon.UnitCount, 1.0)

	resp, err := req.Execute(method, url)

	logger = logger.WithFields(mon.Fields{
		"url": req.URL,
	})

	if err != nil {
		logger.Error(err, "error while requesting url")

		c.writeMetric(metricRequestFailure, method, mon.UnitCount, 1.0)
	}

	responseStatusCode := resp.StatusCode()
	responseStatusCodeString := strconv.Itoa(responseStatusCode)

	if responseStatusCode < http.StatusOK || responseStatusCode >= http.StatusMultipleChoices {
		logger.WithFields(mon.Fields{
			"statusCode": responseStatusCodeString,
		}).Warn("got unexpected response")

		c.writeMetric(metricRequestFailure+responseStatusCodeString, method, mon.UnitCount, 1.0)
	}

	response := &Response{
		Body:            resp.Body(),
		StatusCode:      resp.StatusCode(),
		RequestDuration: resp.Time(),
	}

	requestDurationMs := float64(response.RequestDuration / time.Millisecond)
	c.writeMetric(metricRequestDuration, method, mon.UnitMilliseconds, requestDurationMs)

	return response, err
}

func (c *client) writeMetric(metricName string, dimensionType string, unit string, value float64) {
	c.mo.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: mon.MetricDimensions{
			"Type":        dimensionType,
			"Application": c.config.GetString("app_name"),
		},
		Unit:  unit,
		Value: value,
	})
}
