package log

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

//go:generate go run github.com/vektra/mockery/v2 --name SentryHub
type SentryHub interface {
	ConfigureScope(f func(scope *sentry.Scope))
	WithScope(f func(scope *sentry.Scope))
	CaptureException(exception error) *sentry.EventID
	Flush(timeout time.Duration) bool
}

type SentryHubSettings struct {
	Dsn         string
	Environment string
	AppFamily   string
	AppName     string
	AppGroup    string
}

func NewSentryHub(config cfg.Config) (SentryHub, error) {
	var appId cfg.AppId
	if err := appId.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("failed to pad from config: %w", err)
	}

	settings := &SentryHubSettings{
		Environment: appId.Environment,
		AppFamily:   appId.Family,
		AppName:     appId.Application,
		AppGroup:    appId.Group,
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
			"family":      settings.AppFamily,
			"application": settings.AppName,
			"group":       settings.AppGroup,
		})
	})

	return hub, nil
}
