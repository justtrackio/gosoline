package auth_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver/auth"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
)

func getHeaderKeyMocks(idToken string) (log.Logger, *gin.Context) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)

	header := http.Header{}
	header.Set("X-API-KEY", idToken)

	request := &http.Request{
		Header: header,
	}
	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return logger, ginCtx
}

func getQueryKeyMocks(idToken string) (log.Logger, *gin.Context) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)

	request := &http.Request{
		URL: &url.URL{
			RawQuery: fmt.Sprintf("apiKey=%s", idToken),
		},
	}

	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return logger, ginCtx
}

func TestAuthKeyInHeader_Authenticate_InvalidKeyError(t *testing.T) {
	logger, ginCtx := getHeaderKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"a"}, auth.ProvideValueFromHeader(auth.HeaderApiKey))
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "api key does not match", err.Error())
	}
}

func TestAuthKeyInHeader_Authenticate_ValidKey(t *testing.T) {
	logger, ginCtx := getHeaderKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"t"}, auth.ProvideValueFromHeader(auth.HeaderApiKey))
	_, err := a.IsValid(ginCtx)

	assert.Equal(t, nil, err)
}

func TestAuthKeyInQueryParam_Authenticate_InvalidKeyError(t *testing.T) {
	logger, ginCtx := getQueryKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"a"}, auth.ProvideValueFromQueryParam(auth.ByApiKey))
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "api key does not match", err.Error())
	}
}

func TestAuthKeyInQueryParam_Authenticate_ValidKey(t *testing.T) {
	logger, ginCtx := getQueryKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"t"}, auth.ProvideValueFromQueryParam(auth.ByApiKey))
	_, err := a.IsValid(ginCtx)

	assert.Equal(t, nil, err)
}
