package http

import (
	"github.com/google/go-querystring/query"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cast"
	"gopkg.in/resty.v1"
	"net/url"
)

const HdrAccept = "Accept"
const HdrContentType = "Content-Type"
const HdrUserAgent = "User-Agent"

const ContentTypeApplicationJson = "application/json"
const ContentTypeApplicationXml = "application/xml"
const ContentTypeApplicationFormUrlencoded = "application/x-www-form-urlencoded"

type Request struct {
	errs        error
	resty       *resty.Request
	url         *url.URL
	queryParams url.Values
}

// use NewRequest to create a request, don't create the object inline!
func NewRequest() *Request {
	return &Request{
		resty:       resty.NewRequest(),
		url:         &url.URL{},
		queryParams: url.Values{},
	}
}

// use NewJsonRequest to create a request that already contains the application/json content-type, don't create the object inline!
func NewJsonRequest() *Request {
	return NewRequest().
		WithHeader(HdrAccept, ContentTypeApplicationJson)
}

// use NewXmlRequest to create a request that already contains the application/xml content-type, don't create the object inline!
func NewXmlRequest() *Request {
	return NewRequest().
		WithHeader(HdrAccept, ContentTypeApplicationXml)
}

func (r *Request) WithUrl(rawUrl string) *Request {
	parsedUrl, err := url.Parse(rawUrl)

	if err != nil {
		r.errs = multierror.Append(r.errs, err)

		return r
	}

	r.addQueryValues(parsedUrl.Query())

	parsedUrl.RawQuery = ""

	r.url = parsedUrl

	return r
}

func (r *Request) WithQueryParam(key string, values ...interface{}) *Request {
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

func (r *Request) WithQueryObject(obj interface{}) *Request {
	parts, err := query.Values(obj)

	if err != nil {
		r.errs = multierror.Append(r.errs, err)

		return r
	}

	r.addQueryValues(parts)

	return r
}

func (r *Request) WithQueryMap(values interface{}) *Request {
	parts, err := cast.ToStringMapStringSliceE(values)

	if err != nil {
		r.errs = multierror.Append(r.errs, err)

		return r
	}

	r.addQueryValues(parts)

	return r
}

func (r *Request) WithBasicAuth(username string, password string) *Request {
	r.resty.SetBasicAuth(username, password)

	return r
}

func (r *Request) WithAuthToken(token string) *Request {
	r.resty.SetAuthToken(token)

	return r
}

func (r *Request) WithHeader(key string, value string) *Request {
	r.resty.SetHeader(key, value)

	return r
}

func (r *Request) WithBody(body interface{}) *Request {
	r.resty.SetBody(body)

	return r
}

// The following methods are mainly intended for tests
// You should not need to call them yourself

type Header map[string][]string

func (r *Request) GetHeader() Header {
	header := make(Header)

	// make a copy of our headers to prevent a caller
	// from modifying them
	for key, values := range r.resty.Header {
		header[key] = append(make([]string, 0, len(values)), values...)
	}

	return header
}

func (r *Request) GetBody() interface{} {
	return r.resty.Body
}

func (r *Request) GetToken() string {
	return r.resty.Token
}

func (r *Request) GetUrl() string {
	r.url.RawQuery = r.queryParams.Encode()

	return r.url.String()
}

func (r *Request) GetError() error {
	return r.errs
}

func (r *Request) build() (*resty.Request, string, error) {
	if r.errs != nil {
		return nil, "", r.errs
	}

	r.url.RawQuery = r.queryParams.Encode()
	urlString := r.url.String()

	return r.resty, urlString, nil
}

func (r *Request) addQueryValues(parts url.Values) {
	for key, values := range parts {
		for _, value := range values {
			r.queryParams.Add(key, value)
		}
	}
}
