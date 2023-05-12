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
	"github.com/justtrackio/gosoline/pkg/appctx"
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
	metricWriter   metric.Writer
}

type Settings struct {
	DisableCookies         bool                   `cfg:"disable_cookies" default:"false"`
	FollowRedirects        bool                   `cfg:"follow_redirects" default:"true"`
	RequestTimeout         time.Duration          `cfg:"request_timeout" default:"30s"`
	RetryCount             int                    `cfg:"retry_count" default:"5"`
	RetryMaxWaitTime       time.Duration          `cfg:"retry_max_wait_time" default:"2000ms"`
	RetryResetReaders      bool                   `cfg:"retry_reset_readers" default:"true"`
	RetryWaitTime          time.Duration          `cfg:"retry_wait_time" default:"100ms"`
	CircuitBreakerSettings CircuitBreakerSettings `cfg:"circuit_breaker"`
}

func ProvideHttpClient(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Client, error) {
	type httpClientName string

	return appctx.Provide(ctx, httpClientName(name), func() (Client, error) {
		return newHttpClient(config, logger, name), nil
	})
}

func newHttpClient(config cfg.Config, logger log.Logger, name string) Client {
	metricWriter := metric.NewWriter()
	settings := UnmarshalClientSettings(config, name)

	var httpClient *resty.Client
	if settings.DisableCookies {
		httpClient = resty.NewWithClient(&http.Client{})
	} else {
		httpClient = resty.New()
	}

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
	httpClient.SetRetryResetReaders(settings.RetryResetReaders)

	client := NewHttpClientWithInterfaces(logger, clock.Provider, metricWriter, httpClient)

	if settings.CircuitBreakerSettings.Enabled {
		client = NewCircuitBreakerClientWithInterfaces(client, logger, clock.Provider, name, settings.CircuitBreakerSettings)
	}

	return client
}

func NewHttpClientWithInterfaces(logger log.Logger, clock clock.Clock, metricWriter metric.Writer, httpClient restyClient) Client {
	return &client{
		logger:         logger,
		clock:          clock,
		defaultHeaders: make(headers),
		http:           httpClient,
		metricWriter:   metricWriter,
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
	// If we get an error, we might not actually have sent anything,
	// so the duration will be very low. If we get back an error (e.g., status 500),
	// we log the duration as this is just a valid http response.
	requestDurationMs := float64(resp.Time() / time.Millisecond)
	c.writeMetric(metricRequestDuration, method, metric.UnitMillisecondsAverage, requestDurationMs)

	return response, nil
}

func (c *client) writeMetric(metricName string, method string, unit metric.StandardUnit, value float64) {
	c.metricWriter.WriteOne(&metric.Datum{
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

func GetClientConfigKey(name string) string {
	return fmt.Sprintf("http_client.%s", name)
}

func UnmarshalClientSettings(config cfg.Config, name string) Settings {
	// notify about wrong old settings being used
	if config.IsSet("http_client_retry_count") {
		panic("'http_client_retry_count' was removed, use 'http_client.default.retry_count' instead")
	}
	if config.IsSet("http_client_request_timeout") {
		panic("'http_client_request_timeout' was removed, use 'http_client.default.request_timeout' instead")
	}
	if config.IsSet("http_client.follow_redirects") {
		panic("'http_client.follow_redirects' was removed, use 'http_client.default.follow_redirects' instead")
	}
	if config.IsSet("http_client.request_timeout") {
		panic("'http_client.request_timeout' was removed, use 'http_client.default.request_timeout' instead")
	}
	if config.IsSet("http_client.retry_count") {
		panic("'http_client.retry_count' was removed, use 'http_client.default.retry_count' instead")
	}
	if config.IsSet("http_client.retry_max_wait_time") {
		panic("'http_client.retry_max_wait_time' was removed, use 'http_client.default.retry_max_wait_time' instead")
	}
	if config.IsSet("http_client.retry_reset_readers") {
		panic("'http_client.retry_reset_readers' was removed, use 'http_client.default.retry_reset_readers' instead")
	}
	if config.IsSet("http_client.retry_wait_time") {
		panic("'http_client.retry_wait_time' was removed, use 'http_client.default.retry_wait_time' instead")
	}

	if name == "" {
		name = "default"
	}

	clientsKey := GetClientConfigKey(name)
	defaultClientKey := GetClientConfigKey("default")

	var settings Settings
	config.UnmarshalKey(clientsKey, &settings, cfg.UnmarshalWithDefaultsFromKey(defaultClientKey, "."))

	return settings
}
