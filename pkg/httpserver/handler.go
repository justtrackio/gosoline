package httpserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/imdario/mergo"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

const (
	ApiViewKey      = "X-Api-View"
	ContentTypeHtml = "text/html; charset=utf-8"
	ContentTypeJson = "application/json; charset=utf-8"
)

var ErrAccessForbidden = errors.New("cant access resource")

type Request struct {
	Body     interface{}
	ClientIp string
	Cookies  map[string]string
	Header   http.Header
	Method   string
	Params   gin.Params
	Url      *url.URL
}

// Don't create a response directly, use New*Response instead
type Response struct {
	Body        interface{}
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

func NewResponse(body interface{}, contentType string, statusCode int, header http.Header) *Response {
	return &Response{
		Body:        body,
		Header:      header,
		ContentType: &contentType,
		StatusCode:  statusCode,
	}
}

func NewHtmlResponse(body interface{}) *Response {
	return &Response{
		StatusCode:  http.StatusOK,
		ContentType: mdl.Box(ContentTypeHtml),
		Body:        body,
		Header:      make(http.Header),
	}
}

func NewJsonResponse(body interface{}) *Response {
	return &Response{
		StatusCode:  http.StatusOK,
		ContentType: mdl.Box(ContentTypeJson),
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

func (r *Response) WithBody(body interface{}) *Response {
	r.Body = body

	return r
}

func (r *Response) WithContentType(contentType string) *Response {
	r.ContentType = &contentType

	return r
}

//go:generate mockery --name HandlerWithoutInput
type HandlerWithoutInput interface {
	Handle(requestContext context.Context, request *Request) (response *Response, err error)
}

//go:generate mockery --name HandlerWithInput
type HandlerWithInput interface {
	HandlerWithoutInput
	GetInput() interface{}
}

//go:generate mockery --name HandlerWithMultipleBindings
type HandlerWithMultipleBindings interface {
	HandlerWithInput
	GetBindings() []binding.Binding
}

//go:generate mockery --name HandlerWithStream
type HandlerWithStream interface {
	GetInput() interface{}
	Handle(ginContext *gin.Context, requestContext context.Context, request *Request) (err error)
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

func CreateDownloadHandler(handler HandlerWithStream) gin.HandlerFunc {
	return handleWithStream(handler, binding.Query, defaultErrorHandler)
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

		ginUrl, err := parseUrl(ginCtx)
		if err != nil {
			handleError(ginCtx, errHandler, http.StatusInternalServerError, gin.Error{
				Err:  err,
				Type: gin.ErrorTypePrivate,
			})

			return
		}

		reqCtx := ginCtx.Request.Context()
		request := &Request{
			Method:   ginCtx.Request.Method,
			Header:   ginCtx.Request.Header,
			Cookies:  parseCookies(ginCtx.Request),
			Params:   ginCtx.Params,
			Url:      ginUrl,
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
		body, err := io.ReadAll(ginCtx.Request.Body)
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

	ginUrl, err := parseUrl(ginCtx)
	if err != nil {
		handleError(ginCtx, errHandler, http.StatusInternalServerError, gin.Error{
			Err:  err,
			Type: gin.ErrorTypePrivate,
		})

		return
	}

	if isMutableInput(input) {
		modErr := modifyInput(ginCtx, input)

		if modErr != nil {
			handleError(ginCtx, errHandler, http.StatusBadRequest, gin.Error{
				Err:  modErr,
				Type: gin.ErrorTypePrivate,
			})

			return
		}
	}

	request := &Request{
		Method:   ginCtx.Request.Method,
		Header:   ginCtx.Request.Header,
		Cookies:  parseCookies(ginCtx.Request),
		Params:   ginCtx.Params,
		Url:      ginUrl,
		Body:     input,
		ClientIp: ginCtx.ClientIP(),
	}

	resp, err := handler.Handle(reqCtx, request)

	if errors.Is(err, ErrAccessForbidden) {
		handleForbidden(ginCtx, errHandler, http.StatusForbidden, gin.Error{
			Err:  err,
			Type: gin.ErrorTypePrivate,
		})

		return
	}

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

func isMutableInput(input interface{}) bool {
	if input == nil {
		return false
	}

	if reflect.ValueOf(input).Kind() != reflect.Pointer {
		return false
	}

	if reflect.ValueOf(input).Elem().Kind() != reflect.Struct {
		return false
	}

	return true
}

func parseCookies(request *http.Request) map[string]string {
	result := make(map[string]string)

	for _, cookie := range request.Cookies() {
		result[cookie.Name] = cookie.Value
	}

	return result
}

func parseUrl(ctx *gin.Context) (*url.URL, error) {
	ginUrl := location.Get(ctx)
	if ginUrl == nil {
		ginUrl = &url.URL{}
	}

	reqUrl := ctx.Request.URL
	if reqUrl == nil {
		reqUrl = &url.URL{}
	}

	if err := mergo.Merge(reqUrl, ginUrl); err != nil {
		return nil, fmt.Errorf("could not merge urls: %w", err)
	}

	if ctx.Request.Host != "" {
		reqUrl.Host = ctx.Request.Host
	}

	return reqUrl, nil
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

func handleForbidden(ginCtx *gin.Context, errHandler ErrorHandler, statusCode int, ginError gin.Error) {
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
