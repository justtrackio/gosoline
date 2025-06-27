package auth_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/justtrackio/gosoline/pkg/httpserver/auth"
	authMocks "github.com/justtrackio/gosoline/pkg/httpserver/auth/mocks"
	"github.com/stretchr/testify/assert"
)

func getBasicJwtAuthMocks(t *testing.T, token string) (*authMocks.JwtTokenHandler, *gin.Context) {
	jwtTokenHandler := authMocks.NewJwtTokenHandler(t)

	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	request := &http.Request{
		Header: header,
	}
	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return jwtTokenHandler, ginCtx
}

func TestJwtAuth_Authenticate_IsValid(t *testing.T) {
	tokenHandler, ginCtx := getBasicJwtAuthMocks(t, "token")

	a := auth.NewJWTAuthAuthenticatorWithInterfaces(tokenHandler)

	tokenHandler.EXPECT().Valid("token").Return(true, &jwt.Token{
		Claims: jwt.MapClaims{
			"email": "email",
		},
	}, nil)

	isValid, err := a.IsValid(ginCtx)

	assert.True(t, isValid)
	assert.Nil(t, err)
}

func TestJwtAuth_Authenticate_ValidButMissingEmail(t *testing.T) {
	tokenHandler, ginCtx := getBasicJwtAuthMocks(t, "token")

	a := auth.NewJWTAuthAuthenticatorWithInterfaces(tokenHandler)

	tokenHandler.EXPECT().Valid("token").Return(true, &jwt.Token{
		Claims: jwt.MapClaims{
			"something": "else",
		},
	}, nil)

	isValid, err := a.IsValid(ginCtx)

	assert.False(t, isValid)
	assert.EqualError(t, err, "jwt token is missing email field")
}

func TestJwtAuth_Authenticate_IsValid_Error(t *testing.T) {
	tokenHandler, ginCtx := getBasicJwtAuthMocks(t, "token")

	a := auth.NewJWTAuthAuthenticatorWithInterfaces(tokenHandler)

	tokenHandler.EXPECT().Valid("token").Return(false, &jwt.Token{}, nil)

	isValid, err := a.IsValid(ginCtx)

	assert.False(t, isValid)
	assert.EqualError(t, err, "invalid jwt token provided")
}
