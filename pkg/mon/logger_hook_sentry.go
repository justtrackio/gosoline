package mon

import (
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

type SentryHook struct {
	sentry Sentry
	extra  raven.Extra
}

func NewSentryHook(env string) *SentryHook {
	sentry := raven.DefaultClient
	sentry.SetEnvironment(env)

	return &SentryHook{
		sentry: sentry,
		extra:  make(raven.Extra),
	}
}

func (h SentryHook) WithExtra(extra raven.Extra) *SentryHook {
	newExtra := mergeMapStringInterface(h.extra, extra)

	return &SentryHook{
		sentry: h.sentry,
		extra:  newExtra,
	}
}

func (h SentryHook) Fire(level string, msg string, err error, data *Metadata) error {
	if err == nil {
		return nil
	}

	stringTags := make(map[string]string)
	for k, v := range data.tags {
		stringTags[k] = fmt.Sprint(v)
	}

	cause := errors.Cause(err)
	trace := raven.GetOrNewStacktrace(err, 4, 3, []string{})
	exception := raven.NewException(cause, trace)

	extra := mergeMapStringInterface(h.extra, data.fields)
	extra = mergeMapStringInterface(extra, data.contextFields)
	packet := raven.NewPacketWithExtra(err.Error(), extra, exception)

	_, res := h.sentry.Capture(packet, stringTags)
	err = <-res

	data.fields["sentry_event_id"] = packet.EventID

	return err
}
