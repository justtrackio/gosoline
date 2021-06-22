package auth_test

import (
	"errors"
	"github.com/applike/gosoline/pkg/apiserver/auth"
	authMocks "github.com/applike/gosoline/pkg/apiserver/auth/mocks"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/log/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/oauth2/v2"
	"net/http"
	"testing"
)

func getMocks(idToken string) (log.Logger, *authMocks.TokenInfoProvider, *gin.Context) {
	logger := mocks.NewLoggerMockedAll()
	tokenProvider := new(authMocks.TokenInfoProvider)

	header := http.Header{}
	header.Set("X-ID-TOKEN", idToken)

	request := &http.Request{
		Header: header,
	}
	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return logger, tokenProvider, ginCtx
}

func TestAuthGoogle_Authenticate_EmptyIdTokenError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks("")

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, "c", []string{"h"})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "google auth: zero length token", err.Error())
	}
}

func TestAuthGoogle_Authenticate_IdTokenRequestError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks("test")

	tokenProvider.On("GetTokenInfo", "test").Return(nil, errors.New("test"))
	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, "c", []string{"h"})

	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "google auth: failed requesting token info: test", err.Error())
	}
	tokenProvider.AssertExpectations(t)
}

func TestAuthGoogle_Authenticate_IdTokenStatusCodeError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks("test")

	tokenInfo := &oauth2.Tokeninfo{}
	tokenInfo.HTTPStatusCode = 301

	tokenProvider.On("GetTokenInfo", "test").Return(tokenInfo, nil)

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, "c", []string{"h"})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "google auth: invalid status code 301", err.Error())
	}
	tokenProvider.AssertExpectations(t)
}

func TestAuthGoogle_Authenticate_IdTokenInvalidAudienceError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks("test")

	tokenInfo := &oauth2.Tokeninfo{
		Audience: "infiltrator",
	}
	tokenInfo.HTTPStatusCode = 200

	tokenProvider.On("GetTokenInfo", "test").Return(tokenInfo, nil)

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, "c", []string{"h"})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "google auth: invalid audience", err.Error())
	}
	tokenProvider.AssertExpectations(t)
}

func TestAuthGoogle_Authenticate_IdTokenInvalidEmailSuffixError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks("test")

	tokenInfo := &oauth2.Tokeninfo{
		Audience: "c.de",
		Email:    "a.b@c.be",
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: http.StatusOK,
		},
	}

	tokenProvider.On("GetTokenInfo", "test").Return(tokenInfo, nil)
	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, "c.de", []string{"^.*c\\.de$"})

	_, err := a.IsValid(ginCtx)
	assert.EqualError(t, err, "google auth: address a.b@c.be is not allowed")

	_, err = a.IsValid(ginCtx)
	assert.EqualError(t, err, "token from cache invalidated the user")

	tokenProvider.AssertExpectations(t)
}

func TestAuthGoogle_Authenticate_IdTokenValid(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks("test")

	tokenInfo := &oauth2.Tokeninfo{
		Audience: "c.de",
		Email:    "a.b@c.de",
	}
	tokenInfo.HTTPStatusCode = 200

	tokenProvider.On("GetTokenInfo", "test").Return(tokenInfo, nil)

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, "c.de", []string{"^.*c\\.de$"})
	_, err := a.IsValid(ginCtx)

	assert.NoError(t, err)
	tokenProvider.AssertExpectations(t)
}
