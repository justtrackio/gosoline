package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ConcurrentRequestLimitMiddleware limits the number of concurrently handled requests.
// When the limit is reached, it rejects new requests immediately instead of queueing them.
func ConcurrentRequestLimitMiddleware(settings ConcurrencySettings) gin.HandlerFunc {
	if settings.MaxRequests <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	semaphore := make(chan struct{}, settings.MaxRequests)

	return func(c *gin.Context) {
		select {
		case semaphore <- struct{}{}:
			defer func() {
				<-semaphore
			}()

			c.Next()
		default:
			c.Request = markRequestRejected(c.Request)
			writeRetryAfterHeader(c, settings.RetryAfter)
			c.AbortWithStatusJSON(settings.OverloadStatusCode, gin.H{
				"error": "server overloaded",
			})
		}
	}
}

func writeRetryAfterHeader(c *gin.Context, retryAfter time.Duration) {
	if retryAfter <= 0 {
		return
	}

	seconds := int64(retryAfter / time.Second)
	if retryAfter%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		seconds = 1
	}

	c.Header("Retry-After", strconv.FormatInt(seconds, 10))
}

type rejectedRequestKey struct{}

func markRequestRejected(request *http.Request) *http.Request {
	return request.WithContext(context.WithValue(request.Context(), rejectedRequestKey{}, true))
}

func wasRequestRejected(request *http.Request) bool {
	isRejected, ok := request.Context().Value(rejectedRequestKey{}).(bool)

	return ok && isRejected
}
