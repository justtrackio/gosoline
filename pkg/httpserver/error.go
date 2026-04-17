package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type ErrorHandler func(statusCode int, err error) *Response

func errorHandlerJson(statusCode int, err error) *Response {
	body := gin.H{"err": err.Error()}
	if statusCode >= 500 {
		body = gin.H{"err": "internal server error"}
	}

	return &Response{
		StatusCode:  statusCode,
		ContentType: mdl.Box(ContentTypeJson),
		Body:        body,
	}
}

func WithErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

func GetErrorHandler() ErrorHandler {
	return defaultErrorHandler
}

var defaultErrorHandler = errorHandlerJson
