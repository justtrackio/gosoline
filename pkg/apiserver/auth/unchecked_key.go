package auth

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type uncheckedKeyAuthenticator struct {
	logger   log.Logger
	provider ApiKeyProvider
}

func NewUncheckedKeyHandler(config cfg.Config, logger log.Logger, provider ApiKeyProvider) gin.HandlerFunc {
	auth := NewUncheckedKeyAuthenticator(config, logger, provider)

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

func NewUncheckedKeyAuthenticator(_ cfg.Config, logger log.Logger, provider ApiKeyProvider) Authenticator {
	return NewUncheckedKeyAuthenticatorWithInterfaces(logger, provider)
}

func NewUncheckedKeyAuthenticatorWithInterfaces(logger log.Logger, provider ApiKeyProvider) Authenticator {
	return &uncheckedKeyAuthenticator{
		logger:   logger,
		provider: provider,
	}
}

func (a *uncheckedKeyAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	apiKey := a.provider(ginCtx)

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
