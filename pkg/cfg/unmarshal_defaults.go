package cfg

import "github.com/applike/gosoline/pkg/mapx"

type UnmarshalDefaults func(config Config, finalSettings *mapx.MapX)

func UnmarshalWithDefaultsFromKey(sourceKey string, targetKey string) UnmarshalDefaults {
	return func(config Config, finalSettings *mapx.MapX) {
		if !config.IsSet(sourceKey) {
			return
		}

		sourceValues := config.Get(sourceKey)
		finalSettings.Merge(targetKey, sourceValues)
	}
}

func UnmarshalWithDefaultForKey(targetKey string, setting interface{}) UnmarshalDefaults {
	return func(config Config, finalSettings *mapx.MapX) {
		finalSettings.Set(targetKey, setting)
	}
}
