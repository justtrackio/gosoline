package httpserver_test

import (
	"bytes"
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
