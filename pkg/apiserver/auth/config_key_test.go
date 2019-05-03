package auth_test

import (
	"github.com/applike/gosoline/pkg/apiserver/auth"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func getKeyMocks(idToken string) (mon.Logger, *gin.Context) {
	logger := mocks.NewLoggerMockedAll()

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

func TestAuthKey_Authenticate_InvalidKeyError(t *testing.T) {
	logger, ginCtx := getKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"a"})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "api key does not match", err.Error())
	}
}

func TestAuthKey_Authenticate_ValidKey(t *testing.T) {
	logger, ginCtx := getKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"t"})
	_, err := a.IsValid(ginCtx)

	assert.Equal(t, nil, err)
}
