package log

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

// SentryHub abstracts the Sentry SDK's Hub, allowing for testing and decoupling.
//
//go:generate go run github.com/vektra/mockery/v2 --name SentryHub
type SentryHub interface {
	ConfigureScope(f func(scope *sentry.Scope))
	WithScope(f func(scope *sentry.Scope))
	CaptureException(exception error) *sentry.EventID
	Flush(timeout time.Duration) bool
}

// SentryHubSettings configuration for establishing a connection to Sentry.
type SentryHubSettings struct {
	Dsn          string
	Environment  string
	AppName      string
	AppNamespace string
}

// NewSentryHub creates a new SentryHub using configuration from the "app_id" settings.
func NewSentryHub(config cfg.Config) (SentryHub, error) {
	var err error
	var identity cfg.Identity
	var namespace string

	if identity, err = cfg.GetAppIdentity(config); err != nil {
		return nil, fmt.Errorf("failed to pad from config: %w", err)
	}

	if namespace, err = identity.FormatNamespace("."); err != nil {
		return nil, fmt.Errorf("failed to format namespace: %w", err)
	}

	settings := &SentryHubSettings{
		Environment:  identity.Env,
		AppName:      identity.Name,
		AppNamespace: namespace,
	}

	return NewSentryHubWithSettings(settings)
}

// NewSentryHubWithSettings creates a new SentryHub with the provided settings.
// It initializes the Sentry client and configures the scope with application tags.
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
			"namespace":   settings.AppNamespace,
		})
	})

	return hub, nil
}
