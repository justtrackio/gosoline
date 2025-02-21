package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"
)

type ContextKey string

const (
	RequestContextKey ContextKey = "http-request"
)

func ContextWithRequestMiddleware(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), RequestContextKey, c)
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

func GetRequestFromContext(ctx context.Context) *gin.Context {
	return ctx.Value(RequestContextKey).(*gin.Context)
}
