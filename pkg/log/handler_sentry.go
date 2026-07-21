package log

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	AddHandlerFactory("sentry", handlerSentryFactory)
}

func handlerSentryFactory(config cfg.Config, _ string) (Handler, error) {
	return NewHandlerSentry(config)
}

// HandlerSentry forwards log messages with error levels to Sentry.
type HandlerSentry struct {
	hub SentryHub
}

// NewHandlerSentry creates a new Sentry handler initialized with settings from the provided configuration.
func NewHandlerSentry(config cfg.Config) (*HandlerSentry, error) {
	var err error
	var hub SentryHub

	if hub, err = NewSentryHub(config); err != nil {
		return nil, fmt.Errorf("can not create sentry hub: %w", err)
	}

	return &HandlerSentry{
		hub: hub,
	}, nil
}

// WithContext adds contextual information to the Sentry scope for future errors.
func (h *HandlerSentry) WithContext(name string, context map[string]any) {
	h.hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext(name, context)
	})
}

// ChannelLevel returns nil for the Sentry handler, as it doesn't support channel-specific levels.
// It relies on the global level configuration.
func (h *HandlerSentry) ChannelLevel(string) (level *int, err error) {
	return nil, nil
}

// Level returns the default log level priority for the Sentry handler, which is PriorityError.
// This means, by default, only error logs are sent to Sentry.
func (h *HandlerSentry) Level() int {
	return PriorityError
}

// Log sends the error from the log entry to Sentry.
// It includes fields and context as Sentry context data.
func (h *HandlerSentry) Log(_ context.Context, _ time.Time, _ int, _ string, _ []any, err error, data Data) error {
	if err == nil {
		return nil
	}

	fields := mergeFields(data.Fields, data.ContextFields)

	h.hub.WithScope(func(scope *sentry.Scope) {
		scope.SetContext("fields", fields)

		eventId := h.hub.CaptureException(err)

		if eventId != nil {
			data.Fields = mergeFields(data.Fields, map[string]any{
				"sentry_event_id": *eventId,
			})
		}
	})

	return nil
}
