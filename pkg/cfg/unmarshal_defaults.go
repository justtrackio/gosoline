package cfg

import "github.com/justtrackio/gosoline/pkg/mapx"

type UnmarshalDefaults func(config Config, finalSettings *mapx.MapX) error

func UnmarshalWithDefaultsFromKey(sourceKey string, targetKey string) UnmarshalDefaults {
	return func(config Config, finalSettings *mapx.MapX) error {
		if !config.IsSet(sourceKey) {
			return nil
		}

		sourceValues := config.Get(sourceKey)
		finalSettings.Merge(targetKey, sourceValues)

		return nil
	}
}

func UnmarshalWithDefaultForKey(targetKey string, setting any) UnmarshalDefaults {
	return func(config Config, finalSettings *mapx.MapX) error {
		finalSettings.Set(targetKey, setting)

		return nil
	}
}
