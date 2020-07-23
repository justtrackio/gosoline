package mon

import (
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

//go:generate mockery -name Sentry
type Sentry interface {
	Capture(packet *raven.Packet, captureTags map[string]string) (eventID string, ch chan error)
}

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

func (h SentryHook) Fire(_ string, _ string, err error, data *Metadata) error {
	if err == nil {
		return nil
	}

	stringTags := make(map[string]string)
	for k, v := range data.Tags {
		stringTags[k] = fmt.Sprint(v)
	}

	cause := errors.Cause(err)
	trace := raven.GetOrNewStacktrace(err, 4, 3, []string{})
	exception := raven.NewException(cause, trace)

	extra := mergeMapStringInterface(h.extra, data.Fields)
	extra = mergeMapStringInterface(extra, data.ContextFields)
	packet := raven.NewPacketWithExtra(err.Error(), extra, exception)

	_, res := h.sentry.Capture(packet, stringTags)
	err = <-res

	data.Fields["sentry_event_id"] = packet.EventID

	return err
}
