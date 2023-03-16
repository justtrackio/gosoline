package log

import (
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

type HandlerSentry struct {
	hub SentryHub
}

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

func (h *HandlerSentry) WithContext(name string, context map[string]interface{}) {
	h.hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext(name, context)
	})
}

func (h *HandlerSentry) Channels() []string {
	return []string{}
}

func (h *HandlerSentry) Level() int {
	return PriorityError
}

func (h *HandlerSentry) Log(_ time.Time, _ int, _ string, _ []interface{}, err error, data Data) error {
	if err == nil {
		return nil
	}

	fields := mergeFields(data.Fields, data.ContextFields)

	h.hub.WithScope(func(scope *sentry.Scope) {
		scope.SetContext("fields", fields)

		eventId := h.hub.CaptureException(err)

		if eventId != nil {
			data.Fields = mergeFields(data.Fields, map[string]interface{}{
				"sentry_event_id": *eventId,
			})
		}
	})

	return nil
}
