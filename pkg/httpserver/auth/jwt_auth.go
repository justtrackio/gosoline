package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

const (
	ByJWT         = "jwtAuth"
	headerJwtAuth = "Authorization"
)

type jwtAuthenticator struct {
	jwtTokenHandler JwtTokenHandler
}

func NewJwtAuthHandler(config cfg.Config, name string) (gin.HandlerFunc, error) {
	auth, err := NewJWTAuthAuthenticator(config, name)
	if err != nil {
		return nil, fmt.Errorf("can not create jwt authenticator for %s: %w", name, err)
	}

	return func(ginCtx *gin.Context) {
		valid, err := auth.IsValid(ginCtx)

		if valid {
			return
		}

		if err == nil {
			err = fmt.Errorf("the user jwt token isn't valid nor was there an error")
		}

		ginCtx.JSON(http.StatusUnauthorized, gin.H{"err": err.Error()})
		ginCtx.Abort()
	}, nil
}

func NewJWTAuthAuthenticator(config cfg.Config, name string) (Authenticator, error) {
	jwtTokenHandler, err := NewJwtTokenHandler(config, name)
	if err != nil {
		return nil, fmt.Errorf("can not create jwt token handler for authenticator %s: %w", name, err)
	}

	return NewJWTAuthAuthenticatorWithInterfaces(jwtTokenHandler), nil
}

func NewJWTAuthAuthenticatorWithInterfaces(jwtTokenHandler JwtTokenHandler) Authenticator {
	return &jwtAuthenticator{
		jwtTokenHandler: jwtTokenHandler,
	}
}

func (a *jwtAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	bearerAuth := ginCtx.GetHeader(headerJwtAuth)

	if bearerAuth == "" {
		return false, fmt.Errorf("no credentials provided")
	}

	if !strings.HasPrefix(bearerAuth, "Bearer ") {
		return false, fmt.Errorf("could not find jwt token in header")
	}

	jwtToken := bearerAuth[len("Bearer "):]

	isValid, token, err := a.jwtTokenHandler.Valid(jwtToken)
	if err != nil {
		return false, fmt.Errorf("error while validating jwt token: %w", err)
	}

	if !isValid {
		return false, fmt.Errorf("invalid jwt token provided")
	}

	email, ok := token.Claims.(jwt.MapClaims)["email"].(string)
	if !ok || email == "" {
		return false, fmt.Errorf("jwt token is missing email field")
	}

	subject := &Subject{
		Name:            email,
		Anonymous:       false,
		AuthenticatedBy: ByJWT,
	}
	RequestWithSubject(ginCtx, subject)

	return true, nil
}
