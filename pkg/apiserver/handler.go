package apiserver

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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

type Response struct {
	StatusCode  int
	ContentType string
	Header      http.Header
	Body        interface{}
}

func NewHtmlResponse(body interface{}) *Response {
	return &Response{
		StatusCode:  http.StatusOK,
		ContentType: ContentTypeHtml,
		Body:        body,
		Header:      make(http.Header),
	}
}

func NewJsonResponse(body interface{}) *Response {
	return &Response{
		StatusCode:  http.StatusOK,
		ContentType: ContentTypeJson,
		Body:        body,
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

func CreateJsonHandler(handler HandlerWithInput) gin.HandlerFunc {
	return handleWithInput(handler, binding.JSON, defaultErrorHandler)
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
			writeErrorResponse(ginCtx, errHandler, http.StatusBadRequest, err)
			return
		}

		reqCtx := ginCtx.Request.Context()
		request := &Request{
			Params: ginCtx.Params,
			Body:   input,
		}

		err = handler.Handle(ginCtx, reqCtx, request)

		if err != nil {
			writeErrorResponse(ginCtx, errHandler, http.StatusInternalServerError, err)
			return
		}
	}
}

func handleWithInput(handler HandlerWithInput, binding binding.Binding, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		input := handler.GetInput()
		err := binding.Bind(ginCtx.Request, input)

		if err != nil {
			writeErrorResponse(ginCtx, errHandler, http.StatusBadRequest, err)
			return
		}

		handle(ginCtx, handler, input, errHandler)
	}
}

func handleRaw(handler HandlerWithoutInput, errHandler ErrorHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		body, err := ioutil.ReadAll(ginCtx.Request.Body)

		if err != nil {
			writeErrorResponse(ginCtx, errHandler, http.StatusBadRequest, err)
			return
		}

		handle(ginCtx, handler, string(body), errHandler)
	}
}

func handle(ginCtx *gin.Context, handler HandlerWithoutInput, requestBody interface{}, errHandler ErrorHandler) {
	reqCtx := ginCtx.Request.Context()

	request := &Request{
		Header:   ginCtx.Request.Header,
		Params:   ginCtx.Params,
		Body:     requestBody,
		ClientIp: ginCtx.ClientIP(),
	}

	resp, err := handler.Handle(reqCtx, request)

	if err != nil {
		writeErrorResponse(ginCtx, errHandler, http.StatusInternalServerError, err)
		return
	}

	writeResponseHeaders(ginCtx, resp)
	writeResponseBody(ginCtx, resp)
}

func writeErrorResponse(ginCtx *gin.Context, errHandler ErrorHandler, statusCode int, err error) {
	resp := errHandler(statusCode, err)
	writeResponseBody(ginCtx, resp)
}

func writeResponseHeaders(ginCtx *gin.Context, resp *Response) {
	for name, values := range resp.Header {
		for _, value := range values {
			ginCtx.Header(name, value)
		}
	}
}

func writeResponseBody(ginCtx *gin.Context, resp *Response) {
	switch resp.ContentType {
	case ContentTypeHtml:
		switch b := resp.Body.(type) {
		case string:
			ginCtx.Data(resp.StatusCode, resp.ContentType, []byte(b))
		case []byte:
			ginCtx.Data(resp.StatusCode, resp.ContentType, b)
		}

	case ContentTypeJson:
		ginCtx.JSON(resp.StatusCode, resp.Body)
		break
	}
}
