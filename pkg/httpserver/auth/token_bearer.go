package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
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

// A Getter retrieves a value from the store by the given key. If the value does
// not exist, false is returned and value is not modified.
// value should be a pointer to the model you want to retrieve.
type Getter func(ctx context.Context, key string, value TokenBearer) (bool, error)

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

func (i InvalidTokenErr) As(target any) bool {
	_, ok := target.(InvalidTokenErr)

	// we don't have to write anything as our struct is always empty

	return ok
}

type InvalidBearerErr struct {
	Message string
}

func (i InvalidBearerErr) Error() string {
	if i.Message != "" {
		return i.Message
	}

	return "invalid bearer"
}

func (i InvalidBearerErr) As(target any) bool {
	err, ok := target.(*InvalidBearerErr)
	if ok {
		*err = i
	}

	return ok
}

type (
	TokenBearerProvider func(ctx context.Context, key string, token string) (TokenBearer, error)
	ModelProvider       func() TokenBearer
)

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

	if token == "" {
		return false, InvalidTokenErr{}
	}

	if bearerId == "" {
		return false, &InvalidBearerErr{
			Message: "bearer is empty",
		}
	}

	bearer, err := a.provider(ginCtx.Request.Context(), bearerId, token)
	if err != nil {
		// if the provider responds with an invalid bearer error, don't mask it, maybe they want to communicate
		// certain facts to the caller to help diagnose a problem (if their credentials are almost correct, and we can
		// assume that an attack on them is unlikely)
		bearerErr := InvalidBearerErr{}
		if errors.As(err, &bearerErr) {
			return false, bearerErr
		}

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
		Attributes: map[string]any{
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
		found, err := getter(ctx, key, m)

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
