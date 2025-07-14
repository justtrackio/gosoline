package kvstore

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/redis"
)

const (
	ConfigKeyKvstore = "kvstore"
)

func init() {
	// Should run after mdlsub postprocessor, in case we ever decide to define magic to determine the redis db index
	cfg.AddPostProcessor(7, "gosoline.kvstore.redis", RedisConfigPostProcessor)
}

func RedisConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	if !config.IsSet(ConfigKeyKvstore) {
		return false, nil
	}

	kvstores, err := config.GetStringMap(ConfigKeyKvstore)
	if err != nil {
		return false, fmt.Errorf("failed to get kvstore settings: %w", err)
	}

	for name, kvstore := range kvstores {
		var elements []any
		var kvstoreMap map[string]any
		var ok bool

		if kvstoreMap, ok = kvstore.(map[string]any); !ok {
			continue
		}

		if elements, ok = kvstoreMap["elements"].([]any); !ok {
			continue
		}

		if !funk.Contains(elements, TypeRedis) {
			continue
		}

		kvstoreKey := GetConfigurableKey(name)

		configuration := ChainConfiguration{}
		if err := config.UnmarshalKey(kvstoreKey, &configuration); err != nil {
			return false, fmt.Errorf("failed to unmarshal kvstore redis configuration for %s: %w", name, err)
		}

		if configuration.Redis.DB == 0 {
			continue
		}

		// not reading the whole default settings here as it would implicitly set the hostname/port and other settings,
		// that we don't want to override here
		redisBaseName := RedisBasename(name)
		redisKey := fmt.Sprintf("%s.db", redis.GetRedisConfigKey(redisBaseName))

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(redisKey, configuration.Redis.DB),
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply redis config settings for kvstore %s: %w", kvstoreKey, err)
		}
	}

	return true, nil
}
