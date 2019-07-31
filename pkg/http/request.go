package http

import (
	"fmt"
	"net/http"
	"time"
)

type QueryParams map[string][]interface{}
type Headers map[string]string

func (h Headers) Has(s string) bool {
	_, exists := h[s]
	return exists
}

func (h Headers) Set(key string, val string) {
	if h == nil {
		h = Headers{}
	}
	h[key] = val
}

type Request struct {
	Authorization interface{}
	Url           string
	QueryParams   QueryParams
	Headers       Headers
	Body          interface{}
}

type Response struct {
	Body            []byte
	StatusCode      int
	RequestDuration time.Duration
}

// use NewRequest to create a request, don't create the object inline!
func NewRequest(url string) *Request {
	return &Request{
		Url:     url,
		Headers: Headers{},
	}
}

func (request *Request) GetUrl() (string, error) {
	httpRequest, err := http.NewRequest("GET", request.Url, nil)

	if err != nil {
		return "", err
	}

	query := httpRequest.URL.Query()

	for key, values := range request.QueryParams {
		for _, value := range values {
			query.Add(key, fmt.Sprintf("%v", value))
		}
	}

	httpRequest.URL.RawQuery = query.Encode()

	return httpRequest.URL.String(), nil
}
