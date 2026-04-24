package httpserver_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readBodyHandler is a gin handler that reads the entire request body.
// On success it returns 200 with the body length; on read error it returns 413.
func readBodyHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"err": err.Error()})

		return
	}
	c.JSON(http.StatusOK, gin.H{"len": len(body)})
}

func newMaxBodyRouter(maxBytes int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(httpserver.MaxBodySizeMiddleware(maxBytes))
	router.POST("/", readBodyHandler)

	return router
}

func TestMaxBodySizeMiddleware_SmallBodyPassesThrough(t *testing.T) {
	router := newMaxBodyRouter(100)

	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"len":5`)
}

func TestMaxBodySizeMiddleware_OversizedBodyReturnsError(t *testing.T) {
	// Limit: 3 bytes; body: 11 bytes.
	router := newMaxBodyRouter(3)

	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader("hello world"))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code,
		"body exceeding the limit must cause an error")
}

func TestMaxBodySizeMiddleware_ZeroLimitDisablesEnforcement(t *testing.T) {
	// maxBytes == 0 means no limit.
	router := newMaxBodyRouter(0)

	bigBody := strings.Repeat("x", 1_000_000)
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(bigBody))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code,
		"no limit (maxBytes=0) must allow large bodies")
}

func TestMaxBodySizeMiddleware_ExactLimitBodyPassesThrough(t *testing.T) {
	// Body exactly at the limit must succeed.
	router := newMaxBodyRouter(5)

	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code,
		"body exactly at the limit must be accepted")
}

// TestMaxBodySizeMiddleware_GzipBombExceedsDecompressedLimit verifies that a
// gzip body that is small on the wire but expands beyond the configured limit
// after decompression is rejected. This mirrors the production middleware order:
// decompression runs first (replacing c.Request.Body with the decoded stream),
// then MaxBodySizeMiddleware wraps that stream with http.MaxBytesReader so the
// limit is applied to the decompressed byte count, not the compressed wire size.
func TestMaxBodySizeMiddleware_GzipBombExceedsDecompressedLimit(t *testing.T) {
	// Build a highly compressible payload via a loop (not a const/var).
	// 20 000 iterations × 30 bytes = 600 000 bytes decompressed; compressed to
	// a few hundred bytes because the input is entirely repetitive ASCII.
	var compressed bytes.Buffer
	gw := gzip.NewWriter(&compressed)
	for i := 0; i < 20_000; i++ {
		_, err := gw.Write([]byte("AAAAAAAAAABBBBBBBBBBCCCCCCCCCC"))
		require.NoError(t, err)
	}
	require.NoError(t, gw.Close())

	// 100 000-byte limit: well above the compressed wire size (~hundreds of
	// bytes) but well below the 600 000-byte decompressed payload.
	const limit int64 = 100_000

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Decompression middleware — mirrors decompressionFn in compression.go.
	// Replaces c.Request.Body with the decoded stream before MaxBodySizeMiddleware
	// wraps it, so MaxBytesReader counts decompressed bytes.
	router.Use(func(c *gin.Context) {
		if c.GetHeader("Content-Encoding") == "gzip" {
			reader, _, err := httpserver.NewGZipBodyReader(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)

				return
			}

			c.Request.Body = reader
			c.Request.Header.Del("Content-Encoding")
		}

		c.Next()
	})
	router.Use(httpserver.MaxBodySizeMiddleware(limit))
	router.POST("/", readBodyHandler)

	req, err := http.NewRequest(http.MethodPost, "/", &compressed)
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code,
		"decompressed body exceeding limit must be rejected even when compressed wire size is small")
}
