package mon

import "github.com/pkg/errors"

type ConfigProvider interface {
	AllSettings() map[string]interface{}
}

type SentryExtraProvider func(config ConfigProvider, sentryHook *SentryHook) (*SentryHook, error)

func SentryExtraConfigProvider(config ConfigProvider, sentryHook *SentryHook) (*SentryHook, error) {
	configValues := config.AllSettings()
	sentryHook = sentryHook.WithExtra(map[string]interface{}{
		"config": configValues,
	})

	return sentryHook, nil
}

func SentryExtraEcsMetadataProvider(_ ConfigProvider, sentryHook *SentryHook) (*SentryHook, error) {
	ecsMetadata, err := ReadEcsMetadata()

	if err != nil {
		return sentryHook, errors.Wrap(err, "can not read ecs metadata")
	}

	if ecsMetadata != nil {
		return sentryHook, nil
	}

	sentryHook = sentryHook.WithExtra(map[string]interface{}{
		"ecsMetadata": ecsMetadata,
	})

	return sentryHook, nil
}
