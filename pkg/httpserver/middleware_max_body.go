package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySizeMiddleware limits the size of incoming request bodies.
// When maxBytes is greater than zero, any request body larger than maxBytes
// is rejected with 413 Request Entity Too Large before handler code reads it.
func MaxBodySizeMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
