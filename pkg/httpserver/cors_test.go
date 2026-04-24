package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCorsConfig(pattern string) cfg.Config {
	return cfg.New(map[string]any{
		"api_cors_allowed_origin_pattern": pattern,
		"api_cors_allowed_headers":        []string{"Content-Type"},
		"api_cors_allowed_methods":        []string{"GET", "POST"},
	})
}

// TestCors_AnchoredPattern_PreventsPartialMatch verifies that a pattern like
// `https://example\.com` does NOT match a suffix-extended origin such as
// `https://example.com.evil.com`. Without the ^(?:...)$ anchoring added by the
// fix, the unanchored regex would allow the partial match.
func TestCors_AnchoredPattern_PreventsPartialMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, err := httpserver.Cors(newCorsConfig(`https://example\.com`))
	require.NoError(t, err)

	router := gin.New()
	router.Use(handler)
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Suffix-extended origin must be rejected.
	req, err := http.NewRequest(http.MethodOptions, "/", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://example.com.evil.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"suffix-extended origin must not be allowed by anchored pattern")
}

// TestCors_AnchoredPattern_AllowsExactMatch verifies that the exact origin
// still matches after anchoring.
func TestCors_AnchoredPattern_AllowsExactMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, err := httpserver.Cors(newCorsConfig(`https://example\.com`))
	require.NoError(t, err)

	router := gin.New()
	router.Use(handler)
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Exact match must pass.
	req, err := http.NewRequest(http.MethodOptions, "/", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"),
		"exact origin must be allowed")
}

// TestCors_AnchoredPattern_PreventsPrefixBypass verifies that a prefix cannot
// be injected either (e.g. `evil.https://example.com` must not match).
func TestCors_AnchoredPattern_PreventsPrefixBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, err := httpserver.Cors(newCorsConfig(`https://example\.com`))
	require.NoError(t, err)

	router := gin.New()
	router.Use(handler)
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, err := http.NewRequest(http.MethodOptions, "/", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Origin", "evil.https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
		"prefix-extended origin must not be allowed by anchored pattern")
}
