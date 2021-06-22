package auth_test

import (
	"encoding/base64"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver/auth"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/log/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func getBasicAuthMocks(user string, password string) (log.Logger, *gin.Context) {
	logger := mocks.NewLoggerMockedAll()

	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, password)))))

	request := &http.Request{
		Header: header,
	}
	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return logger, ginCtx
}

func TestBasicAuth_Authenticate_InvalidUser(t *testing.T) {
	logger, ginCtx := getBasicAuthMocks("user", "password")

	a := auth.NewBasicAuthAuthenticatorWithInterfaces(logger, map[string]string{
		"other user": "other password",
	})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "invalid credentials provided", err.Error())
	}
}

func TestBasicAuth_Authenticate_InvalidPassword(t *testing.T) {
	logger, ginCtx := getBasicAuthMocks("user", "password")

	a := auth.NewBasicAuthAuthenticatorWithInterfaces(logger, map[string]string{
		"user": "other password",
	})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "invalid credentials provided", err.Error())
	}
}

func TestBasicAuth_Authenticate_ValidUser(t *testing.T) {
	logger, ginCtx := getBasicAuthMocks("user", "password")

	a := auth.NewBasicAuthAuthenticatorWithInterfaces(logger, map[string]string{
		"user": "password",
	})
	_, err := a.IsValid(ginCtx)

	assert.Equal(t, nil, err)
}
