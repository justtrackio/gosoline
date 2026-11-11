package httpserver_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLoggingPrivacyBoundary(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var output bytes.Buffer
			handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityDebug, log.FormatterJson, "main", time.RFC3339, &output)
			logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
			router := gin.New()
			router.Use(httpserver.NewLoggingMiddlewareWithInterfaces(logger, httpserver.LoggingSettings{
				RequestQueryParameters: []string{"limit", "password"},
				RequestHeaders:         []string{"Authorization", "X-Api-Key"},
				RequestReferer:         true,
				RequestBody:            true,
				RequestBodyFields:      []string{"safe", "password"},
			}, clock.Provider))
			body := `{"safe":"diagnostic","password":"body-secret-473","unlisted":"private-body-473"}`
			router.POST("/auth/:id", func(c *gin.Context) {
				got, err := io.ReadAll(c.Request.Body)
				require.NoError(t, err)
				assert.Equal(t, body, string(got))
				assert.Equal(t, "query-secret-473", c.Query("password"))
				c.Status(status)
			})
			req := httptest.NewRequest(http.MethodPost, "https://example.test/auth/path-secret-473?limit=10&password=query-secret-473&unlisted=private-query-473", strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer bearer-secret-473")
			req.Header.Set("X-Api-Key", "api-secret-473")
			req.Header.Set("Referer", "https://user:pass@example.test/form?token=referer-secret-473#fragment-secret-473")
			router.ServeHTTP(httptest.NewRecorder(), req)
			for _, secret := range []string{"body-secret-473", "private-body-473", "query-secret-473", "private-query-473", "bearer-secret-473", "api-secret-473", "referer-secret-473", "fragment-secret-473", "path-secret-473"} {
				assert.NotContains(t, output.String(), secret)
			}
			assert.Contains(t, output.String(), "/auth/:id")
			assert.Contains(t, output.String(), "diagnostic")
		})
	}
}

func TestRawBodyLoggingNeedsAllowlist(t *testing.T) {
	var output bytes.Buffer
	handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityDebug, log.FormatterJson, "main", time.RFC3339, &output)
	logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
	router := gin.New()
	router.Use(httpserver.NewLoggingMiddlewareWithInterfaces(logger, httpserver.LoggingSettings{RequestBody: true}, clock.Provider))
	router.POST("/auth", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/auth?token=query-secret-473", strings.NewReader("opaque-secret-473")))
	assert.NotContains(t, output.String(), "opaque-secret-473")
	assert.NotContains(t, output.String(), "query-secret-473")
}

func TestSelectedHeaderPrivacy(t *testing.T) {
	for _, sampled := range []bool{true, false} {
		t.Run(fmt.Sprint(sampled), func(t *testing.T) {
			var output bytes.Buffer
			handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityDebug, log.FormatterJson, "main", time.RFC3339, &output)
			logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
			require.NoError(t, logger.Option(log.WithSamplingEnabled(true)))
			require.NoError(t, logger.Option(log.WithContextFieldsResolver(log.ContextFieldsResolver)))
			selected := []string{"X-Session-Id", "Proxy-Authorization", "x-id-token", "X-CSRF-Token", "X-Auth-Token", "X-Amz-Security-Token", "X-Custom-Credential", "Referer", "Location", "Content-Location"}
			settings := httpserver.LoggingSettings{RequestHeaders: append(selected, "X-Frontend-Version", "Content-Type")}
			router := gin.New()
			router.Use(httpserver.NewLoggingMiddlewareWithInterfaces(logger, settings, clock.Provider))
			router.GET("/headers", func(c *gin.Context) {
				for _, name := range selected {
					assert.Contains(t, c.GetHeader(name), "header-private-473")
				}
				assert.Equal(t, "frontend-v42", c.GetHeader("X-Frontend-Version"))
				c.Status(http.StatusBadRequest)
			})
			req := httptest.NewRequest(http.MethodGet, "/headers", http.NoBody)
			req = req.WithContext(smplctx.WithSampling(req.Context(), smplctx.Sampling{Sampled: sampled}))
			for _, name := range selected {
				req.Header.Set(name, "https://user:header-private-473@example.test/path?value=header-private-473#header-private-473")
				req.Header.Add(name, "second-header-private-473")
			}
			req.Header.Set("X-Frontend-Version", "frontend-v42")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Unselected", "unselected-private-473")
			router.ServeHTTP(httptest.NewRecorder(), req)
			assert.NotContains(t, output.String(), "header-private-473")
			assert.NotContains(t, output.String(), "unselected-private-473")
			assert.Contains(t, output.String(), "frontend-v42")
			assert.Contains(t, output.String(), "application/json")
			assert.Contains(t, output.String(), "[REDACTED]")
		})
	}
}

func TestProtectedRequestErrorBranches(t *testing.T) {
	for _, tc := range []struct {
		name       string
		kind       gin.ErrorType
		cause      error
		diagnostic string
	}{
		{"canceled", gin.ErrorTypePrivate, context.Canceled, "request canceled"},
		{"connection", gin.ErrorTypePrivate, io.EOF, "connection error"},
		{"bind", gin.ErrorTypeBind, errors.New("bad input"), "bind error"},
		{"render", gin.ErrorTypeRender, errors.New("render failed"), "render error"},
		{"validation", gin.ErrorTypePrivate, validation.NewError(errors.New("invalid field")), "validation error"},
		{"default", gin.ErrorTypePrivate, errors.New("provider failed"), "provider failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityDebug, log.FormatterJson, "main", time.RFC3339, &output)
			logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
			require.NoError(t, logger.Option(log.WithSamplingEnabled(true)))
			router := gin.New()
			router.Use(httpserver.NewLoggingMiddlewareWithInterfaces(logger, httpserver.LoggingSettings{}, clock.Provider))
			protected := log.RedactError(tc.cause, "error-private-473")
			wrapped := fmt.Errorf("operation error-private-473: %w", protected)
			router.GET("/errors/:id", func(c *gin.Context) {
				require.Error(t, c.Error(wrapped).SetType(tc.kind))
				c.Status(http.StatusBadRequest)
			})
			req := httptest.NewRequest(http.MethodGet, "/errors/path-private-473", http.NoBody)
			req = req.WithContext(smplctx.WithSampling(req.Context(), smplctx.Sampling{Sampled: false}))
			router.ServeHTTP(httptest.NewRecorder(), req)
			assert.NotContains(t, output.String(), "error-private-473")
			assert.NotContains(t, output.String(), "path-private-473")
			assert.Contains(t, output.String(), tc.diagnostic)
			assert.Contains(t, output.String(), "/errors/:id")
			assert.Contains(t, wrapped.Error(), "error-private-473")
			assert.ErrorIs(t, wrapped, tc.cause)
		})
	}
}

type privacyFailingReader struct{ err error }

func (r privacyFailingReader) Read([]byte) (int, error) { return 0, r.err }

func TestProtectedBodyReadError(t *testing.T) {
	var output bytes.Buffer
	handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityDebug, log.FormatterJson, "main", time.RFC3339, &output)
	logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
	require.NoError(t, logger.Option(log.WithSamplingEnabled(true)))
	router := gin.New()
	router.Use(httpserver.NewLoggingMiddlewareWithInterfaces(logger, httpserver.LoggingSettings{RequestBody: true, RequestBodyFields: []string{"operation"}}, clock.Provider))
	router.POST("/read", func(c *gin.Context) { c.Status(http.StatusBadRequest) })
	cause := log.RedactError(io.ErrUnexpectedEOF, "read-private-473")
	req := httptest.NewRequest(http.MethodPost, "/read", privacyFailingReader{fmt.Errorf("read-private-473: %w", cause)})
	req = req.WithContext(smplctx.WithSampling(req.Context(), smplctx.Sampling{Sampled: false}))
	router.ServeHTTP(httptest.NewRecorder(), req)
	assert.Contains(t, output.String(), "can not read request body")
	assert.NotContains(t, output.String(), "read-private-473")
}
