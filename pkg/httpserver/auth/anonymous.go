package auth

import (
	"github.com/gin-gonic/gin"
)

type anonymousAuthenticator struct{}

func NewAnonymousAuthenticator() Authenticator {
	return NewAnonymousAuthenticatorWithInterfaces()
}

func NewAnonymousAuthenticatorWithInterfaces() Authenticator {
	return &anonymousAuthenticator{}
}

func (a *anonymousAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	user := &Subject{
		Name:            Anonymous,
		Anonymous:       true,
		AuthenticatedBy: ByAnonymous,
		Attributes:      map[string]interface{}{},
	}

	RequestWithSubject(ginCtx, user)

	return true, nil
}
