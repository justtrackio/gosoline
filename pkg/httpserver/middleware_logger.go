package httpserver

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reqctx"
)

type logCall struct {
	logger   log.Logger
	settings LoggingSettings
	fields   log.Fields
}

func LoggingMiddleware(logger log.Logger, settings LoggingSettings) gin.HandlerFunc {
	logger = logger.WithChannel("http")

	return NewLoggingMiddlewareWithInterfaces(logger, settings, clock.Provider)
}

func NewLoggingMiddlewareWithInterfaces(logger log.Logger, settings LoggingSettings, clock clock.Clock) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		start := clock.Now()

		ctx := log.InitContext(ginCtx.Request.Context())
		ctx = reqctx.New(ctx)

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

		lp := newLogCall(logger, settings)
		lp.prepare(ginCtx)

		ginCtx.Next()

		requestTimeSeconds := clock.Since(start).Seconds()

		lp.finalize(ginCtx, requestTimeSeconds)
	}
}

func newLogCall(logger log.Logger, settings LoggingSettings) *logCall {
	return &logCall{
		logger:   logger,
		settings: settings,
		fields:   log.Fields{},
	}
}

func (lc *logCall) prepare(ginCtx *gin.Context) {
	req := ginCtx.Request

	lc.fields["bytes"] = ginCtx.Writer.Size()
	lc.fields["client_ip"] = ginCtx.ClientIP()
	lc.fields["host"] = req.Host
	lc.fields["protocol"] = req.Proto
	lc.fields["request_method"] = req.Method
	lc.fields["request_path"] = req.URL.Path
	lc.fields["request_path_raw"] = getPathRaw(ginCtx)
	lc.fields["request_query"] = req.URL.RawQuery
	lc.fields["request_referer"] = req.Referer()
	lc.fields["request_user_agent"] = req.UserAgent()
	lc.fields["scheme"] = req.URL.Scheme

	if !lc.settings.RequestBody {
		return
	}

	buf, err := io.ReadAll(req.Body)
	if err != nil {
		lc.logger.Warn(req.Context(), "can not read request body: %s", err.Error())

		return
	}

	// restore the body so another handler can read it
	req.Body = io.NopCloser(bytes.NewBuffer(buf))

	if lc.settings.RequestBodyBase64 {
		lc.fields["request_body"] = string(base64.Encode(buf))
	} else {
		lc.fields["request_body"] = string(buf)
	}
}

func (lc *logCall) finalize(ginCtx *gin.Context, requestTimeSecond float64) {
	status := ginCtx.Writer.Status()

	// these fields can only be added after all previous handlers have finished
	lc.fields = funk.MergeMaps(lc.fields, getRequestSizeFields(ginCtx))
	lc.fields["bytes"] = ginCtx.Writer.Size()
	lc.fields["request_time"] = requestTimeSecond
	lc.fields["status"] = status

	// only log query parameters in full for successful requests to avoid logging them from bad crawlers
	if status != http.StatusUnauthorized && status != http.StatusForbidden && status != http.StatusNotFound {
		queryParameters := make(map[string]string)
		query := ginCtx.Request.URL.Query()

		for k := range query {
			queryParameters[k] = query.Get(k)
		}

		lc.fields["request_query_parameters"] = queryParameters
	}

	ctx := ginCtx.Request.Context()
	logger := lc.logger.WithFields(lc.fields)
	method, path, proto := lc.fields["request_method"], lc.fields["request_path"], lc.fields["protocol"]

	if len(ginCtx.Errors) == 0 {
		logger.Info(ctx, "%s %s %s", method, path, proto)

		return
	}

	for _, e := range ginCtx.Errors {
		switch {
		case exec.IsRequestCanceled(e):
			logger.Info(ctx, "%s %s %s - request canceled: %s", method, path, proto, e.Error())
		case exec.IsConnectionError(e):
			logger.Info(ctx, "%s %s %s - connection error: %s", method, path, proto, e.Error())
		case e.IsType(gin.ErrorTypeBind):
			logger.Warn(ctx, "%s %s %s - bind error: %s", method, path, proto, e.Err.Error())
		case e.IsType(gin.ErrorTypeRender):
			logger.Warn(ctx, "%s %s %s - render error: %s", method, path, proto, e.Err.Error())
		default:
			logger.Error(ctx, "%s %s %s: %w", method, path, proto, e.Err)
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
