package auth

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	ByAnonymous     = "anonymous"
	ByApiKey        = "apiKey"
	HeaderApiKey    = "X-API-KEY"
	configApiKeys   = "api_auth_keys"
	AttributeApiKey = "apiKey"
)

type configKeyAuthenticator struct {
	logger   log.Logger
	keys     []string
	provider ApiKeyProvider
}

type ApiKeyProvider func(ginCtx *gin.Context) string

func NewConfigKeyHandler(config cfg.Config, logger log.Logger, provider ApiKeyProvider) (gin.HandlerFunc, error) {
	auth, err := NewConfigKeyAuthenticator(config, logger, provider)
	if err != nil {
		return nil, fmt.Errorf("could not create config key authenticator: %w", err)
	}

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
	}, nil
}

func NewConfigKeyAuthenticator(config cfg.Config, logger log.Logger, provider ApiKeyProvider) (Authenticator, error) {
	keys, err := config.GetStringSlice(configApiKeys)
	if err != nil {
		return nil, fmt.Errorf("could not get string slice from config: %w", err)
	}

	keys = funk.Filter(keys, func(key string) bool {
		return key != ""
	})

	return NewConfigKeyAuthenticatorWithInterfaces(logger, keys, provider), nil
}

func NewConfigKeyAuthenticatorWithInterfaces(logger log.Logger, keys []string, provider ApiKeyProvider) Authenticator {
	return &configKeyAuthenticator{
		logger:   logger,
		keys:     keys,
		provider: provider,
	}
}

func (a *configKeyAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	apiKey := a.provider(ginCtx)

	if apiKey == "" {
		return false, fmt.Errorf("no api key provided")
	}

	if len(a.keys) == 0 {
		return false, fmt.Errorf("there are no api keys configured")
	}

	if !slices.Contains(a.keys, apiKey) {
		return false, fmt.Errorf("api key does not match")
	}

	user := &Subject{
		Name:            Anonymous,
		Anonymous:       true,
		AuthenticatedBy: ByApiKey,
		Attributes: map[string]any{
			AttributeApiKey: apiKey,
		},
	}

	RequestWithSubject(ginCtx, user)

	return true, nil
}

func ProvideValueFromQueryParam(queryParam string) ApiKeyProvider {
	return func(ginCtx *gin.Context) string {
		return ginCtx.Query(queryParam)
	}
}

func ProvideValueFromHeader(header string) ApiKeyProvider {
	return func(ginCtx *gin.Context) string {
		return ginCtx.GetHeader(header)
	}
}

func ProvideValueFromUriPath(param string) ApiKeyProvider {
	return func(ginCtx *gin.Context) string {
		return ginCtx.Param(param)
	}
}
