package log_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretRendering(t *testing.T) {
	secret := log.Secret("test-password-473")
	for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q", "%x"} {
		assert.Equal(t, log.Redacted, fmt.Sprintf(format, secret))
	}
	encoded, err := json.Marshal(secret)
	require.NoError(t, err)
	assert.Equal(t, `"[REDACTED]"`, string(encoded))
}

func TestRedactErrorPreservesControlFlowButDetachesExport(t *testing.T) {
	cause := errors.New("provider rejected credential=my-secret-473")
	protected := log.RedactError(cause, "my-secret-473")
	wrapped := fmt.Errorf("operation credential=my-secret-473: %w", protected)
	assert.ErrorIs(t, wrapped, cause)
	safe := log.SafeError(wrapped)
	assert.NotContains(t, safe.Error(), "my-secret-473")
	assert.Contains(t, safe.Error(), "provider rejected")
	assert.Nil(t, errors.Unwrap(safe))
	assert.NotContains(t, fmt.Sprintf("%#v", protected), "my-secret-473")
	assert.Nil(t, log.RedactError(nil, "my-secret-473"))
}

func TestRedactErrorEncodingsAndDiagnosticBoundaries(t *testing.T) {
	value := "test secret/+473"
	for _, form := range []string{
		value, url.QueryEscape(value), url.PathEscape(value),
		base64.StdEncoding.EncodeToString([]byte(value)), base64.RawURLEncoding.EncodeToString([]byte(value)),
	} {
		err := log.RedactError(errors.New("provider credential='"+form+"'"), value)
		assert.Equal(t, "provider credential='[REDACTED]'", err.Error())
	}
	err := log.RedactError(errors.New("Invalid refresh_token. credential=refresh_toke; password=x"), "refresh_toke", "x")
	assert.Equal(t, "Invalid refresh_token. credential=[REDACTED]; password=[REDACTED]", err.Error())
}

func TestSanitizeFieldsCopiesNestedCredentials(t *testing.T) {
	fields := log.Fields{"password": "password-473", "nested": log.Fields{"api_key": "key-473", "status": "refresh_token"}}
	safe := log.SanitizeFields(fields)
	assert.Equal(t, log.Redacted, safe["password"])
	assert.Equal(t, log.Redacted, safe["nested"].(map[string]any)["api_key"])
	assert.Equal(t, "refresh_token", safe["nested"].(map[string]any)["status"])
	assert.Equal(t, "password-473", fields["password"])
	cycle := map[string]any{}
	cycle["self"] = cycle
	_, err := json.Marshal(log.SanitizeFields(cycle))
	require.NoError(t, err)
}

func TestLoggerPrivacyAtSerializedBoundary(t *testing.T) {
	for name, formatter := range map[string]log.Formatter{"json": log.FormatterJson, "simple": log.FormatterSimple, "console": log.FormatterConsole} {
		t.Run(name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityDebug, formatter, "main", time.RFC3339, buf)
			logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
			err := log.RedactError(errors.New("rejected api-key-473"), "api-key-473")
			//nolint:nosecrets // Deliberately inject raw credentials to verify masking in serialized output.
			logger.WithFields(log.Fields{"password": "password-473", "headers": map[string]string{"Authorization": "Bearer bearer-473"}}).
				Error(t.Context(), "api-key-473 operation failed: %w", err)
			for _, secret := range []string{"password-473", "bearer-473", "api-key-473"} {
				assert.NotContains(t, buf.String(), secret)
			}
			assert.Contains(t, buf.String(), "operation failed")
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	assert.Equal(t, "https://example.test/login", log.SanitizeURL("https://user:password@example.test/login?api_key=secret#token"))
	assert.Equal(t, log.Redacted, log.SanitizeURL("https://host/%zz?password=secret"))
}

func TestErrorPrivacyDepthFailsClosed(t *testing.T) {
	err := log.RedactError(errors.New("secret-473"), "secret-473")
	for range 40 {
		err = fmt.Errorf("operation: %w", err)
	}
	assert.Equal(t, "error chain omitted: privacy depth limit", log.SafeError(err).Error())
}

func TestRequestHeaderForLog(t *testing.T) {
	headers := http.Header{}
	for _, name := range []string{"Accept", "Accept-Encoding", "Content-Type", "Content-Encoding", "User-Agent", "X-Frontend-Version", "X-Request-Id"} {
		headers.Set(name, "safe-diagnostic")
		assert.Equal(t, "safe-diagnostic", log.RequestHeaderForLog(headers, strings.ToLower(name)))
	}
	for _, name := range []string{"Authorization", "Proxy-Authorization", "X-ID-TOKEN", "X-Custom-Auth", "Referer", "Location", "Cookie", "Set-Cookie"} {
		headers.Set(name, "private-value-473")
		assert.Equal(t, log.Redacted, log.RequestHeaderForLog(headers, strings.ToLower(name)))
		assert.Equal(t, "private-value-473", headers.Get(name))
	}
	assert.Equal(t, log.Redacted, log.RequestHeaderForLog(headers, "X-Missing"))
	assert.Empty(t, log.RequestHeaderForLog(nil, "Content-Type"))
}
