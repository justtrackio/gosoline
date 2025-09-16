package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/justtrackio/gosoline/pkg/httpserver/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getJwtToken(t *testing.T, issuer string, secret string, expirationDuration int) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &auth.JwtClaims{
		Name:  "testName",
		Email: "testMail",
		Image: "testImage",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expirationDuration) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
		},
	})

	tokenString, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	return tokenString
}

func TestJwtTokenHandler_Sign_Valid(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 15 * time.Minute,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	token, err := h.Sign(auth.SignUserInput{
		Name:  "test",
		Email: "mail",
		Image: "image",
	})

	assert.NoError(t, err)
	assert.NotNil(t, token)
}

func TestJwtTokenHandler_IsValid_Valid(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 15 * time.Minute,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	jwtToken := getJwtToken(t, "me", "1", 10000)

	isValid, _, err := h.Valid(jwtToken)

	assert.NoError(t, err)
	assert.True(t, isValid)
}

func TestJwtTokenHandler_IsValid_InvalidSigningSecret(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "123",
		Issuer:         "me",
		ExpireDuration: 15 * time.Minute,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	jwtToken := getJwtToken(t, "me", "1", 10000)

	isValid, _, err := h.Valid(jwtToken)

	assert.EqualError(t, err, "token signature is invalid: signature is invalid")
	assert.False(t, isValid)
}

func TestJwtTokenHandler_IsValid_InvalidExpiredToken(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 15 * time.Minute,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	jwtToken := getJwtToken(t, "me", "1", -1)

	isValid, _, err := h.Valid(jwtToken)

	assert.EqualError(t, err, "token has invalid claims: token is expired")
	assert.False(t, isValid)
}

func TestJwtTokenHandler_IsValid_InvalidIssuer(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 15 * time.Minute,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	jwtToken := getJwtToken(t, "me32424", "1", 10000)

	isValid, _, err := h.Valid(jwtToken)

	assert.EqualError(t, err, "invalid issuer")
	assert.False(t, isValid)
}

func TestJwtTokenHandler_Sign_IsValid_Valid(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 15 * time.Minute,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	token, err := h.Sign(auth.SignUserInput{
		Name:  "test",
		Email: "mail",
		Image: "image",
	})

	assert.NoError(t, err)

	isValid, jwtToken, err := h.Valid(*token)
	require.NoError(t, err)
	require.True(t, isValid)
	require.NotNil(t, token)

	claims := jwtToken.Claims.(jwt.MapClaims)

	assert.Equal(t, "test", claims["name"])
	assert.Equal(t, "mail", claims["email"])
	assert.Equal(t, "image", claims["image"])
}
