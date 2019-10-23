package apiserver

import (
	"context"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"io/ioutil"
	"net/http"
)

const (
	ApiViewKey      = "X-Api-View"
	ContentTypeJson = "application/json; charset=utf-8"
	ContentTypeHtml = "text/html; charset=utf-8"
)

type Request struct {
	Header   http.Header
	Params   gin.Params
	Body     interface{}
	ClientIp string
}

// Don't create a response directly, use New*Response instead
type Response struct {
	StatusCode  int
	ContentType *string // might be nil
	Header      http.Header
	Body        interface{}
}

type emptyRenderer struct{}

func (e emptyRenderer) Render(http.ResponseWriter) error {
	return nil
}

func (e emptyRenderer) WriteContentType(w http.ResponseWriter) {
}

func NewHtmlResponse(body interface{}) *Response {
	return &Response{
		StatusCode:  http.StatusOK,
		ContentType: mdl.String(ContentTypeHtml),
		Body:        body,
		Header:      make(http.Header),
	}
}

func NewJsonResponse(body interface{}) *Response {
	return &Response{
		StatusCode:  http.StatusOK,
		ContentType: mdl.String(ContentTypeJson),
		Body:        body,
		Header:      make(http.Header),
	}
}

func NewRedirectResponse(url string) *Response {
	header := make(http.Header)
	header.Set("Location", url)

	return &Response{
		StatusCode:  http.StatusFound,
		ContentType: nil,
		Body:        nil,
		Header:      header,
	}
}

func NewStatusResponse(statusCode int) *Response {
	return &Response{
		StatusCode:  statusCode,
		ContentType: nil,
		Body:        nil,
		Header:      make(http.Header),
	}
}

func (r *Response) AddHeader(key string, value string) {
	r.Header.Add(key, value)
}

type HandlerWithoutInput interface {
	Handle(requestContext context.Context, request *Request) (response *Response, error error)
}

type HandlerWithInput interface {
	HandlerWithoutInput
	GetInput() interface{}
}

type HandlerWithMultipleBindings interface {
	HandlerWithInput
	GetBindings() []binding.Binding
}

type HandlerWithStream interface {
	Handle(ginContext *gin.Context, requestContext context.Context, request *Request) (error error)
	GetInput() interface{}
}

func CreateHandler(handler HandlerWithoutInput) gin.HandlerFunc {
	return handleWithoutInput(handler, defaultErrorHandler)
}

func CreateRawHandler(handler HandlerWithoutInput) gin.HandlerFunc {
	return handleRaw(handler, defaultErrorHandler)
}

func CreateReaderHandler(handler HandlerWithoutInput) gin.HandlerFunc {
	return handleReader(handler, defaultErrorHandler)
}

func CreateJsonHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithInput(handler, binding.JSON, defaultErrorHandler)
}

func CreateMultipleBindingsHandler(handler HandlerWithMultipleBindings) gin.HandlerFunc {
	return handleWithMultipleBindings(handler, defaultErrorHandler)
}

func CreateStreamHandler(handler HandlerWithStream) gin.HandlerFunc {
	return handleWithStream(handler, binding.JSON, defaultErrorHandler)
}

func CreateQueryHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithInput(handler, binding.Query, defaultErrorHandler)
}

func handleWithoutInput(handler HandlerWithoutInput, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		handle(ginCtx, handler, nil, errHandler)
	}
}

func handleWithStream(handler HandlerWithStream, binding binding.Binding, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()
		err := binding.Bind(ginCtx.Request, input)

		if err != nil {
			handleError(ginCtx, errHandler, http.StatusBadRequest, err)
			return
		}

		reqCtx := ginCtx.Request.Context()
		request := &Request{
			Header:   ginCtx.Request.Header,
			Params:   ginCtx.Params,
			Body:     input,
			ClientIp: ginCtx.ClientIP(),
		}

		err = handler.Handle(ginCtx, reqCtx, request)

		if err != nil {
			handleError(ginCtx, errHandler, http.StatusInternalServerError, err)
			return
		}
	}
}

func handleWithInput(handler HandlerWithInput, binding binding.Binding, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()
		err := binding.Bind(ginCtx.Request, input)

		if err != nil {
			handleError(ginCtx, errHandler, http.StatusBadRequest, err)
			return
		}

		handle(ginCtx, handler, input, errHandler)
	}
}

func handleWithMultipleBindings(handler HandlerWithMultipleBindings, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()
		bindings := handler.GetBindings()

		for i := 0; i < len(bindings); i++ {
			err := bindings[i].Bind(ginCtx.Request, input)

			if err != nil {
				handleError(ginCtx, errHandler, http.StatusBadRequest, err)
				return
			}

		}

		handle(ginCtx, handler, input, errHandler)
	}
}

func handleRaw(handler HandlerWithoutInput, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		body, err := ioutil.ReadAll(ginCtx.Request.Body)

		if err != nil {
			handleError(ginCtx, errHandler, http.StatusBadRequest, err)
			return
		}

		handle(ginCtx, handler, string(body), errHandler)
	}
}

func handleReader(handler HandlerWithoutInput, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		handle(ginCtx, handler, ginCtx.Request.Body, errHandler)
	}
}

func handle(ginCtx *gin.Context, handler HandlerWithoutInput, input interface{}, errHandler ErrorHandler) {
	reqCtx := ginCtx.Request.Context()

	request := &Request{
		Header:   ginCtx.Request.Header,
		Params:   ginCtx.Params,
		Body:     input,
		ClientIp: ginCtx.ClientIP(),
	}

	resp, err := handler.Handle(reqCtx, request)

	if err != nil {
		handleError(ginCtx, errHandler, http.StatusInternalServerError, err)
		return
	}

	writer, err := mkResponseBodyWriter(resp)

	if err != nil {
		handleError(ginCtx, errHandler, http.StatusInternalServerError, err)
		return
	}

	writeResponseHeaders(ginCtx, resp)
	writer(ginCtx)
}

func handleError(ginCtx *gin.Context, errHandler ErrorHandler, statusCode int, err error) {
	_ = ginCtx.Error(err)
	resp := errHandler(statusCode, err)

	writer, err := mkResponseBodyWriter(resp)

	if err != nil {
		panic(errors.WithMessage(err, "Error creating writer for error handler"))
	}

	writer(ginCtx)
}

func writeResponseHeaders(ginCtx *gin.Context, resp *Response) {
	for name, values := range resp.Header {
		for _, value := range values {
			ginCtx.Header(name, value)
		}
	}
}

func mkResponseBodyWriter(resp *Response) (func(ginCtx *gin.Context), error) {
	if resp.ContentType == nil {
		return func(ginCtx *gin.Context) {
			ginCtx.Render(resp.StatusCode, emptyRenderer{})
		}, nil
	}

	if *resp.ContentType == ContentTypeJson {
		return func(ginCtx *gin.Context) {
			ginCtx.JSON(resp.StatusCode, resp.Body)
		}, nil
	}

	if b, ok := resp.Body.([]byte); ok {
		return func(ginCtx *gin.Context) {
			ginCtx.Data(resp.StatusCode, *resp.ContentType, b)
		}, nil
	}

	data, err := cast.ToStringE(resp.Body)

	if err != nil {
		return nil, err
	}

	return func(ginCtx *gin.Context) {
		ginCtx.Data(resp.StatusCode, *resp.ContentType, []byte(data))
	}, nil
}
