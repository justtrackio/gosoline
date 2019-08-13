package apiserver

import (
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/gin-gonic/gin"
)

type ErrorHandler func(statusCode int, err error) *Response

func errorHandlerJson(statusCode int, err error) *Response {
	return &Response{
		StatusCode:  statusCode,
		ContentType: mdl.String(ContentTypeJson),
		Body:        gin.H{"err": err.Error()},
	}
}

func WithErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = errorHandlerJson
