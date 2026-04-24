package auth

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cache"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/pkg/errors"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

const ByGoogle = "google"

//go:generate go run github.com/vektra/mockery/v2 --name TokenInfoProvider
type TokenInfoProvider interface {
	GetTokenInfo(string) (*oauth2.Tokeninfo, error)
}

type GoogleTokenProvider struct {
	oauth2Service *oauth2.Service
}

func (p *GoogleTokenProvider) GetTokenInfo(idToken string) (*oauth2.Tokeninfo, error) {
	return p.oauth2Service.Tokeninfo().IdToken(idToken).Do()
}

const (
	tokenCacheTTL       = 5 * time.Minute
	tokenCacheMaxSize   = 10_000
	tokenCachePruneSize = 500
)

type configGoogleAuthenticator struct {
	logger           log.Logger
	tokenCache       cache.Cache[oauth2.Tokeninfo]
	tokenProvider    TokenInfoProvider
	validAudiences   []string
	allowedAddresses []string
}

func NewConfigGoogleHandler(config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	auth, err := NewConfigGoogleAuthenticator(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create configGoogleAuthenticator: %w", err)
	}

	return func(ginCtx *gin.Context) {
		valid, err := auth.IsValid(ginCtx)

		if valid {
			return
		}

		if err == nil {
			err = fmt.Errorf("the google token wasn't valid nor was there an error")
		}

		ginCtx.JSON(http.StatusUnauthorized, gin.H{"err": err.Error()})
		ginCtx.Abort()
	}, nil
}

func NewConfigGoogleAuthenticator(config cfg.Config, logger log.Logger) (Authenticator, error) {
	// it will never be used, because we specify a http client here already
	ctx := context.Background()
	clientOption := option.WithHTTPClient(http.DefaultClient)

	oauth2Service, err := oauth2.NewService(ctx, clientOption)
	if err != nil {
		return nil, fmt.Errorf("failed creating google oauth2 client: %w", err)
	}

	tokenProvider := &GoogleTokenProvider{
		oauth2Service: oauth2Service,
	}

	clientIds, err := config.GetStringSlice("api_auth_google_client_ids")
	if err != nil {
		return nil, fmt.Errorf("failed getting google client ids: %w", err)
	}

	allowedAddresses, err := config.GetStringSlice("api_auth_google_allowed_addresses")
	if err != nil {
		return nil, fmt.Errorf("failed getting google allowed addresses: %w", err)
	}

	return NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, clientIds, allowedAddresses), nil
}

func NewConfigGoogleAuthenticatorWithInterfaces(logger log.Logger, tokenProvider TokenInfoProvider, clientIds []string, allowedAddresses []string) Authenticator {
	tokenCache := cache.New[oauth2.Tokeninfo](tokenCacheMaxSize, tokenCachePruneSize, tokenCacheTTL)

	return &configGoogleAuthenticator{
		logger:           logger,
		validAudiences:   clientIds,
		allowedAddresses: allowedAddresses,
		tokenCache:       tokenCache,
		tokenProvider:    tokenProvider,
	}
}

func (a *configGoogleAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	idToken := ginCtx.GetHeader("X-ID-TOKEN")

	if idToken == "" {
		return false, fmt.Errorf("google auth: zero length token")
	}

	reqCtx := ginCtx.Request.Context()

	tokenInfo, err := a.tokenCache.ProvideWithError(idToken, func() (oauth2.Tokeninfo, error) {
		return a.fetchAndValidateToken(reqCtx, idToken)
	})
	if err != nil {
		return false, err
	}

	user := a.getSubjectForToken(&tokenInfo)
	RequestWithSubject(ginCtx, user)

	return true, nil
}

func (a *configGoogleAuthenticator) fetchAndValidateToken(ctx context.Context, idToken string) (oauth2.Tokeninfo, error) {
	a.logger.Info(ctx, "token not in cache, will perform request")

	tokenInfo, err := a.tokenProvider.GetTokenInfo(idToken)
	if err != nil {
		return oauth2.Tokeninfo{}, errors.Wrap(err, "google auth: failed requesting token info")
	}

	if tokenInfo.HTTPStatusCode > 299 {
		return oauth2.Tokeninfo{}, fmt.Errorf("google auth: invalid status code %d", tokenInfo.HTTPStatusCode)
	}

	if !slices.Contains(a.validAudiences, tokenInfo.Audience) {
		return oauth2.Tokeninfo{}, fmt.Errorf("google auth: invalid audience")
	}

	ok, err := a.isAddressAllowed(tokenInfo.Email)
	if err != nil {
		return oauth2.Tokeninfo{}, fmt.Errorf("google auth: can not check if address is allowed")
	}

	if !ok {
		return oauth2.Tokeninfo{}, fmt.Errorf("google auth: address %s is not allowed", tokenInfo.Email)
	}

	return *tokenInfo, nil
}

func (a *configGoogleAuthenticator) isAddressAllowed(address string) (bool, error) {
	for _, allowed := range a.allowedAddresses {
		ok, err := regexp.MatchString("^(?:"+allowed+")$", address)
		if err != nil {
			return false, fmt.Errorf("can not compile regex for allowed address check: %w", err)
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

func (a *configGoogleAuthenticator) getSubjectForToken(tokenInfo *oauth2.Tokeninfo) *Subject {
	return &Subject{
		Name:            tokenInfo.Email,
		Anonymous:       false,
		AuthenticatedBy: ByGoogle,
	}
}
