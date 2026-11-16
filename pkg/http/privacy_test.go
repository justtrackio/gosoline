package http

import (
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestRequestCredentialsProtectedInReturnedErrors(t *testing.T) {
	request := NewRequest(nil).
		WithUrl("https://example.test/token?password=query-secret-473").
		WithBasicAuth("user", "basic-secret-473").
		WithAuthToken("bearer-secret-473").
		WithHeader("X-Api-Key", "api-secret-473").
		WithSensitiveValues("body-secret-473")
	cause := errors.New("provider echoed query-secret-473 basic-secret-473 bearer-secret-473 api-secret-473 body-secret-473")
	protected := request.protectError(cause, request.GetUrl())
	assert.ErrorIs(t, protected, cause)
	for _, value := range []string{"query-secret-473", "basic-secret-473", "bearer-secret-473", "api-secret-473", "body-secret-473"} {
		assert.NotContains(t, protected.Error(), value)
	}
	assert.Nil(t, errors.Unwrap(log.SafeError(protected)))
	assert.Contains(t, request.GetUrl(), "query-secret-473")
}
