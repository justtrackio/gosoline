package apiserver

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

func LoggingMiddleware(logger mon.Logger) gin.HandlerFunc {
	chLogger := logger.WithChannel("http")

	return func(ginCtx *gin.Context) {
		start := time.Now()

		ginCtx.Next()

		req := ginCtx.Request
		ctx := req.Context()
		log := chLogger.WithContext(ctx)

		path := req.URL.Path
		query := req.URL.RawQuery
		pathRaw := getPathRaw(ginCtx)

		method := ginCtx.Request.Method
		requestTimeNano := time.Since(start)
		requestTimeSecond := float64(requestTimeNano) / float64(time.Second)

		log.WithFields(mon.Fields{
			"bytes":            ginCtx.Writer.Size(),
			"client_ip":        ginCtx.ClientIP(),
			"host":             req.Host,
			"protocol":         req.Proto,
			"request_method":   method,
			"request_path":     path,
			"request_query":    query,
			"request_path_raw": pathRaw,
			"request_time":     requestTimeSecond,
			"scheme":           req.URL.Scheme,
			"status":           ginCtx.Writer.Status(),
		}).Infof("%s %s %s", method, path, req.Proto)
	}
}
func getPathRaw(ginCtx *gin.Context) string {
	path := ginCtx.Request.URL.Path

	for i := range ginCtx.Params {
		p := ginCtx.Params[i]
		k := fmt.Sprintf(":%s", p.Key)
		path = strings.Replace(path, p.Value, k, 1)
	}

	return path
}
