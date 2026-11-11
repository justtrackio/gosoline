package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reqctx"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type logCall struct {
	logger   log.Logger
	settings LoggingSettings
	fields   log.Fields
	method   string
	path     string
	proto    string
}

func LoggingMiddleware(logger log.Logger, settings LoggingSettings) gin.HandlerFunc {
	logger = logger.WithChannel("http")

	return NewLoggingMiddlewareWithInterfaces(logger, settings, clock.Provider)
}

func NewLoggingMiddlewareWithInterfaces(logger log.Logger, settings LoggingSettings, clock clock.Clock) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		start := clock.Now()

		reqCtx := log.WithFingersCrossedScope(ginCtx.Request.Context())
		reqCtx = log.InitContext(reqCtx)
		reqCtx = reqctx.New(reqCtx)

		if requestId := ginCtx.Request.Header.Get("X-Request-Id"); requestId != "" {
			reqCtx = log.MutateGlobalContextFields(reqCtx, map[string]any{
				"request_id": requestId,
			})
		}

		if ginCtx.Request.Header.Get("X-Session-Id") != "" {
			reqCtx = log.MutateGlobalContextFields(reqCtx, map[string]any{
				"session_id": log.Redacted,
			})
		}

		ginCtx.Request = ginCtx.Request.WithContext(reqCtx)

		lp := newLogCall(logger, settings)
		lp.prepare(ginCtx)

		ginCtx.Next()

		requestTimeSeconds := clock.Since(start).Seconds()

		lp.finalize(ginCtx, requestTimeSeconds)

		// If the request failed, ensure we flush buffered (not-sampled) logs.
		if ginCtx.Writer.Status() >= http.StatusBadRequest {
			log.FlushFingersCrossedScope(reqCtx)
		}
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
	lc.method, lc.path, lc.proto = req.Method, ginCtx.FullPath(), req.Proto
	if lc.path == "" {
		lc.path = "[unmatched]"
	}
	lc.fields["protocol"] = lc.proto
	lc.fields["request_method"] = lc.method
	lc.fields["request_path"] = lc.path
	if lc.settings.RequestReferer {
		lc.fields["request_referer"] = log.SanitizeURL(req.Referer())
	}
	lc.fields["request_user_agent"] = req.UserAgent()
	lc.fields["scheme"] = req.URL.Scheme

	headers := make(map[string]string)
	for _, key := range lc.settings.RequestHeaders {
		headers[key] = log.RequestHeaderForLog(req.Header, key)
	}

	if len(headers) > 0 {
		lc.fields["request_headers"] = headers
	}

	if !lc.settings.RequestBody || len(lc.settings.RequestBodyFields) == 0 {
		return
	}

	buf, err := io.ReadAll(req.Body)
	if err != nil {
		lc.logger.Warn(req.Context(), "can not read request body: %s", log.SafeError(err).Error())

		return
	}

	// restore the body so another handler can read it
	req.Body = io.NopCloser(bytes.NewBuffer(buf))

	var body map[string]any
	if err := json.Unmarshal(buf, &body); err != nil {
		// Never fall back to raw payloads on invalid JSON or another content type.
		return
	}
	selected := make(map[string]any)
	for _, key := range lc.settings.RequestBodyFields {
		if value, ok := body[key]; ok {
			selected[key] = value
		}
	}
	safe := log.SanitizeFields(selected)
	if lc.settings.RequestBodyBase64 {
		encoded, err := json.Marshal(safe)
		if err == nil {
			lc.fields["request_body"] = string(base64.Encode(encoded))
		}
	} else {
		lc.fields["request_body"] = safe
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

		for _, key := range lc.settings.RequestQueryParameters {
			if !query.Has(key) {
				continue
			}
			if log.IsSensitiveField(key) {
				queryParameters[key] = log.Redacted
			} else {
				queryParameters[key] = query.Get(key)
			}
		}

		lc.fields["request_query_parameters"] = queryParameters
	}

	ctx := ginCtx.Request.Context()
	logger := lc.logger.WithFields(lc.fields)
	method, path, proto := lc.method, lc.path, lc.proto

	if len(ginCtx.Errors) == 0 {
		logger.Info(ctx, "%s %s %s", method, path, proto)

		return
	}

	for _, e := range ginCtx.Errors {
		switch {
		case exec.IsRequestCanceled(e):
			logger.Info(ctx, "%s %s %s - request canceled: %s", method, path, proto, log.SafeError(e.Err).Error())
		case exec.IsConnectionError(e):
			logger.Info(ctx, "%s %s %s - connection error: %s", method, path, proto, log.SafeError(e.Err).Error())
		case e.IsType(gin.ErrorTypeBind):
			logger.Warn(ctx, "%s %s %s - bind error: %s", method, path, proto, formatBindError(e.Err))
		case e.IsType(gin.ErrorTypeRender):
			logger.Warn(ctx, "%s %s %s - render error: %s", method, path, proto, log.SafeError(e.Err).Error())
		case validation.IsValidationError(e):
			logger.Warn(ctx, "%s %s %s - validation error: %s", method, path, proto, log.SafeError(e.Err).Error())
		default:
			logger.Error(ctx, "%s %s %s: %w", method, path, proto, log.SafeError(e.Err))
		}
	}
}

func formatBindError(err error) string {
	var failures validator.ValidationErrors
	if !errors.As(err, &failures) {
		// Parser messages may contain rejected input. Keep their type, not the payload.
		return fmt.Sprintf("%T", err)
	}
	details := make([]string, 0, len(failures))
	for _, failure := range failures {
		details = append(details, fmt.Sprintf("field=%s tag=%s", failure.Field(), failure.Tag()))
	}

	return strings.Join(details, "; ")
}
