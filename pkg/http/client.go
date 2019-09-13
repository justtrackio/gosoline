package http

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"gopkg.in/resty.v1"
	"net/http"
	netUrl "net/url"
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
		if netErr, ok := err.(*netUrl.Error); ok && netErr.Err == context.Canceled {
			// Unwrap the error so our callers can simply check if the request was canceled and
			// react accordingly. The caller can not check for this upfront as the request could
			// be canceled while we wait for an answer of the server, causing this error to get
			// thrown.
			err = context.Canceled
		} else {
			// Only log an error if the error was not caused by a canceled context
			// Otherwise a user might spam our error logs by just canceling a lot of requests
			// (or many users spam us because sometimes they cancel requests)
			logger.Error(err, "error while requesting url")

			c.writeMetric(metricRequestFailure, method, mon.UnitCount, 1.0)
		}
	}

	responseStatusCode := resp.StatusCode()
	responseStatusCodeString := strconv.Itoa(responseStatusCode)

	if err == nil && (responseStatusCode < http.StatusOK || responseStatusCode >= http.StatusMultipleChoices) {
		// We should not log this if we got an error - in that case the response status code could be 0
		// and that does not make for a good log message
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

	if err == nil {
		// Only log the duration if we did not get an error.
		// If we get an error, we might not actually have send anything,
		// so the duration will be very low. If we get back an error (e.g., status 500),
		// we log the duration as this is just a valid http response.
		requestDurationMs := float64(response.RequestDuration / time.Millisecond)
		c.writeMetric(metricRequestDuration, method, mon.UnitMilliseconds, requestDurationMs)
	}

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
