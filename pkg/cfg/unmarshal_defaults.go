package cfg

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/mapx"
)

type UnmarshalDefaults func(config Config, finalSettings *mapx.MapX) error

func UnmarshalWithDefaultsFromKey(sourceKey string, targetKey string) UnmarshalDefaults {
	return func(config Config, finalSettings *mapx.MapX) error {
		if !config.IsSet(sourceKey) {
			return nil
		}

		sourceValues, err := config.Get(sourceKey)
		if err != nil {
			return fmt.Errorf("could not load source value for key %s: %w", sourceKey, err)
		}

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
