package http

import (
	"io"
	"net/url"
	"sync"

	httpHeaders "github.com/go-http-utils/headers"
	"github.com/go-resty/resty/v2"
	"github.com/google/go-querystring/query"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cast"
)

type Request struct {
	errs           error
	outputFile     *string
	queryParams    url.Values
	restyRequest   *resty.Request
	url            *url.URL
	forwardTraceId bool
}

var r struct {
	instance *resty.Client
	lck      sync.Mutex
}

// NewRequest or client.NewRequest() creates a request, don't create the object inline!
func NewRequest(client Client) *Request {
	r.lck.Lock()
	defer r.lck.Unlock()

	if r.instance == nil {
		r.instance = resty.New()
	}

	if client == nil {
		return &Request{
			restyRequest: r.instance.NewRequest(),
			url:          &url.URL{},
			queryParams:  url.Values{},
		}
	}

	return client.NewRequest()
}

// NewJsonRequest creates a request that already contains the application/json content-type, don't create the object inline!
func NewJsonRequest(client Client) *Request {
	return NewRequest(client).
		WithHeader(httpHeaders.Accept, MimeTypeApplicationJson)
}

// NewXmlRequest creates a request that already contains the application/xml content-type, don't create the object inline!
func NewXmlRequest(client Client) *Request {
	return NewRequest(client).
		WithHeader(httpHeaders.Accept, MimeTypeApplicationXml)
}

func (r *Request) WithUrl(rawUrl string) *Request {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		r.errs = multierror.Append(r.errs, err)

		return r
	}

	r.url = parsedUrl

	if len(r.queryParams) != 0 {
		r.takeQueryParamsFromUrl()
	}

	return r
}

func (r *Request) WithQueryParam(key string, values ...any) *Request {
	r.takeQueryParamsFromUrl()

	for _, value := range values {
		str, err := cast.ToStringE(value)
		if err != nil {
			r.errs = multierror.Append(r.errs, err)

			continue
		}

		r.queryParams.Add(key, str)
	}

	return r
}

func (r *Request) WithQueryObject(obj any) *Request {
	parts, err := query.Values(obj)
	if err != nil {
		r.errs = multierror.Append(r.errs, err)

		return r
	}

	r.takeQueryParamsFromUrl()
	r.addQueryValues(parts)

	return r
}

func (r *Request) WithQueryMap(values any) *Request {
	parts, err := cast.ToStringMapStringSliceE(values)
	if err != nil {
		r.errs = multierror.Append(r.errs, err)

		return r
	}

	r.takeQueryParamsFromUrl()
	r.addQueryValues(parts)

	return r
}

func (r *Request) WithBasicAuth(username string, password string) *Request {
	r.restyRequest.SetBasicAuth(username, password)

	return r
}

func (r *Request) WithAuthToken(token string) *Request {
	r.restyRequest.SetAuthToken(token)

	return r
}

func (r *Request) WithHeader(key string, value string) *Request {
	r.restyRequest.SetHeader(key, value)

	return r
}

func (r *Request) WithBody(body any) *Request {
	r.restyRequest.SetBody(body)

	return r
}

func (r *Request) WithMultipartFile(param, fileName string, reader io.Reader) *Request {
	r.restyRequest.SetFileReader(param, fileName, reader)

	return r
}

func (r *Request) WithMultipartFormData(params url.Values) *Request {
	r.restyRequest.SetFormDataFromValues(params)

	return r
}

func (r *Request) WithOutputFile(path string) *Request {
	r.outputFile = &path

	return r
}

// WithForwardTraceId allows you to configure whether you want to add the trace ID header to the downstream systems
func (r *Request) WithForwardTraceId(forwardTraceId bool) *Request {
	r.forwardTraceId = forwardTraceId

	return r
}

// The following methods are mainly intended for tests
// You should not need to call them yourself

type Header map[string][]string

func (r *Request) GetHeader() Header {
	header := make(Header)

	// make a copy of our headers to prevent a caller
	// from modifying them
	for key, values := range r.restyRequest.Header {
		header[key] = append(make([]string, 0, len(values)), values...)
	}

	return header
}

func (r *Request) GetBody() any {
	return r.restyRequest.Body
}

func (r *Request) GetToken() string {
	return r.restyRequest.Token
}

func (r *Request) GetUrl() string {
	if len(r.queryParams) != 0 {
		r.url.RawQuery = r.queryParams.Encode()
	}

	return r.url.String()
}

func (r *Request) GetError() error {
	return r.errs
}

func (r *Request) build() (*resty.Request, string, error) {
	if r.errs != nil {
		return nil, "", r.errs
	}

	return r.restyRequest, r.GetUrl(), nil
}

func (r *Request) addQueryValues(parts url.Values) {
	for key, values := range parts {
		for _, value := range values {
			r.queryParams.Add(key, value)
		}
	}
}

// takeQueryParamsFromUrl parses the query parameters in the url object (if any) and merges them into the queryParams object
// We need this because we keep query params at two places: Either directly (without decoding and re-encoding) in the url
// object or in the queryParams object (after adding some manually). This allows you to specify urls with a specific encoding
// if you need to influence which characters are percent-encoded (because of a non-behaving service on the other side you
// can't change).
func (r *Request) takeQueryParamsFromUrl() {
	if r.url.RawQuery == "" {
		return
	}

	r.addQueryValues(r.url.Query())
	r.url.RawQuery = ""
}
