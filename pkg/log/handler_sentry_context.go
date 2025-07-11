package log

import (
	"fmt"
)

type ConfigProvider interface {
	AllSettings() map[string]any
}

type SentryContextProvider func(config ConfigProvider, sentryHook *HandlerSentry) error

func SentryContextConfigProvider(config ConfigProvider, handler *HandlerSentry) error {
	configValues := config.AllSettings()
	handler.WithContext("config", configValues)

	return nil
}

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
