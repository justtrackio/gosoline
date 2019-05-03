package http

import (
	"github.com/applike/gosoline/pkg/mon"
	"gopkg.in/resty.v1"
	"net/http"
	"strconv"
	"time"
)

const GetRequest = "GET"
const PostRequest = "POST"
const HdrUserAgent = "User-Agent"
const metricRequest = "HttpClientRequest"
const metricRequestFailure = "HttpClientRequestFailure"

//go:generate mockery -name Client
type Client interface {
	Get(request *Request) (responseBody []byte, err error)
	Post(request *Request) (responseBody []byte, err error)
	SetTimeout(timeout time.Duration)
	SetUserAgent(ua string)
}

type client struct {
	mo             mon.MetricWriter
	logger         mon.Logger
	http           *resty.Client
	defaultHeaders Headers
}

func NewHttpClient(logger mon.Logger) Client {
	mo := mon.NewMetricDaemonWriter()

	return NewHttpClientWithInterfaces(logger, mo)
}

func NewHttpClientWithInterfaces(logger mon.Logger, mo mon.MetricWriter) Client {
	httpClient := resty.New()
	httpClient.SetRetryCount(5)             //TODO: make it configurable
	httpClient.SetTimeout(time.Second * 30) //TODO: make it configurable

	return &client{
		mo:             mo,
		logger:         logger,
		http:           httpClient,
		defaultHeaders: map[string]string{},
	}
}

func (c *client) SetTimeout(timeout time.Duration) {
	c.http.SetTimeout(timeout)
}

func (c *client) SetUserAgent(ua string) {
	c.defaultHeaders[HdrUserAgent] = ua
}

func (c *client) Get(request *Request) (responseBody []byte, err error) {
	if !request.Headers.Has("Accept") {
		request.Headers.Set("Accept", "application/json")
	}

	return c.do(GetRequest, request)
}

func (c *client) Post(request *Request) (responseBody []byte, err error) {
	if !request.Headers.Has("Content-Type") {
		request.Headers.Set("Content-Type", "application/json")
	}

	return c.do(PostRequest, request)
}

func (c *client) do(method string, request *Request) (responseBody []byte, err error) {
	req := c.buildRawRequest(request)
	url, err := request.GetUrl()

	if err != nil {
		c.logger.Error(err, "failed to assemble request")

		return nil, err
	}

	c.logger.WithFields(mon.Fields{
		"url":    url,
		"method": method,
	}).Debug("Requesting URL")

	c.writeMetric(metricRequest, method)

	resp, err := req.Execute(method, url)

	if err != nil {
		c.logger.WithFields(mon.Fields{
			"err": err,
		}).Errorf(err, "error while requesting url: %s", req.URL)

		c.writeMetric(metricRequestFailure, method)
	}

	responseStatusCode := resp.StatusCode()
	responseStatusCodeString := strconv.Itoa(responseStatusCode)

	if responseStatusCode != http.StatusOK {
		c.logger.WithFields(mon.Fields{
			"statusCode": responseStatusCodeString,
			"url":        req.URL,
		}).Warn("got unexpected response")
	}

	c.writeMetric(metricRequestFailure+responseStatusCodeString, method)

	return resp.Body(), err
}

func (c client) buildRawRequest(request *Request) *resty.Request {
	req := c.http.R()
	req.SetHeaders(c.defaultHeaders)
	req.SetHeaders(request.Headers)
	req.SetBody(request.Body)

	switch request.Authorization.(type) {
	case BasicAuth:
		username := request.Authorization.(BasicAuth).Username
		password := request.Authorization.(BasicAuth).Password
		req.SetBasicAuth(username, password)
		break
	case OAuth:
		token := request.Authorization.(OAuth).Token
		req.SetAuthToken(token)
		break
	default:
		break
	}

	return req
}

func (c *client) writeMetric(metricName string, dimensionType string) {
	c.mo.WriteOne(&mon.MetricDatum{
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: mon.MetricDimensions{
			"Type": dimensionType,
		},
		Unit:  mon.UnitCount,
		Value: 1.0,
	})
}
