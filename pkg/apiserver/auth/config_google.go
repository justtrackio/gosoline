package auth

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"google.golang.org/api/oauth2/v2"
	"net/http"
	"strings"
	"sync"
)

const ByGoogle = "google"

//go:generate mockery -name TokenInfoProvider
type TokenInfoProvider interface {
	GetTokenInfo(string) (*oauth2.Tokeninfo, error)
}

type GoogleTokenProvider struct {
	oauth2Service *oauth2.Service
}

func (p *GoogleTokenProvider) GetTokenInfo(idToken string) (*oauth2.Tokeninfo, error) {
	return p.oauth2Service.Tokeninfo().IdToken(idToken).Do()
}

type configGoogleAuthenticator struct {
	logger        mon.Logger
	tokenCache    map[string]*oauth2.Tokeninfo
	tokenProvider TokenInfoProvider
	mutex         sync.Mutex
	validAudience string
	hostedDomain  string
}

func NewConfigGoogleHandler(config cfg.Config, logger mon.Logger) gin.HandlerFunc {
	auth := NewConfigGoogleAuthenticator(config, logger)

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
	}
}

func NewConfigGoogleAuthenticator(config cfg.Config, logger mon.Logger) Authenticator {
	oauth2Service, err := oauth2.New(http.DefaultClient)

	if err != nil {
		logger.Panic(err, "failed creating google oauth2 client")
	}

	tokenProvider := &GoogleTokenProvider{
		oauth2Service: oauth2Service,
	}

	clientId := config.GetString("api_auth_google_client_id")
	hostedDomain := config.GetString("api_auth_google_hosted_domain")

	return NewConfigGoogleAuthenticatorWithInterfaces(logger, tokenProvider, clientId, hostedDomain)
}

func NewConfigGoogleAuthenticatorWithInterfaces(logger mon.Logger, tokenProvider TokenInfoProvider, clientId string, hostedDomain string) Authenticator {
	return &configGoogleAuthenticator{
		logger:        logger,
		validAudience: clientId,
		hostedDomain:  hostedDomain,
		tokenCache:    make(map[string]*oauth2.Tokeninfo),
		tokenProvider: tokenProvider,
	}
}

func (a *configGoogleAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	idToken := ginCtx.GetHeader("X-ID-TOKEN")

	if len(idToken) == 0 {
		return false, fmt.Errorf("google auth: zero length token")
	}

	reqCtx := ginCtx.Request.Context()
	logger := a.logger.WithContext(reqCtx)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if tokenInfo, ok := a.tokenCache[idToken]; ok && tokenInfo == nil {
		logger.Debug("token was in cache but invalid")

		return false, fmt.Errorf("token from cache invalidated the user")
	}

	if tokenInfo, ok := a.tokenCache[idToken]; ok {
		logger.Debug("idToken was already in cache and valid")

		user := a.getSubjectForToken(tokenInfo)
		RequestWithSubject(ginCtx, user)

		return true, nil
	}

	logger.WithFields(mon.Fields{
		"id_token": idToken,
	}).Info("token not in cache, will perform request")

	tokenInfo, err := a.tokenProvider.GetTokenInfo(idToken)

	if err != nil {
		a.tokenCache[idToken] = nil

		return false, errors.Wrap(err, "google auth: failed requesting token info")
	}

	if tokenInfo.HTTPStatusCode > 299 {
		a.tokenCache[idToken] = nil

		return false, fmt.Errorf("google auth: invalid status code %d", tokenInfo.HTTPStatusCode)
	}

	if tokenInfo.Audience != a.validAudience {
		a.tokenCache[idToken] = nil

		return false, fmt.Errorf("google auth: invalid audience")
	}

	check := strings.Builder{}
	check.WriteString("@")
	check.WriteString(a.hostedDomain)

	if !strings.HasSuffix(tokenInfo.Email, check.String()) {
		a.tokenCache[idToken] = nil

		return false, fmt.Errorf("google auth: invalid email suffix")
	}

	a.tokenCache[idToken] = tokenInfo

	user := a.getSubjectForToken(tokenInfo)
	RequestWithSubject(ginCtx, user)

	return true, nil
}

func (a *configGoogleAuthenticator) getSubjectForToken(tokenInfo *oauth2.Tokeninfo) *Subject {
	return &Subject{
		Name:            tokenInfo.Email,
		Anonymous:       false,
		AuthenticatedBy: ByGoogle,
	}
}
