package apiserver

import "github.com/gin-gonic/gin"

type ErrorHandler func(statusCode int, err error) *Response

func errorHandlerJson(statusCode int, err error) *Response {
	return &Response{
		StatusCode:  statusCode,
		ContentType: ContentTypeJson,
		Body:        gin.H{"err": err.Error()},
	}
}

func WithErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = errorHandlerJson
