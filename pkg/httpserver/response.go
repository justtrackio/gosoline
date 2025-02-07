package httpserver

import (
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/mdl"
	"google.golang.org/protobuf/proto"
)

// Don't create a response directly, use New*Response instead
type Response struct {
	Body        any
	ContentType *string // might be nil
	Header      http.Header
	StatusCode  int
}

type emptyRenderer struct{}

func (e emptyRenderer) Render(http.ResponseWriter) error {
	return nil
}

func (e emptyRenderer) WriteContentType(_ http.ResponseWriter) {
}

func NewResponse(body any, contentType string, statusCode int, header http.Header, options ...ResponseOption) *Response {
	resp := &Response{
		Body:        body,
		Header:      header,
		ContentType: mdl.NilIfEmpty(contentType),
		StatusCode:  statusCode,
	}

	for _, option := range options {
		option(resp)
	}

	return resp
}

func NewHtmlResponse(body any, options ...ResponseOption) *Response {
	return NewResponse(body, ContentTypeHtml, http.StatusOK, make(http.Header), options...)
}

func NewJsonResponse(body any, options ...ResponseOption) *Response {
	return NewResponse(body, ContentTypeJson, http.StatusOK, make(http.Header), options...)
}

func NewProtobufResponse(body ProtobufEncodable, options ...ResponseOption) (*Response, error) {
	message, err := body.ToMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to transform body to protobuf message: %w", err)
	}

	bytes, err := proto.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to encode body as protobuf message: %w", err)
	}

	return NewResponse(bytes, ContentTypeProtobuf, http.StatusOK, make(http.Header), options...), nil
}

func NewRedirectResponse(url string, options ...ResponseOption) *Response {
	header := make(http.Header)
	header.Set("Location", url)

	return NewResponse(nil, "", http.StatusFound, header, options...)
}

func NewStatusResponse(statusCode int, options ...ResponseOption) *Response {
	var body any
	var contentType string
	// for responses with a client or server error, provide a small response body
	// (unless overwritten later by our caller anyway) containing the error message.
	// we must not do this for the 1xx, 2xx, or 3xx range of status codes as they
	// would interpret the body in some way
	if statusCode >= http.StatusBadRequest {
		body = http.StatusText(statusCode)
		contentType = ContentTypeText
	}

	return NewResponse(body, contentType, statusCode, make(http.Header), options...)
}

func (r *Response) AddHeader(key string, value string) {
	r.Header.Add(key, value)
}

func (r *Response) WithBody(body any) *Response {
	r.Body = body

	return r
}

func (r *Response) WithContentType(contentType string) *Response {
	r.ContentType = &contentType

	return r
}

type ResponseOption func(resp *Response)

func WithStatusCode(statusCode int) ResponseOption {
	return func(resp *Response) {
		resp.StatusCode = statusCode
	}
}
