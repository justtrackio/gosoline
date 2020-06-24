package cfg

import (
	"fmt"
	"github.com/imdario/mergo"
)

type UnmarshalDefaults func(config Config, finalSettings *map[string]interface{}) error

func UnmarshalWithDefaultsFromKey(sourceKey string, targetKey string) UnmarshalDefaults {
	return func(config Config, finalSettings *map[string]interface{}) error {
		if !config.IsSet(sourceKey) {
			return nil
		}

		sourceValues := config.Get(sourceKey)

		if msi, ok := sourceValues.(map[string]interface{}); ok {
			sourceMsi := NewMap()
			sourceMsi.Set(targetKey, msi)

			if err := mergo.Merge(finalSettings, sourceMsi.Msi(), mergo.WithOverride); err != nil {
				return err
			}

			return nil
		}

		return fmt.Errorf("source values should be a msi")
	}
}
