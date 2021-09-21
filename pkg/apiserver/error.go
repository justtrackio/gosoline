package apiserver

import (
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/mdl"
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

func GetErrorHandler() ErrorHandler {
	return defaultErrorHandler
}

var defaultErrorHandler = errorHandlerJson
