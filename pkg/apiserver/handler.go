package apiserver

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/coffin"
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

func (e emptyRenderer) WriteContentType(_ http.ResponseWriter) {
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

func CreateJsonHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithInput(handler, binding.JSON, defaultErrorHandler)
}

func CreateMultiPartFormHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithMultiPartFormInput(handler, defaultErrorHandler)
}

func CreateMultipleBindingsHandler(handler HandlerWithMultipleBindings) gin.HandlerFunc {
	return handleWithMultipleBindings(handler, defaultErrorHandler)
}

func CreateRawHandler(handler HandlerWithoutInput) gin.HandlerFunc {
	return handleRaw(handler, defaultErrorHandler)
}

func CreateReaderHandler(handler HandlerWithoutInput) gin.HandlerFunc {
	return handleReader(handler, defaultErrorHandler)
}

func CreateQueryHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithInput(handler, binding.Query, defaultErrorHandler)
}

func CreateSseHandler(handler HandlerWithStream) gin.HandlerFunc {
	return handleWithStream(handler, binding.Query, defaultErrorHandler)
}

func CreateStreamHandler(handler HandlerWithStream) gin.HandlerFunc {
	return handleWithStream(handler, binding.JSON, defaultErrorHandler)
}

func handleWithInput(handler HandlerWithInput, binding binding.Binding, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()
		err := binding.Bind(ginCtx.Request, input)

		if err != nil {
			handleError(ginCtx, errHandler, http.StatusBadRequest, gin.Error{
				Err:  err,
				Type: gin.ErrorTypeBind,
			})
			return
		}

		handle(ginCtx, handler, input, errHandler)
	}
}

func handleWithoutInput(handler HandlerWithoutInput, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		handle(ginCtx, handler, nil, errHandler)
	}
}

func handleWithMultiPartFormInput(handler HandlerWithInput, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()
		err := binding.FormMultipart.Bind(ginCtx.Request, input)

		if err != nil && !errors.Is(err, http.ErrMissingFile) {
			handleError(ginCtx, errHandler, http.StatusBadRequest, gin.Error{
				Err:  err,
				Type: gin.ErrorTypeBind,
			})
			return
		}

		handle(ginCtx, handler, input, errHandler)
	}
}

func handleWithStream(handler HandlerWithStream, binding binding.Binding, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()
		err := binding.Bind(ginCtx.Request, input)

		if err != nil {
			handleError(ginCtx, errHandler, http.StatusBadRequest, gin.Error{
				Err:  err,
				Type: gin.ErrorTypeBind,
			})
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
			handleError(ginCtx, errHandler, http.StatusInternalServerError, gin.Error{
				Err:  err,
				Type: gin.ErrorTypePrivate,
			})
			return
		}
	}
}

func handleWithMultipleBindings(handler HandlerWithMultipleBindings, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()
		bindings := handler.GetBindings()

		for i := 0; i < len(bindings); i++ {
			err := bindings[i].Bind(ginCtx.Request, input)

			if err != nil {
				handleError(ginCtx, errHandler, http.StatusBadRequest, gin.Error{
					Err:  err,
					Type: gin.ErrorTypeBind,
				})
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
			handleError(ginCtx, errHandler, http.StatusBadRequest, gin.Error{
				Err:  err,
				Type: gin.ErrorTypeBind,
			})
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
		handleError(ginCtx, errHandler, http.StatusInternalServerError, gin.Error{
			Err:  err,
			Type: gin.ErrorTypePrivate,
		})
		return
	}

	writer, err := mkResponseBodyWriter(resp)

	if err != nil {
		handleError(ginCtx, errHandler, http.StatusInternalServerError, gin.Error{
			Err:  err,
			Type: gin.ErrorTypeRender,
		})
		return
	}

	writeResponseHeaders(ginCtx, resp)
	writer(ginCtx)
}

func handleError(ginCtx *gin.Context, errHandler ErrorHandler, statusCode int, ginError gin.Error) {
	_ = ginCtx.Error(&ginError)
	resp := errHandler(statusCode, ginError.Err)

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
		return withRecover(func(ginCtx *gin.Context) {
			ginCtx.Render(resp.StatusCode, emptyRenderer{})
		}), nil
	}

	if *resp.ContentType == ContentTypeJson {
		return withRecover(func(ginCtx *gin.Context) {
			ginCtx.JSON(resp.StatusCode, resp.Body)
		}), nil
	}

	if b, ok := resp.Body.([]byte); ok {
		return withRecover(func(ginCtx *gin.Context) {
			ginCtx.Data(resp.StatusCode, *resp.ContentType, b)
		}), nil
	}

	data, err := cast.ToStringE(resp.Body)

	if err != nil {
		return nil, err
	}

	return withRecover(func(ginCtx *gin.Context) {
		ginCtx.Data(resp.StatusCode, *resp.ContentType, []byte(data))
	}), nil
}

type ResponseBodyWriterError struct {
	Err error
}

func (e ResponseBodyWriterError) Error() string {
	return fmt.Sprintf("handler response body writer error: %s", e.Err.Error())
}

func (e ResponseBodyWriterError) Unwrap() error {
	return e.Err
}

func (e ResponseBodyWriterError) Is(err error) bool {
	_, ok := err.(ResponseBodyWriterError)

	return ok
}

func withRecover(f func(ginCtx *gin.Context)) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		defer func() {
			err := coffin.ResolveRecovery(recover())

			if err == nil {
				return
			}

			// When building the response body, should there be an e.g. broken pipe error, we do not want to
			// log an application error. Instead, RecoveryWithSentry within middleware_recover will log a warning instead
			panic(ResponseBodyWriterError{Err: err})
		}()

		f(ginCtx)
	}
}
