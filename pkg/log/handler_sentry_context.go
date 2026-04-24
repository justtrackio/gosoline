package log

import (
	"fmt"
	"strings"
)

// ConfigProvider defines an interface for retrieving all configuration settings as a map.
type ConfigProvider interface {
	AllSettings() map[string]any
}

// SentryContextProvider is a function type for attaching extra context (like config or ECS metadata) to the Sentry handler.
type SentryContextProvider func(config ConfigProvider, sentryHook *HandlerSentry) error

var sentrySensitivePatterns = []string{
	"password", "secret", "token", "key", "dsn", "credential",
}

func isSentrySensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, pattern := range sentrySensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

func maskSensitiveConfigValues(m map[string]any) map[string]any {
	masked := make(map[string]any, len(m))
	for k, v := range m {
		if isSentrySensitiveKey(k) {
			masked[k] = "***"

			continue
		}
		if nested, ok := v.(map[string]any); ok {
			masked[k] = maskSensitiveConfigValues(nested)

			continue
		}
		masked[k] = v
	}

	return masked
}

// SentryContextConfigProvider attaches the configuration (with sensitive values masked) as context to Sentry events.
func SentryContextConfigProvider(config ConfigProvider, handler *HandlerSentry) error {
	configValues := config.AllSettings()
	handler.WithContext("config", maskSensitiveConfigValues(configValues))
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
