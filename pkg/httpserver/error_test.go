package httpserver_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// marshalBody marshals resp.Body to JSON and returns it as a string so we can
// compare without depending on the concrete map type returned by gin.H.
func marshalBody(t *testing.T, resp *httpserver.Response) string {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)

	return string(b)
}

func TestErrorHandlerJson_5xxReturnsGenericMessage(t *testing.T) {
	handler := httpserver.GetErrorHandler()
	resp := handler(http.StatusInternalServerError, fmt.Errorf("super secret internal detail"))

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.JSONEq(t, `{"err":"internal server error"}`, marshalBody(t, resp),
		"500 must not expose the real error message")
}

func TestErrorHandlerJson_503ReturnsGenericMessage(t *testing.T) {
	handler := httpserver.GetErrorHandler()
	resp := handler(http.StatusServiceUnavailable, fmt.Errorf("db connection pool exhausted"))

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.JSONEq(t, `{"err":"internal server error"}`, marshalBody(t, resp),
		"503 must not expose the real error message")
}

func TestErrorHandlerJson_4xxExposesActualError(t *testing.T) {
	handler := httpserver.GetErrorHandler()
	resp := handler(http.StatusBadRequest, fmt.Errorf("validation failed: missing field"))

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.JSONEq(t, `{"err":"validation failed: missing field"}`, marshalBody(t, resp),
		"4xx must expose the actual error message")
}

func TestErrorHandlerJson_404ExposesActualError(t *testing.T) {
	handler := httpserver.GetErrorHandler()
	resp := handler(http.StatusNotFound, fmt.Errorf("resource not found"))

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.JSONEq(t, `{"err":"resource not found"}`, marshalBody(t, resp))
}

func TestErrorHandlerJson_499ClientCancelExposesActualError(t *testing.T) {
	// 499 is used for client-cancelled requests; it is < 500 so the real message is shown.
	handler := httpserver.GetErrorHandler()
	resp := handler(499, fmt.Errorf("context canceled"))

	assert.JSONEq(t, `{"err":"context canceled"}`, marshalBody(t, resp))
}
