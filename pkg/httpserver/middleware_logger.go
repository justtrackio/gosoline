package httpserver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/log"
)

func LoggingMiddleware(logger log.Logger, settings LoggingSettings) gin.HandlerFunc {
	chLogger := logger.WithChannel("http")

	return func(ginCtx *gin.Context) {
		start := time.Now()
		var requestBody []byte

		if settings.RequestBody {
			buf, err := io.ReadAll(ginCtx.Request.Body)
			if err != nil {
				chLogger.Warn("can not read request body: %s", err.Error())
			} else {
				requestBody = buf
				ginCtx.Request.Body = io.NopCloser(bytes.NewBuffer(buf))
			}
		}

		ctx := log.InitContext(ginCtx.Request.Context())

		if requestId := ginCtx.Request.Header.Get("X-Request-Id"); requestId != "" {
			ctx = log.MutateGlobalContextFields(ctx, map[string]any{
				"request_id": requestId,
			})
		}

		if sessionId := ginCtx.Request.Header.Get("X-Session-Id"); sessionId != "" {
			ctx = log.MutateGlobalContextFields(ctx, map[string]any{
				"session_id": sessionId,
			})
		}

		ginCtx.Request = ginCtx.Request.WithContext(ctx)

		ginCtx.Next()

		req := ginCtx.Request
		ctx = req.Context()

		path := req.URL.Path
		pathRaw := getPathRaw(ginCtx)

		referer := req.Referer()
		userAgent := req.UserAgent()
		status := ginCtx.Writer.Status()

		queryRaw := req.URL.RawQuery

		method := ginCtx.Request.Method
		requestTimeNano := time.Since(start)
		requestTimeSecond := float64(requestTimeNano) / float64(time.Second)

		fields := getRequestSizeFields(ginCtx)
		fields["bytes"] = ginCtx.Writer.Size()
		fields["client_ip"] = ginCtx.ClientIP()
		fields["host"] = req.Host
		fields["protocol"] = req.Proto
		fields["request_method"] = method
		fields["request_path"] = path
		fields["request_path_raw"] = pathRaw
		fields["request_query"] = queryRaw
		fields["request_referer"] = referer
		fields["request_user_agent"] = userAgent
		fields["request_time"] = requestTimeSecond
		fields["scheme"] = req.URL.Scheme
		fields["status"] = status

		if settings.RequestBody && requestBody != nil {
			if !settings.RequestBodyBase64 {
				fields["request_body"] = string(requestBody)
			} else {
				fields["request_body"] = string(base64.Encode(requestBody))
			}
		}

		// only log query parameters in full for successful requests to avoid logging them from bad crawlers
		if status != http.StatusUnauthorized && status != http.StatusForbidden && status != http.StatusNotFound {
			queryParameters := make(map[string]string)
			query := req.URL.Query()

			for k := range query {
				queryParameters[k] = query.Get(k)
			}

			fields["request_query_parameters"] = queryParameters
		}

		ctxLogger := chLogger.WithContext(ctx).WithFields(fields)

		if len(ginCtx.Errors) == 0 {
			ctxLogger.Info("%s %s %s", method, path, req.Proto)

			return
		}

		for _, e := range ginCtx.Errors {
			switch e.Type {
			case gin.ErrorTypeBind:
				if errors.Is(e.Err, io.EOF) || errors.Is(e.Err, io.ErrUnexpectedEOF) {
					ctxLogger.Warn("%s %s %s - network error - client has gone away - %v", method, path, req.Proto, e.Err)
				} else {
					ctxLogger.Warn("%s %s %s - bind error - %v", method, path, req.Proto, e.Err)
				}
			case gin.ErrorTypeRender:
				ctxLogger.Warn("%s %s %s - render error - %v", method, path, req.Proto, e.Err)
			default:
				ctxLogger.Error("%s %s %s: %w", method, path, req.Proto, e.Err)
			}
		}
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
