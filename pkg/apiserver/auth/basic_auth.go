package auth

import (
	"encoding/base64"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

const (
	ByBasicAuth = "basicAuth"

	headerBasicAuth = "Authorization"
	configBasicAuth = "api_auth_basic_users"
	AttributeUser   = "user"
)

type basicAuthAuthenticator struct {
	logger log.Logger
	users  map[string]string
}

func NewBasicAuthHandler(config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	auth, err := NewBasicAuthAuthenticator(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create basicAuthAuthenticator: %w", err)
	}

	appName := config.GetString("app_name")

	return func(ginCtx *gin.Context) {
		valid, err := auth.IsValid(ginCtx)

		if valid {
			return
		}

		if err == nil {
			err = fmt.Errorf("the user credentials weren't valid nor was there an error")
		}

		ginCtx.Header("www-authenticate", fmt.Sprintf("Basic realm=\"%s\"", appName))
		ginCtx.JSON(http.StatusUnauthorized, gin.H{"err": err.Error()})
		ginCtx.Abort()
	}, nil
}

func NewBasicAuthAuthenticator(config cfg.Config, logger log.Logger) (Authenticator, error) {
	userEntries := config.GetStringSlice(configBasicAuth)

	users := make(map[string]string)

	for _, user := range userEntries {
		if user == "" {
			continue
		}

		split := strings.SplitN(user, ":", 2)
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid basic auth credentials: %s", user)
		}

		users[split[0]] = split[1]
	}

	return NewBasicAuthAuthenticatorWithInterfaces(logger, users), nil
}

func NewBasicAuthAuthenticatorWithInterfaces(logger log.Logger, users map[string]string) Authenticator {
	return &basicAuthAuthenticator{
		logger: logger,
		users:  users,
	}
}

func (a *basicAuthAuthenticator) IsValid(ginCtx *gin.Context) (bool, error) {
	basicAuth := ginCtx.GetHeader(headerBasicAuth)

	if basicAuth == "" {
		return false, fmt.Errorf("no credentials provided")
	}

	if !strings.HasPrefix(basicAuth, "Basic ") {
		return false, fmt.Errorf("invalid credentials provided")
	}

	auth, err := base64.StdEncoding.DecodeString(basicAuth[6:])

	if err != nil {
		return false, err
	}

	split := strings.SplitN(string(auth), ":", 2)

	if len(split) != 2 {
		return false, fmt.Errorf("invalid credentials provided")
	}

	if password, ok := a.users[split[0]]; ok {
		if password != split[1] {
			return false, fmt.Errorf("invalid credentials provided")
		}

		user := &Subject{
			Name:            Anonymous,
			Anonymous:       true,
			AuthenticatedBy: ByBasicAuth,
			Attributes: map[string]interface{}{
				AttributeUser: split[0],
			},
		}

		RequestWithSubject(ginCtx, user)

		return true, nil

	}

	return false, fmt.Errorf("invalid credentials provided")
}
