package auth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	ByTokenBearer           = "tokenBearer"
	configBearerIdHeader    = "api_auth_bearer_id_header"
	configBearerTokenHeader = "api_auth_bearer_token_header"
	AttributeToken          = "token"
	AttributeTokenBearerId  = "tokenBearerId"
	AttributeTokenBearer    = "tokenBearer"
)

type TokenBearer interface {
	GetToken() string
}

type Getter interface {
	// Retrieve a value from the store by the given key. If the value does
	// not exist, false is returned and value is not modified.
	// value should be a pointer to the model you want to retrieve.
	Get(ctx context.Context, key interface{}, value interface{}) (bool, error)
}

type tokenBearerAuthenticator struct {
	logger      log.Logger
	keyHeader   string
	tokenHeader string
	provider    TokenBearerProvider
}

type InvalidTokenErr struct{}

func (i InvalidTokenErr) Error() string {
	return "invalid token"
}

func (i InvalidTokenErr) Is(err error) bool {
	_, ok := err.(InvalidTokenErr)

	return ok
}

func (i InvalidTokenErr) As(target interface{}) bool {
	_, ok := target.(InvalidTokenErr)

	// we don't have to write anything as our struct is always empty

	return ok
}

type TokenBearerProvider func(ctx context.Context, key string, token string) (TokenBearer, error)
type ModelProvider func() TokenBearer

func NewTokenBearerHandler(config cfg.Config, logger log.Logger, provider TokenBearerProvider) gin.HandlerFunc {
	auth := NewTokenBearerAuthenticator(config, logger, provider)

	return func(ginCtx *gin.Context) {
		valid, err := auth.IsValid(ginCtx)

		if valid {
			return
		}

		if err == nil {
			err = fmt.Errorf("the token wasn't valid nor was there an error")
		}

		ginCtx.JSON(http.StatusUnauthorized, gin.H{"err": err.Error()})
		ginCtx.Abort()
	}
}

func NewTokenBearerAuthenticator(config cfg.Config, logger log.Logger, provider TokenBearerProvider) Authenticator {
	keyHeader := config.GetString(configBearerIdHeader)
	tokenHeader := config.GetString(configBearerTokenHeader)

	return NewTokenBearerAuthenticatorWithInterfaces(logger, keyHeader, tokenHeader, provider)
}

func NewTokenBearerAuthenticatorWithInterfaces(logger log.Logger, keyHeader string, tokenHeader string, provider TokenBearerProvider) Authenticator {
	return &tokenBearerAuthenticator{
		logger:      logger,
		keyHeader:   keyHeader,
		tokenHeader: tokenHeader,
		provider:    provider,
	}
}

func (a *tokenBearerAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	bearerId := ginCtx.GetHeader(a.keyHeader)
	token := ginCtx.GetHeader(a.tokenHeader)

	if len(bearerId) == 0 || len(token) == 0 {
		return false, InvalidTokenErr{}
	}

	bearer, err := a.provider(ginCtx.Request.Context(), bearerId, token)

	if err != nil {
		return false, InvalidTokenErr{}
	}

	if bearer == nil {
		return false, InvalidTokenErr{}
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(bearer.GetToken())) != 1 {
		return false, InvalidTokenErr{}
	}

	user := &Subject{
		Name:            Anonymous,
		Anonymous:       true,
		AuthenticatedBy: ByTokenBearer,
		Attributes: map[string]interface{}{
			AttributeToken:         token,
			AttributeTokenBearerId: bearerId,
			AttributeTokenBearer:   bearer,
		},
	}

	RequestWithSubject(ginCtx, user)

	return true, nil
}

func ProvideTokenBearerFromGetter(getter Getter, getModel ModelProvider) TokenBearerProvider {
	return func(ctx context.Context, key string, _ string) (TokenBearer, error) {
		m := getModel()
		found, err := getter.Get(ctx, key, m)

		if err == nil && found {
			return m, nil
		}

		return nil, err
	}
}

func ProvideTokenBearerFromDdb(repo ddb.Repository, getModel ModelProvider) TokenBearerProvider {
	return func(ctx context.Context, key string, _ string) (TokenBearer, error) {
		m := getModel()
		result, err := repo.GetItem(ctx, repo.GetItemBuilder().WithHash(key), m)

		if err == nil && result.IsFound {
			return m, nil
		}

		return nil, err
	}
}
