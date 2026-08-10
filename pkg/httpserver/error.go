package httpserver

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/pkg/errors"
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
// Mappers are evaluated in registration order and the first matching mapper
// determines the status code. The built-in mappers are registered during
// package initialization, before importing packages can register their own.
func RegisterErrorMapper(mapper ErrorMapper) {
	if mapper == nil {
		panic("error mapper is required")
	}

	errorMappersMu.Lock()
	defer errorMappersMu.Unlock()

	errorMappers = append(errorMappers, mapper)
}

func init() {
	RegisterErrorMapper(func(err error) (int, bool) {
		if errors.Is(err, ErrAccessForbidden) {
			return http.StatusForbidden, true
		}

		return 0, false
	})

	RegisterErrorMapper(func(err error) (int, bool) {
		if exec.IsRequestCanceled(err) {
			return HttpStatusClientWentAway, true
		}

		return 0, false
	})

	RegisterErrorMapper(func(err error) (int, bool) {
		if validation.IsValidationError(err) {
			return http.StatusBadRequest, true
		}

		return 0, false
	})
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
