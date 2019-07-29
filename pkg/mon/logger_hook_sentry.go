package mon

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

type sentryHook struct {
	sentry Sentry
}

func NewSentryHook(config cfg.Config) *sentryHook {
	env := config.GetString("env")

	sentry := raven.DefaultClient
	sentry.SetEnvironment(env)

	return &sentryHook{
		sentry: sentry,
	}
}

func (h sentryHook) Fire(level string, msg string, err error, fields Fields, contextFields ContextFields, tags Tags, configValues ConfigValues, context context.Context, ecsMetadata EcsMetadata) error {
	if err == nil {
		return nil
	}

	cause := errors.Cause(err)
	trace := raven.GetOrNewStacktrace(err, 5, 3, []string{})
	exception := raven.NewException(cause, trace)

	extra := raven.Extra{
		//"config": configValues,
		"fields":       fields,
		"context":      contextFields,
		"ecs_metadata": ecsMetadata,
	}

	packet := raven.NewPacketWithExtra(err.Error(), extra, exception)

	_, res := h.sentry.Capture(packet, tags)
	err = <-res

	fields["sentry_event_id"] = packet.EventID

	return err
}
