package httpserver

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type ErrorHandler func(statusCode int, err error) *Response

// ErrorMapper maps an application error to an HTTP status code. The handled
// result indicates whether the mapper applies to the error.
type ErrorMapper func(err error) (statusCode int, handled bool)

var (
	errorMappersMu sync.RWMutex
	errorMappers   []ErrorMapper
)

// RegisterErrorMapper adds a status mapper used by the HTTP handler helpers.
// Mappers are evaluated for errors that would otherwise use the generic 500
// fallback; the built-in forbidden, cancellation, and validation mappings take
// precedence.
func RegisterErrorMapper(mapper ErrorMapper) {
	if mapper == nil {
		panic("error mapper is required")
	}

	errorMappersMu.Lock()
	defer errorMappersMu.Unlock()

	errorMappers = append(errorMappers, mapper)
}

func errorStatusCodeFromMappers(err error) (int, bool) {
	errorMappersMu.RLock()
	mappers := append([]ErrorMapper(nil), errorMappers...)
	errorMappersMu.RUnlock()

	for _, mapper := range mappers {
		if statusCode, handled := mapper(err); handled {
			return statusCode, true
		}
	}

	return 0, false
}

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
