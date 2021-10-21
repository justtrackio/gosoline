package auth_test

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/justtrackio/gosoline/pkg/apiserver/auth"
	"github.com/stretchr/testify/assert"
)

func getJwtToken(issuer string, secret string, expirationDuration int) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &auth.JwtClaims{
		Name:  "testName",
		Email: "testMail",
		Image: "testImage",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expirationDuration) * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    issuer,
		},
	})

	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestJwtTokenHandler_Sign_Valid(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 10000,
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
		ExpireDuration: 10000,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	jwtToken := getJwtToken("me", "1", 10000)

	isValid, _, err := h.Valid(jwtToken)

	assert.NoError(t, err)
	assert.True(t, isValid)
}

func TestJwtTokenHandler_IsValid_InvalidSigningSecret(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "123",
		Issuer:         "me",
		ExpireDuration: 10000,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	jwtToken := getJwtToken("me", "1", 10000)

	isValid, _, err := h.Valid(jwtToken)

	assert.EqualError(t, err, "signature is invalid")
	assert.False(t, isValid)
}

func TestJwtTokenHandler_IsValid_InvalidExpiredToken(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 10000,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	jwtToken := getJwtToken("me", "1", -1)

	isValid, _, err := h.Valid(jwtToken)

	assert.EqualError(t, err, "Token is expired")
	assert.False(t, isValid)
}

func TestJwtTokenHandler_IsValid_InvalidIssuer(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 10000,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	jwtToken := getJwtToken("me32424", "1", 10000)

	isValid, _, err := h.Valid(jwtToken)

	assert.EqualError(t, err, "invalid issuer")
	assert.False(t, isValid)
}

func TestJwtTokenHandler_Sign_IsValid_Valid(t *testing.T) {
	settings := auth.JwtTokenHandlerSettings{
		SigningSecret:  "1",
		Issuer:         "me",
		ExpireDuration: 10000,
	}

	h := auth.NewJwtTokenHandlerWithInterfaces(settings)

	token, err := h.Sign(auth.SignUserInput{
		Name:  "test",
		Email: "mail",
		Image: "image",
	})

	assert.NoError(t, err)

	isValid, jwtToken, err := h.Valid(*token)

	claims := jwtToken.Claims.(jwt.MapClaims)

	assert.Equal(t, "test", claims["name"])
	assert.Equal(t, "mail", claims["email"])
	assert.Equal(t, "image", claims["image"])
	assert.NoError(t, err)
	assert.True(t, isValid)
	assert.NotNil(t, token)
}
