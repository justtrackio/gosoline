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
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

const (
	ApiViewKey               = "X-Api-View"
	ContentTypeText          = "text/plain; charset=utf-8"
	ContentTypeHtml          = "text/html; charset=utf-8"
	ContentTypeJson          = "application/json; charset=utf-8"
	ContentTypeProtobuf      = "application/x-protobuf"
	HttpStatusClientWentAway = 499
)

var ErrAccessForbidden = errors.New("cant access resource")

type Request struct {
	Body     any
	ClientIp string
	Cookies  map[string]string
	Header   http.Header
	Method   string
	Params   gin.Params
	Url      *url.URL
}

//go:generate go run github.com/vektra/mockery/v2 --name HandlerWithoutInput
type HandlerWithoutInput interface {
	Handle(requestContext context.Context, request *Request) (response *Response, err error)
}

//go:generate go run github.com/vektra/mockery/v2 --name HandlerWithInput
type HandlerWithInput interface {
	HandlerWithoutInput
	GetInput() any
}

//go:generate go run github.com/vektra/mockery/v2 --name HandlerWithMultipleBindings
type HandlerWithMultipleBindings interface {
	HandlerWithInput
	GetBindings() []binding.Binding
}

//go:generate go run github.com/vektra/mockery/v2 --name HandlerWithStream
type HandlerWithStream interface {
	GetInput() any
	Handle(ginContext *gin.Context, requestContext context.Context, request *Request) (err error)
}

// CreateHandler creates a gin.HandlerFunc that handles the request without data binding and without passing the body to the handler
func CreateHandler(handler HandlerWithoutInput) gin.HandlerFunc {
	return handleWithoutInput(handler, defaultErrorHandler)
}

// CreateJsonHandler creates a gin.HandlerFunc that handles the request with json binding
// Example input struct from handler.GetInput():
//
//	type example struct{
//	  A string `json:"a" binding:"required"`
//	}
func CreateJsonHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithBindingInput(handler, binding.JSON, defaultErrorHandler)
}

// CreateProtobufHandler creates a gin.HandlerFunc that handles the request with protobuf binding
// Example input struct from handler.GetInput():
//
//	type example struct{ // <- this struct must implement ProtobufDecodable
//	  A string `json:"a" binding:"required"`
//	}
func CreateProtobufHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithBindingInput(handler, protobufBinding, defaultErrorHandler)
}

// CreateMultiPartFormHandler creates a gin.HandlerFunc that handles the request with Form multipart data binding
// Example input struct from handler.GetInput():
//
//	type example struct{
//	  File *multipart.FileHeader `form:"file" binding:"required"`
//	}
func CreateMultiPartFormHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithMultiPartFormInput(handler, defaultErrorHandler)
}

// CreateMultipleBindingsHandler creates a gin.HandlerFunc that handles the request with the bindings the input struct specifies
// Example input struct from handler.GetInput():
//
//	type example struct{
//	  A string `json:"a" binding:"required"`
//	  B string `form:"b" binding:"required"`
//	}
func CreateMultipleBindingsHandler(handler HandlerWithMultipleBindings) gin.HandlerFunc {
	return handleWithMultipleBindings(handler, defaultErrorHandler)
}

// CreateRawHandler creates a gin.HandlerFunc that handles the request without input binding and passes the body to the handler as []byte
func CreateRawHandler(handler HandlerWithoutInput) gin.HandlerFunc {
	return handleRaw(handler, defaultErrorHandler)
}

// CreateReaderHandler creates a gin.HandlerFunc that handles the request without input binding and passes the body to the handler as io.ReadCloser
func CreateReaderHandler(handler HandlerWithoutInput) gin.HandlerFunc {
	return handleReader(handler, defaultErrorHandler)
}

// CreateQueryHandler creates a gin.HandlerFunc that handles the request and uses the query parameters as input binding
// Example input struct from handler.GetInput():
//
//	type example struct{
//	  A string `form:"a" binding:"required"`
//	}
func CreateQueryHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithBindingInput(handler, binding.Query, defaultErrorHandler)
}

// CreateUriHandler creates a gin.HandlerFunc that handles the request and uses the uri parameters as input binding
// Example input struct from handler.GetInput():
//
//	type example struct{
//	  A string `uri:"a" binding:"required"`
//	}
func CreateUriHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithBindingUriInput(handler, defaultErrorHandler)
}

// CreateSseHandler creates a gin.HandlerFunc that handles the stream request and uses the query parameters as input binding
// Example input struct from handler.GetInput():
//
//	type example struct{
//	  A string `query:"a" binding:"required"`
//	}
func CreateSseHandler(handler HandlerWithStream) gin.HandlerFunc {
	return handleWithStream(handler, binding.Query, defaultErrorHandler)
}

// CreateStreamHandler creates a gin.HandlerFunc that handles the stream request and uses the json body as input binding
// Example input struct from handler.GetInput():
//
//	type example struct{
//	  A string `json:"a" binding:"required"`
//	}
func CreateStreamHandler(handler HandlerWithStream) gin.HandlerFunc {
	return handleWithStream(handler, binding.JSON, defaultErrorHandler)
}

// CreateDownloadHandler creates a gin.HandlerFunc that handles the stream request and uses the query parameters as input binding
// Example input struct from handler.GetInput():
//
//	type example struct{
//	  A string `query:"a" binding:"required"`
//	}
func CreateDownloadHandler(handler HandlerWithStream) gin.HandlerFunc {
	return handleWithStream(handler, binding.Query, defaultErrorHandler)
}

func handleWithBindingInput(handler HandlerWithInput, binding binding.Binding, errHandler ErrorHandler) gin.HandlerFunc {
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

func handleWithBindingUriInput(handler HandlerWithInput, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()

		err := ginCtx.ShouldBindUri(input)
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
			validErr := &validation.Error{}
			if errors.As(err, &validErr) {
				handleError(ginCtx, errHandler, http.StatusBadRequest, gin.Error{
					Err:  validErr,
					Type: gin.ErrorTypePrivate,
				})

				return
			}

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

func handle(ginCtx *gin.Context, handler HandlerWithoutInput, input any, errHandler ErrorHandler) {
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

	if exec.IsRequestCanceled(err) {
		handleError(ginCtx, errHandler, HttpStatusClientWentAway, gin.Error{
			Err:  err,
			Type: gin.ErrorTypePrivate,
		})

		return
	}

	validErr := &validation.Error{}
	if errors.As(err, &validErr) {
		handleError(ginCtx, errHandler, http.StatusBadRequest, gin.Error{
			// only pass the validation error in case it is wrapped in something else
			Err:  validErr,
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

func isMutableInput(input any) bool {
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
	//nolint:errcheck // we just want to add the error to the context and are not interested in the result
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
