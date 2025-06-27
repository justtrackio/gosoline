package auth_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver/auth"
	authMocks "github.com/justtrackio/gosoline/pkg/httpserver/auth/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/oauth2/v2"
)

func getMocks(t *testing.T, idToken string) (log.Logger, *authMocks.TokenInfoProvider, *gin.Context) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	tokenProvider := authMocks.NewTokenInfoProvider(t)

	ginCtx := getGinCtx(idToken)

	return logger, tokenProvider, ginCtx
}

func getGinCtx(idToken string) *gin.Context {
	header := http.Header{}
	header.Set("X-ID-TOKEN", idToken)

	request := &http.Request{
		Header: header,
	}
	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return ginCtx
}

func TestAuthGoogle_Authenticate_EmptyIdTokenError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks(t, "")

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, []string{"c"}, []string{"h"})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "google auth: zero length token", err.Error())
	}
}

func TestAuthGoogle_Authenticate_IdTokenRequestError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks(t, "test")

	tokenProvider.EXPECT().GetTokenInfo("test").Return(nil, errors.New("test"))
	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, []string{"c"}, []string{"h"})

	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "google auth: failed requesting token info: test", err.Error())
	}
}

func TestAuthGoogle_Authenticate_IdTokenStatusCodeError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks(t, "test")

	tokenInfo := &oauth2.Tokeninfo{}
	tokenInfo.HTTPStatusCode = 301

	tokenProvider.EXPECT().GetTokenInfo("test").Return(tokenInfo, nil)

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, []string{"c"}, []string{"h"})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "google auth: invalid status code 301", err.Error())
	}
}

func TestAuthGoogle_Authenticate_IdTokenInvalidAudienceError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks(t, "test")

	tokenInfo := &oauth2.Tokeninfo{
		Audience: "infiltrator",
	}
	tokenInfo.HTTPStatusCode = 200

	tokenProvider.EXPECT().GetTokenInfo("test").Return(tokenInfo, nil)

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, []string{"c"}, []string{"h"})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "google auth: invalid audience", err.Error())
	}
}

func TestAuthGoogle_Authenticate_IdTokenInvalidEmailSuffixError(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks(t, "test")

	tokenInfo := &oauth2.Tokeninfo{
		Audience: "c.de",
		Email:    "a.b@c.be",
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: http.StatusOK,
		},
	}

	tokenProvider.EXPECT().GetTokenInfo("test").Return(tokenInfo, nil)
	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, []string{"c.de"}, []string{"^.*c\\.de$"})

	_, err := a.IsValid(ginCtx)
	assert.EqualError(t, err, "google auth: address a.b@c.be is not allowed")

	_, err = a.IsValid(ginCtx)
	assert.EqualError(t, err, "token from cache invalidated the user")
}

func TestAuthGoogle_Authenticate_IdTokenValid(t *testing.T) {
	logger, tokenProvider, ginCtx := getMocks(t, "test")

	tokenInfo := &oauth2.Tokeninfo{
		Audience: "c.de",
		Email:    "a.b@c.de",
	}
	tokenInfo.HTTPStatusCode = 200

	tokenProvider.EXPECT().GetTokenInfo("test").Return(tokenInfo, nil)

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, []string{"c.de"}, []string{"^.*c\\.de$"})
	_, err := a.IsValid(ginCtx)

	assert.NoError(t, err)
}

func TestAuthGoogle_Authenticate_MultipleAudiences(t *testing.T) {
	logger, tokenProvider, _ := getMocks(t, "")

	tokenInfoA := &oauth2.Tokeninfo{
		Audience: "c.de",
		Email:    "a.b@c.de",
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: http.StatusOK,
		},
	}

	ginTestACtx := getGinCtx("testA")

	tokenProvider.EXPECT().GetTokenInfo("testA").Return(tokenInfoA, nil)

	a := auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, []string{"d.de", "c.de"}, []string{"^.*[cd]\\.de$"})
	_, err := a.IsValid(ginTestACtx)

	assert.NoError(t, err)

	ginTestBCtx := getGinCtx("testB")

	tokenInfoB := &oauth2.Tokeninfo{
		Audience: "d.de",
		Email:    "b.a@d.de",
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: http.StatusOK,
		},
	}

	tokenProvider.EXPECT().GetTokenInfo("testB").Return(tokenInfoB, nil)

	a = auth.NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, []string{"d.de", "c.de"}, []string{"^.*[cd]\\.de$"})
	_, err = a.IsValid(ginTestBCtx)

	assert.NoError(t, err)

	assert.NoError(t, err)
}
