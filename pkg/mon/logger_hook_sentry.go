package mon

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

//go:generate mockery -name Sentry
type Sentry interface {
	CaptureException(exception error, hint *sentry.EventHint, scope sentry.EventModifier) *sentry.EventID
}

type SentryHook struct {
	sentry Sentry
	extra  map[string]interface{}
}

func NewSentryHook(env string) *SentryHook {
	client, _ := sentry.NewClient(sentry.ClientOptions{
		Environment: env,
	})

	return &SentryHook{
		sentry: client,
		extra:  make(map[string]interface{}),
	}
}

func (h SentryHook) WithExtra(extra map[string]interface{}) *SentryHook {
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

	extra := mergeMapStringInterface(h.extra, data.Fields)
	extra = mergeMapStringInterface(extra, data.ContextFields)

	scope := sentry.NewScope()
	scope.SetTags(stringTags)
	scope.SetExtras(extra)

	data.Fields["sentry_event_id"] = h.sentry.CaptureException(cause, nil, scope)

	return err
}
