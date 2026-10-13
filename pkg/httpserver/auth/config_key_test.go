package auth_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver/auth"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getHeaderKeyMocks(idToken string) (log.Logger, *gin.Context) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)

	header := http.Header{}
	header.Set("X-API-KEY", idToken)

	request := &http.Request{
		Header: header,
	}
	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return logger, ginCtx
}

func getQueryKeyMocks(idToken string) (log.Logger, *gin.Context) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)

	request := &http.Request{
		URL: &url.URL{
			RawQuery: fmt.Sprintf("apiKey=%s", idToken),
		},
	}

	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return logger, ginCtx
}

func TestAuthKeyInHeader_Authenticate_InvalidKeyError(t *testing.T) {
	logger, ginCtx := getHeaderKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"a"}, auth.ProvideValueFromHeader(auth.HeaderApiKey))
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "api key does not match", err.Error())
	}
}

func TestAuthKeyInHeader_Authenticate_ValidKey(t *testing.T) {
	logger, ginCtx := getHeaderKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"t"}, auth.ProvideValueFromHeader(auth.HeaderApiKey))
	_, err := a.IsValid(ginCtx)

	assert.Equal(t, nil, err)
}

func TestAuthKeyInQueryParam_Authenticate_InvalidKeyError(t *testing.T) {
	logger, ginCtx := getQueryKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"a"}, auth.ProvideValueFromQueryParam(auth.ByApiKey))
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "api key does not match", err.Error())
	}
}

func TestAuthKeyInQueryParam_Authenticate_ValidKey(t *testing.T) {
	logger, ginCtx := getQueryKeyMocks("t")

	a := auth.NewConfigKeyAuthenticatorWithInterfaces(logger, []string{"t"}, auth.ProvideValueFromQueryParam(auth.ByApiKey))
	_, err := a.IsValid(ginCtx)

	assert.Equal(t, nil, err)
}

func TestConfigKeyConfiguredCredentials(t *testing.T) {
	for _, entries := range []any{
		[]string{"", " synthetic-api-key "},
		[]any{"", " synthetic-api-key "},
		", synthetic-api-key ",
	} {
		t.Run(fmt.Sprintf("%T", entries), func(t *testing.T) {
			settings := authConfigUnrelatedSettings()
			settings["api_auth_keys"] = entries
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)
			logger, ginCtx := getHeaderKeyMocks("synthetic-api-key")
			authenticator, err := auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader(auth.HeaderApiKey))
			require.NoError(t, err)
			requireAuthConfigPreservesUnrelatedSettings(t, config, "api_auth_keys")
			valid, err := authenticator.IsValid(ginCtx)
			require.NoError(t, err)
			require.True(t, valid)
			subject := auth.GetSubject(ginCtx.Request.Context())
			require.Equal(t, "synthetic-api-key", subject.Attributes[auth.AttributeApiKey])
			require.Equal(t, auth.ByApiKey, subject.AuthenticatedBy)
			require.True(t, subject.Anonymous)

			output := captureAuthConfigDebug(t, config)
			require.NotContains(t, output, "synthetic-api-key")
			require.Contains(t, output, "visible")
			stored, err := config.GetStringSlice("api_auth_keys")
			require.NoError(t, err)
			require.Equal(t, []string{"", "synthetic-api-key"}, stored)
		})
	}
}

func TestConfigKeyMissingAndEmptyConfiguration(t *testing.T) {
	logger, ginCtx := getHeaderKeyMocks("synthetic-api-key")
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{})
	_, err := auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader(auth.HeaderApiKey))
	require.Error(t, err)

	for _, entries := range [][]string{{}, {"", " "}} {
		config = cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"api_auth_keys": entries})
		authenticator, err := auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader(auth.HeaderApiKey))
		require.NoError(t, err)
		valid, err := authenticator.IsValid(ginCtx)
		require.Error(t, err)
		require.False(t, valid)
	}
}

func TestConfigKeyConfiguredLiteralCredentials(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"api_auth_keys": []string{"!nodecode {literal}"},
		"literal":       "must-not-replace",
	})
	logger, ginCtx := getHeaderKeyMocks("{literal}")
	authenticator, err := auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader(auth.HeaderApiKey))
	require.NoError(t, err)
	valid, err := authenticator.IsValid(ginCtx)
	require.NoError(t, err)
	require.True(t, valid)
}

func TestConfigKeyEnvironmentCredentials(t *testing.T) {
	for _, settings := range []map[string]any{
		{},
		{"api_auth_keys": []string{"old-key"}},
	} {
		for key, value := range authConfigUnrelatedSettings() {
			settings[key] = value
		}
		env := cfg.NewMemoryEnvProvider()
		require.NoError(t, env.SetEnv("API_AUTH_KEYS", "synthetic-env-key"))
		config := cfg.NewWithInterfaces(env, settings)
		require.NoError(t, config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer)))
		logger, ginCtx := getHeaderKeyMocks("synthetic-env-key")
		authenticator, err := auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader(auth.HeaderApiKey))
		require.NoError(t, err)
		requireAuthConfigPreservesUnrelatedSettings(t, config, "api_auth_keys")
		valid, err := authenticator.IsValid(ginCtx)
		require.NoError(t, err)
		require.True(t, valid)
		require.NotContains(t, captureAuthConfigDebug(t, config), "synthetic-env-key")
		require.NotContains(t, config.AllSettings(), "")
	}
}
