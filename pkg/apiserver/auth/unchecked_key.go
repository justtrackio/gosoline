package auth

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
	"net/http"
)

type uncheckedKeyAuthenticator struct {
	logger mon.Logger
}

func NewUncheckedKeyHandler(config cfg.Config, logger mon.Logger) gin.HandlerFunc {
	auth := NewUncheckedKeyAuthenticator(config, logger)

	return func(ginCtx *gin.Context) {
		valid, err := auth.IsValid(ginCtx)

		if valid {
			return
		}

		if err == nil {
			err = fmt.Errorf("the api key wasn't valid nor was there an error")
		}

		ginCtx.JSON(http.StatusUnauthorized, gin.H{"err": err.Error()})
		ginCtx.Abort()
	}
}

func NewUncheckedKeyAuthenticator(_ cfg.Config, logger mon.Logger) Authenticator {
	return NewUncheckedKeyAuthenticatorWithInterfaces(logger)
}

func NewUncheckedKeyAuthenticatorWithInterfaces(logger mon.Logger) Authenticator {
	return &uncheckedKeyAuthenticator{
		logger: logger,
	}
}

func (a *uncheckedKeyAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	apiKey := ginCtx.GetHeader(headerApiKey)

	if apiKey == "" {
		return false, fmt.Errorf("no api key provided")
	}

	user := &Subject{
		Name:            Anonymous,
		Anonymous:       true,
		AuthenticatedBy: ByApiKey,
		Attributes: map[string]interface{}{
			AttributeApiKey: apiKey,
		},
	}

	RequestWithSubject(ginCtx, user)

	return true, nil
}
