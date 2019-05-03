package auth

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
	"github.com/thoas/go-funk"
	"net/http"
)

const (
	ByApiKey = "apiKey"

	headerApiKey    = "X-API-KEY"
	configApiKeys   = "api_auth_keys"
	AttributeApiKey = "apiKey"
)

type configKeyAuthenticator struct {
	logger mon.Logger
	keys   []string
}

func NewConfigKeyHandler(config cfg.Config, logger mon.Logger) gin.HandlerFunc {
	auth := NewConfigKeyAuthenticator(config, logger)

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

func NewConfigKeyAuthenticator(config cfg.Config, logger mon.Logger) Authenticator {
	keys := config.GetStringSlice(configApiKeys)
	keys = funk.FilterString(keys, func(key string) bool {
		return key != ""
	})

	return NewConfigKeyAuthenticatorWithInterfaces(logger, keys)
}

func NewConfigKeyAuthenticatorWithInterfaces(logger mon.Logger, keys []string) Authenticator {
	return &configKeyAuthenticator{
		logger: logger,
		keys:   keys,
	}
}

func (a *configKeyAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	apiKey := ginCtx.GetHeader(headerApiKey)

	if apiKey == "" {
		return false, fmt.Errorf("no api key provided")
	}

	if len(a.keys) == 0 {
		return false, fmt.Errorf("there are no api keys configured")
	}

	if !funk.ContainsString(a.keys, apiKey) {
		return false, fmt.Errorf("api key does not match")
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
