package log

import (
	"fmt"
)

// ConfigProvider defines an interface for retrieving all configuration settings as a map.
type ConfigProvider interface {
	AllSettings() map[string]any
}

// SentryContextProvider is a function type for attaching extra context (like config or ECS metadata) to the Sentry handler.
type SentryContextProvider func(config ConfigProvider, sentryHook *HandlerSentry) error

// SentryContextConfigProvider attaches the entire configuration as context to Sentry events.
func SentryContextConfigProvider(config ConfigProvider, handler *HandlerSentry) error {
	configValues := config.AllSettings()
	handler.WithContext("config", configValues)

	return nil
}

// SentryContextEcsMetadataProvider attaches ECS metadata (if available) as context to Sentry events.
func SentryContextEcsMetadataProvider(_ ConfigProvider, handler *HandlerSentry) error {
	ecsMetadata, err := ReadEcsMetadata()
	if err != nil {
		return fmt.Errorf("can not read ecs metadata: %w", err)
	}

	if ecsMetadata == nil {
		return nil
	}

	handler.WithContext("ecsMetadata", ecsMetadata)

	return nil
}
