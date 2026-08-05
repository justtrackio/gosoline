package httpserver_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/assert"
)

type errorReturningHandler struct {
	err error
}

func (h errorReturningHandler) Handle(context.Context, *httpserver.Request) (*httpserver.Response, error) {
	return nil, h.err
}

func TestCreateHandler_UsesRegisteredErrorMapper(t *testing.T) {
	expectedError := errors.New("authorization denied")
	httpserver.RegisterErrorMapper(func(err error) (int, bool) {
		if errors.Is(err, expectedError) {
			return http.StatusForbidden, true
		}

		return 0, false
	})

	handler := httpserver.CreateHandler(errorReturningHandler{
		err: fmt.Errorf("request rejected: %w", expectedError),
	})
	response := httpserver.HttpTest("GET", "/action", "/action", "", handler)

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.JSONEq(t, `{"err":"request rejected: authorization denied"}`, response.Body.String())
}

func TestCreateHandler_PreservesBuiltInForbiddenMapping(t *testing.T) {
	httpserver.RegisterErrorMapper(func(err error) (int, bool) {
		if errors.Is(err, httpserver.ErrAccessForbidden) {
			return http.StatusTeapot, true
		}

		return 0, false
	})

	handler := httpserver.CreateHandler(errorReturningHandler{err: httpserver.ErrAccessForbidden})
	response := httpserver.HttpTest("GET", "/action", "/action", "", handler)

	assert.Equal(t, http.StatusForbidden, response.Code)
}

func TestRegisterErrorMapperRejectsNil(t *testing.T) {
	assert.Panics(t, func() {
		httpserver.RegisterErrorMapper(nil)
	})
}
