package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	netUrl "net/url"
	"time"

	httpHeaders "github.com/go-http-utils/headers"
	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	DeleteRequest         = "DELETE"
	GetRequest            = "GET"
	PostRequest           = "POST"
	PutRequest            = "PUT"
	PatchRequest          = "PATCH"
	OptionsRequest        = "OPTIONS"
	metricRequest         = "HttpClientRequest"
	metricError           = "HttpClientError"
	metricResponseCode    = "HttpClientResponseCode"
	metricRequestDuration = "HttpRequestDuration"
)

//go:generate mockery --name Client
type Client interface {
	Delete(ctx context.Context, request *Request) (*Response, error)
	Get(ctx context.Context, request *Request) (*Response, error)
	Patch(ctx context.Context, request *Request) (*Response, error)
	Post(ctx context.Context, request *Request) (*Response, error)
	Put(ctx context.Context, request *Request) (*Response, error)
	SetTimeout(timeout time.Duration)
	SetUserAgent(ua string)
	SetProxyUrl(p string)
	SetCookies(cs []*http.Cookie)
	SetCookie(c *http.Cookie)
	SetRedirectValidator(allowRequest func(request *http.Request) bool)
	AddRetryCondition(f RetryConditionFunc)
	NewRequest() *Request
	NewJsonRequest() *Request
	NewXmlRequest() *Request
}

type RetryConditionFunc func(*Response, error) bool

type Response struct {
	Body            []byte
	Header          http.Header
	Cookies         []*http.Cookie
	StatusCode      int
	RequestDuration time.Duration
	TotalDuration   *time.Duration
}

type headers map[string]string

type client struct {
	logger         log.Logger
	clock          clock.Clock
	defaultHeaders headers
	http           restyClient
	mo             metric.Writer
}

type Settings struct {
	RequestTimeout   time.Duration `cfg:"request_timeout" default:"30s"`
	RetryCount       int           `cfg:"retry_count" default:"5"`
	RetryWaitTime    time.Duration `cfg:"retry_wait_time" default:"100ms"`
	RetryMaxWaitTime time.Duration `cfg:"retry_max_wait_time" default:"2000ms"`
	FollowRedirects  bool          `cfg:"follow_redirects" default:"true"`
}

func NewHttpClient(config cfg.Config, logger log.Logger) Client {
	c := clock.NewRealClock()

	mo := metric.NewWriter()

	settings := &Settings{}
	config.UnmarshalKey("http_client", settings)

	// allow the old settings to be used for now... we will eventually remove this though
	if config.IsSet("http_client_retry_count") {
		settings.RetryCount = config.GetInt("http_client_retry_count")
		logger.Warn("http_client_retry_count is deprecated, use http_client.retry_count instead")
	}
	if config.IsSet("http_client_request_timeout") {
		settings.RequestTimeout = config.GetDuration("http_client_request_timeout")
		logger.Warn("http_client_request_timeout is deprecated, use http_client.request_timeout instead")
	}

	httpClient := resty.New()
	if settings.FollowRedirects {
		httpClient.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))
	} else {
		httpClient.SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}))
	}
	httpClient.SetRetryCount(settings.RetryCount)
	httpClient.SetTimeout(settings.RequestTimeout)
	httpClient.SetRetryWaitTime(settings.RetryWaitTime)
	httpClient.SetRetryMaxWaitTime(settings.RetryMaxWaitTime)

	return NewHttpClientWithInterfaces(logger, c, mo, httpClient)
}

func NewHttpClientWithInterfaces(logger log.Logger, c clock.Clock, mo metric.Writer, httpClient restyClient) Client {
	return &client{
		logger:         logger,
		clock:          c,
		defaultHeaders: make(headers),
		http:           httpClient,
		mo:             mo,
	}
}

func (c *client) NewRequest() *Request {
	return &Request{
		queryParams:  netUrl.Values{},
		restyRequest: c.http.NewRequest(),
		url:          &netUrl.URL{},
	}
}

func (c *client) NewJsonRequest() *Request {
	return c.NewRequest().WithHeader(httpHeaders.Accept, MimeTypeApplicationJson)
}

func (c *client) NewXmlRequest() *Request {
	return c.NewRequest().WithHeader(httpHeaders.Accept, MimeTypeApplicationXml)
}

func (c *client) SetCookie(hc *http.Cookie) {
	c.http.SetCookie(hc)
}

func (c *client) SetCookies(cs []*http.Cookie) {
	c.http.SetCookies(cs)
}

func (c *client) SetTimeout(timeout time.Duration) {
	c.http.SetTimeout(timeout)
}

func (c *client) SetUserAgent(ua string) {
	c.defaultHeaders[httpHeaders.UserAgent] = ua
}

func (c *client) SetProxyUrl(p string) {
	c.http.SetProxy(p)
}

func (c *client) AddRetryCondition(f RetryConditionFunc) {
	c.http.AddRetryCondition(func(r *resty.Response, e error) bool {
		conditionResult := f(buildResponse(r, nil), e)
		if conditionResult {
			c.logger.Warn("retry attempt %d for request %s", r.Request.Attempt, r.Request.URL)
		}

		return conditionResult
	})
}

func (c *client) SetRedirectValidator(allowRequest func(request *http.Request) bool) {
	c.http.SetRedirectPolicy(
		resty.FlexibleRedirectPolicy(10),
		resty.RedirectPolicyFunc(func(request *http.Request, _ []*http.Request) error {
			if !allowRequest(request) {
				return http.ErrUseLastResponse
			}

			return nil
		}),
	)
}

func (c *client) Delete(ctx context.Context, request *Request) (*Response, error) {
	return c.do(ctx, DeleteRequest, request)
}

func (c *client) Get(ctx context.Context, request *Request) (*Response, error) {
	return c.do(ctx, GetRequest, request)
}

func (c *client) Patch(ctx context.Context, request *Request) (*Response, error) {
	return c.do(ctx, PatchRequest, request)
}

func (c *client) Post(ctx context.Context, request *Request) (*Response, error) {
	return c.do(ctx, PostRequest, request)
}

func (c *client) Put(ctx context.Context, request *Request) (*Response, error) {
	return c.do(ctx, PutRequest, request)
}

func (c *client) do(ctx context.Context, method string, request *Request) (*Response, error) {
	req, url, err := request.build()
	logger := c.logger.WithContext(ctx).WithFields(log.Fields{
		"url":    url,
		"method": method,
	})

	if err != nil {
		logger.Error("failed to assemble request: %w", err)
		return nil, fmt.Errorf("failed to assemble request: %w", err)
	}

	req.SetContext(ctx)
	req.SetHeaders(c.defaultHeaders)

	if request.outputFile != nil {
		req.SetOutput(*request.outputFile)
	}

	c.writeMetric(metricRequest, method, metric.UnitCount, 1.0)
	start := c.clock.Now()
	resp, err := req.Execute(method, url)

	if errors.Is(err, context.Canceled) {
		return nil, err
	}

	totalDuration := c.clock.Now().Sub(start)

	// Only log an error if the error was not caused by a canceled context
	// Otherwise a user might spam our error logs by just canceling a lot of requests
	// (or many users spam us because sometimes they cancel requests)
	if err != nil {
		c.writeMetric(metricError, method, metric.UnitCount, 1.0)
		return nil, fmt.Errorf("failed to perform %s request to %s: %w", request.restyRequest.Method, request.url.String(), err)
	}

	metricName := fmt.Sprintf("%s%dXX", metricResponseCode, resp.StatusCode()/100)
	c.writeMetric(metricName, method, metric.UnitCount, 1.0)

	response := buildResponse(resp, &totalDuration)

	// Only log the duration if we did not get an error.
	// If we get an error, we might not actually have send anything,
	// so the duration will be very low. If we get back an error (e.g., status 500),
	// we log the duration as this is just a valid http response.
	requestDurationMs := float64(resp.Time() / time.Millisecond)
	c.writeMetric(metricRequestDuration, method, metric.UnitMillisecondsAverage, requestDurationMs)

	return response, nil
}

func (c *client) writeMetric(metricName string, method string, unit metric.StandardUnit, value float64) {
	c.mo.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: metric.Dimensions{
			"Method": method,
		},
		Unit:  unit,
		Value: value,
	})
}

func buildResponse(resp *resty.Response, totalDuration *time.Duration) *Response {
	if resp == nil {
		return nil
	}

	return &Response{
		Body:            resp.Body(),
		Cookies:         resp.Cookies(),
		Header:          resp.Header(),
		RequestDuration: resp.Time(),
		StatusCode:      resp.StatusCode(),
		TotalDuration:   totalDuration,
	}
}
