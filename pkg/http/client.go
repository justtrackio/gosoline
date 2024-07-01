package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	netUrl "net/url"
	"runtime"
	"time"

	httpHeaders "github.com/go-http-utils/headers"
	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/tracing"
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
	metricWriter   metric.Writer
	defaultHeaders headers
	http           restyClient
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
	TransportSettings      TransportSettings      `cfg:"transport"`
	EnableTracing          bool
}

type TransportSettings struct {
	// TLSHandshakeTimeout specifies the maximum amount of time to
	// wait for a TLS handshake. Zero means no timeout.
	TLSHandshakeTimeout time.Duration `cfg:"tls_handshake_timeout" default:"10s"`

	// DisableKeepAlives, if true, disables HTTP keep-alives and
	// will only use the connection to the server for a single
	// HTTP request.
	//
	// This is unrelated to the similarly named TCP keep-alives.
	DisableKeepAlives bool `cfg:"disable_keep_alives" default:"false"`

	// DisableCompression, if true, prevents the Transport from
	// requesting compression with an "Accept-Encoding: gzip"
	// request header when the Request contains no existing
	// Accept-Encoding value. If the Transport requests gzip on
	// its own and gets a gzipped response, it's transparently
	// decoded in the Response.Body. However, if the user
	// explicitly requested gzip it is not automatically
	// uncompressed.
	DisableCompression bool `cfg:"disable_compression" default:"false"`

	// MaxIdleConns controls the maximum number of idle (keep-alive)
	// connections across all hosts. Zero means no limit.
	MaxIdleConns int `cfg:"max_idle_conns" default:"100"`

	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle
	// (keep-alive) connections to keep per-host. If zero,
	// GOMAXPROCS+1 is used.
	MaxIdleConnsPerHost int `cfg:"max_idle_conns_per_host" default:"0"`

	// MaxConnsPerHost optionally limits the total number of
	// connections per host, including connections in the dialing,
	// active, and idle states. On limit violation, dials will block.
	//
	// Zero means no limit.
	MaxConnsPerHost int `cfg:"max_conns_per_host" default:"0"`

	// IdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	// Zero means no limit.
	IdleConnTimeout time.Duration `cfg:"idle_conn_timeout" default:"90s"`

	// ResponseHeaderTimeout, if non-zero, specifies the amount of
	// time to wait for a server's response headers after fully
	// writing the request (including its body, if any). This
	// time does not include the time to read the response body.
	ResponseHeaderTimeout time.Duration `cfg:"response_header_timeout" default:"0s"`

	// ExpectContinueTimeout, if non-zero, specifies the amount of
	// time to wait for a server's first response headers after fully
	// writing the request headers if the request has an
	// "Expect: 100-continue" header. Zero means no timeout and
	// causes the body to be sent immediately, without
	// waiting for the server to approve.
	// This time does not include the time to send the request header.
	ExpectContinueTimeout time.Duration `cfg:"expect_continue_timeout" default:"1s"`

	// MaxResponseHeaderBytes specifies a limit on how many
	// response bytes are allowed in the server's response
	// header.
	//
	// Zero means to use a default limit.
	MaxResponseHeaderBytes int64 `cfg:"max_response_header_bytes" default:"0"`

	// WriteBufferSize specifies the size of the write buffer used
	// when writing to the transport.
	// If zero, a default (currently 4KB) is used.
	WriteBufferSize int `cfg:"write_buffer_size" default:"0"`

	// ReadBufferSize specifies the size of the read buffer used
	// when reading from the transport.
	// If zero, a default (currently 4KB) is used.
	ReadBufferSize int `cfg:"read_buffer_size" default:"0"`

	DialerSettings DialerSettings `cfg:"dialer"`
}

type DialerSettings struct {
	// KeepAlive specifies the interval between keep-alive
	// probes for an active network connection.
	// If zero, keep-alive probes are sent with a default value
	// (currently 15 seconds), if supported by the protocol and operating
	// system. Network protocols or operating systems that do
	// not support keep-alives ignore this field.
	// If negative, keep-alive probes are disabled.
	KeepAlive time.Duration `cfg:"keep_alive" default:"30s"`

	// Timeout is the maximum amount of time a dial will wait for
	// a connect to complete. If Deadline is also set, it may fail
	// earlier.
	//
	// The default is no timeout.
	//
	// When using TCP and dialing a host name with multiple IP
	// addresses, the timeout may be divided between them.
	//
	// With or without a timeout, the operating system may impose
	// its own earlier timeout. For instance, TCP timeouts are
	// often around 3 minutes.
	Timeout time.Duration `cfg:"timeout" default:"30s"`

	// DualStack previously enabled RFC 6555 Fast Fallback
	// support, also known as "Happy Eyeballs", in which IPv4 is
	// tried soon if IPv6 appears to be misconfigured and
	// hanging.
	//
	// Deprecated: Fast Fallback is enabled by default. To
	// disable, set FallbackDelay to a negative value.
	DualStack bool `cfg:"dual_stack" default:"true"`

	// FallbackDelay specifies the length of time to wait before
	// spawning a RFC 6555 Fast Fallback connection. That is, this
	// is the amount of time to wait for IPv6 to succeed before
	// assuming that IPv6 is misconfigured and falling back to
	// IPv4.
	//
	// If zero, a default delay of 300ms is used.
	// A negative value disables Fast Fallback support.
	FallbackDelay time.Duration `cfg:"fallback_delay" default:"0s"`
}

func ProvideHttpClient(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Client, error) {
	type httpClientName string

	return appctx.Provide(ctx, httpClientName(name), func() (Client, error) {
		return newHttpClient(ctx, config, logger, name)
	})
}

func newHttpClient(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Client, error) {
	metricWriter := metric.NewWriter()
	tracer, err := tracing.ProvideTracer(ctx, config, logger)
	if err != nil {
		return nil, err
	}
	settings := UnmarshalClientSettings(config, name)
	restyClient := newRestyClient(tracer, settings)
	client := NewHttpClientWithInterfaces(logger, clock.Provider, metricWriter, restyClient)

	if settings.CircuitBreakerSettings.Enabled {
		client = NewCircuitBreakerClientWithInterfaces(client, logger, clock.Provider, name, settings.CircuitBreakerSettings)
	}

	return client, nil
}

func newRestyClient(tracer tracing.Tracer, settings Settings) *resty.Client {
	baseHttpClient := &http.Client{}
	if settings.EnableTracing {
		baseHttpClient = tracer.HttpClient(baseHttpClient)
	}

	httpClient := resty.NewWithClient(baseHttpClient)
	if settings.DisableCookies {
		httpClient.SetCookieJar(nil)
	}

	dialer := &net.Dialer{
		Timeout:       settings.TransportSettings.DialerSettings.Timeout,
		KeepAlive:     settings.TransportSettings.DialerSettings.KeepAlive,
		DualStack:     settings.TransportSettings.DialerSettings.DualStack,
		FallbackDelay: settings.TransportSettings.DialerSettings.FallbackDelay,
	}

	if settings.TransportSettings.MaxIdleConnsPerHost == 0 {
		settings.TransportSettings.MaxIdleConnsPerHost = runtime.GOMAXPROCS(0) + 1
	}

	transport := &http.Transport{
		Proxy:                  http.ProxyFromEnvironment,
		DialContext:            dialer.DialContext,
		ForceAttemptHTTP2:      true,
		TLSHandshakeTimeout:    settings.TransportSettings.TLSHandshakeTimeout,
		DisableKeepAlives:      settings.TransportSettings.DisableKeepAlives,
		DisableCompression:     settings.TransportSettings.DisableCompression,
		MaxIdleConns:           settings.TransportSettings.MaxIdleConns,
		MaxIdleConnsPerHost:    settings.TransportSettings.MaxIdleConnsPerHost,
		MaxConnsPerHost:        settings.TransportSettings.MaxConnsPerHost,
		IdleConnTimeout:        settings.TransportSettings.IdleConnTimeout,
		ResponseHeaderTimeout:  settings.TransportSettings.ResponseHeaderTimeout,
		ExpectContinueTimeout:  settings.TransportSettings.ExpectContinueTimeout,
		MaxResponseHeaderBytes: settings.TransportSettings.MaxResponseHeaderBytes,
		WriteBufferSize:        settings.TransportSettings.WriteBufferSize,
		ReadBufferSize:         settings.TransportSettings.ReadBufferSize,
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
	httpClient.SetTransport(transport)

	return httpClient
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
