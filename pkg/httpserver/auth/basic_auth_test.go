package auth_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/httpserver/auth"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getBasicAuthMocks(user string, password string) (log.Logger, *gin.Context) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)

	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, password)))))

	request := &http.Request{
		Header: header,
	}
	ginCtx := &gin.Context{
		Request: request,
	}

	ginCtx.Request = request

	return logger, ginCtx
}

func TestBasicAuth_Authenticate_InvalidUser(t *testing.T) {
	logger, ginCtx := getBasicAuthMocks("user", "password")

	a := auth.NewBasicAuthAuthenticatorWithInterfaces(logger, map[string]string{
		"other user": "other password",
	})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "invalid credentials provided", err.Error())
	}
}

func TestBasicAuth_Authenticate_InvalidPassword(t *testing.T) {
	logger, ginCtx := getBasicAuthMocks("user", "password")

	a := auth.NewBasicAuthAuthenticatorWithInterfaces(logger, map[string]string{
		"user": "other password",
	})
	_, err := a.IsValid(ginCtx)

	if assert.Error(t, err) {
		assert.Equal(t, "invalid credentials provided", err.Error())
	}
}

func TestBasicAuth_Authenticate_ValidUser(t *testing.T) {
	logger, ginCtx := getBasicAuthMocks("user", "password")

	a := auth.NewBasicAuthAuthenticatorWithInterfaces(logger, map[string]string{
		"user": "password",
	})
	_, err := a.IsValid(ginCtx)

	assert.Equal(t, nil, err)
}

func TestBasicAuth_Authenticate_ValidUser_SubjectHasCorrectFields(t *testing.T) {
	logger, ginCtx := getBasicAuthMocks("alice", "correctpass")

	a := auth.NewBasicAuthAuthenticatorWithInterfaces(logger, map[string]string{
		"alice": "correctpass",
	})
	ok, err := a.IsValid(ginCtx)

	assert.True(t, ok)
	assert.NoError(t, err)

	subj := auth.GetSubject(ginCtx.Request.Context())
	assert.Equal(t, "anon", subj.Name)
	assert.True(t, subj.Anonymous)
	assert.Equal(t, auth.ByBasicAuth, subj.AuthenticatedBy)
}

func captureAuthConfigDebug(t *testing.T, config cfg.Config) string {
	t.Helper()

	var output bytes.Buffer
	handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterJson, "main", time.RFC3339, &output)
	logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
	require.NoError(t, cfg.DebugConfig(t.Context(), config, logger))

	return output.String()
}

func authConfigUnrelatedSettings() map[string]any {
	return map[string]any{
		"public": "visible",
		"x[0]":   "literal-bracket",
		"x":      []string{"sequence-value"},
		"a.b":    "literal-dot",
		"a":      map[string]any{"b": "nested-value"},
	}
}

func requireAuthConfigPreservesUnrelatedSettings(t *testing.T, config cfg.Config, credentialKey string) {
	t.Helper()

	settings := make(map[string]any)
	for key, value := range config.AllSettings() {
		if key != credentialKey {
			settings[key] = value
		}
	}
	require.Equal(t, authConfigUnrelatedSettings(), settings)
}

func TestBasicAuthConfiguredCredentials(t *testing.T) {
	for _, entries := range []any{
		[]string{"", " alice:old ", " alice:synthetic-password:with-colon "},
		[]any{"", " alice:old ", " alice:synthetic-password:with-colon "},
		", alice:old , alice:synthetic-password:with-colon ",
	} {
		t.Run(fmt.Sprintf("%T", entries), func(t *testing.T) {
			settings := authConfigUnrelatedSettings()
			settings["api_auth_basic_users"] = entries
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)
			logger, ginCtx := getBasicAuthMocks("alice", "synthetic-password:with-colon")
			authenticator, err := auth.NewBasicAuthAuthenticator(config, logger)
			require.NoError(t, err)
			requireAuthConfigPreservesUnrelatedSettings(t, config, "api_auth_basic_users")
			valid, err := authenticator.IsValid(ginCtx)
			require.NoError(t, err)
			require.True(t, valid)
			subject := auth.GetSubject(ginCtx.Request.Context())
			require.Equal(t, "alice", subject.Attributes[auth.AttributeUser])
			require.Equal(t, auth.ByBasicAuth, subject.AuthenticatedBy)
			require.True(t, subject.Anonymous)

			output := captureAuthConfigDebug(t, config)
			require.NotContains(t, output, "synthetic-password")
			require.Contains(t, output, "visible")
			stored, err := config.GetStringSlice("api_auth_basic_users")
			require.NoError(t, err)
			require.Equal(t, []string{"", "alice:old", "alice:synthetic-password:with-colon"}, stored)
		})
	}
}

func TestBasicAuthMissingAndEmptyConfiguration(t *testing.T) {
	logger, ginCtx := getBasicAuthMocks("alice", "password")
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{})
	_, err := auth.NewBasicAuthAuthenticator(config, logger)
	require.Error(t, err)

	config = cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"api_auth_basic_users": []string{}})
	authenticator, err := auth.NewBasicAuthAuthenticator(config, logger)
	require.NoError(t, err)
	valid, err := authenticator.IsValid(ginCtx)
	require.Error(t, err)
	require.False(t, valid)
}

func TestBasicAuthMalformedCredentialsDoNotLeak(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"api_auth_basic_users": []string{"alice:valid-password", "synthetic-malformed-private"},
	})
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)
	_, err := auth.NewBasicAuthAuthenticator(config, logger)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "synthetic-malformed-private")
	require.Contains(t, err.Error(), "api_auth_basic_users[1]")
	require.NotContains(t, captureAuthConfigDebug(t, config), "synthetic-malformed-private")
}

func TestBasicAuthConfiguredLiteralCredentials(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"api_auth_basic_users": []string{"!nodecode alice:{literal}"},
		"literal":              "must-not-replace",
	})
	logger, ginCtx := getBasicAuthMocks("alice", "{literal}")
	authenticator, err := auth.NewBasicAuthAuthenticator(config, logger)
	require.NoError(t, err)
	valid, err := authenticator.IsValid(ginCtx)
	require.NoError(t, err)
	require.True(t, valid)
}

func TestBasicAuthEnvironmentCredentials(t *testing.T) {
	for _, settings := range []map[string]any{
		{},
		{"api_auth_basic_users": []string{"alice:old-password"}},
	} {
		for key, value := range authConfigUnrelatedSettings() {
			settings[key] = value
		}
		env := cfg.NewMemoryEnvProvider()
		require.NoError(t, env.SetEnv("API_AUTH_BASIC_USERS", "alice:synthetic-env-password"))
		config := cfg.NewWithInterfaces(env, settings)
		require.NoError(t, config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer)))
		logger, ginCtx := getBasicAuthMocks("alice", "synthetic-env-password")
		authenticator, err := auth.NewBasicAuthAuthenticator(config, logger)
		require.NoError(t, err)
		requireAuthConfigPreservesUnrelatedSettings(t, config, "api_auth_basic_users")
		valid, err := authenticator.IsValid(ginCtx)
		require.NoError(t, err)
		require.True(t, valid)
		require.NotContains(t, captureAuthConfigDebug(t, config), "synthetic-env-password")
		require.NotContains(t, config.AllSettings(), "")
	}
}
