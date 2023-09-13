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

func NewJwtAuthHandler(config cfg.Config) gin.HandlerFunc {
	auth := NewJWTAuthAuthenticator(config)

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
	}
}

func NewJWTAuthAuthenticator(config cfg.Config) Authenticator {
	jwtTokenHandler := NewJwtTokenHandler(config)

	return NewJWTAuthAuthenticatorWithInterfaces(jwtTokenHandler)
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

	subject := &Subject{
		Name:            token.Claims.(jwt.MapClaims)["email"].(string),
		Anonymous:       false,
		AuthenticatedBy: ByJWT,
	}
	RequestWithSubject(ginCtx, subject)

	return true, nil
}
