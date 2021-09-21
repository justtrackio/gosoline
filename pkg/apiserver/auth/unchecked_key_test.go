package auth_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/apiserver/auth"
	"github.com/stretchr/testify/assert"
)

func TestUncheckedKey_Authenticate_InvalidKeyError(t *testing.T) {
	logger, ginCtx := getHeaderKeyMocks("")

	a := auth.NewUncheckedKeyAuthenticatorWithInterfaces(logger)
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "no api key provided", err.Error())
	}
}

func TestUncheckedKey_Authenticate_ValidKey(t *testing.T) {
	logger, ginCtx := getHeaderKeyMocks("t")

	a := auth.NewUncheckedKeyAuthenticatorWithInterfaces(logger)
	_, err := a.IsValid(ginCtx)

	ctx := ginCtx.Request.Context()
	sub := auth.GetSubject(ctx)

	assert.Equal(t, nil, err)
	assert.Equal(t, auth.Anonymous, sub.Name)
	assert.True(t, sub.Anonymous)
	assert.Equal(t, auth.ByApiKey, sub.AuthenticatedBy)
	assert.Equal(t, "t", sub.Attributes[auth.AttributeApiKey])
}
