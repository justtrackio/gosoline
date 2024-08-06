package log

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

//go:generate mockery --name SentryHub
type SentryHub interface {
	ConfigureScope(f func(scope *sentry.Scope))
	WithScope(f func(scope *sentry.Scope))
	CaptureException(exception error) *sentry.EventID
	Flush(timeout time.Duration) bool
}

func NewSentryHub(config cfg.Config) (SentryHub, error) {
	settings := &SentryHubSettings{
		Environment: config.GetString("env"),
		AppName:     config.GetString("app_name"),
	}

	return NewSentryHubWithSettings(settings)
}

func NewSentryHubWithSettings(settings *SentryHubSettings) (SentryHub, error) {
	options := sentry.ClientOptions{
		Dsn:         settings.Dsn,
		Environment: settings.Environment,
	}

	var err error
	var client *sentry.Client
	scope := sentry.NewScope()

	if client, err = sentry.NewClient(options); err != nil {
		return nil, fmt.Errorf("can not create sentry client: %w", err)
	}

	hub := sentry.NewHub(client, scope)
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTags(map[string]string{
			"application": settings.AppName,
		})
	})

	return hub, nil
}
