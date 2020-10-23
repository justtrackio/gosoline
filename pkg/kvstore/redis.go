package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
	"github.com/applike/gosoline/pkg/refl"
	"strings"
)

type redisKvStore struct {
	client   redis.Client
	settings *Settings
}

func RedisBasename(settings *Settings) string {
	return fmt.Sprintf("kvstore_%s", settings.Name)
}

func NewRedisKvStore(config cfg.Config, logger mon.Logger, settings *Settings) KvStore {
	settings.PadFromConfig(config)

	redisName := RedisBasename(settings)
	client := redis.ProvideClient(config, logger, redisName)

	return NewRedisKvStoreWithInterfaces(client, settings)
}

func NewRedisKvStoreWithInterfaces(client redis.Client, settings *Settings) KvStore {
	return NewMetricStoreWithInterfaces(&redisKvStore{
		client:   client,
		settings: settings,
	}, settings)
}

func (s *redisKvStore) Contains(_ context.Context, key interface{}) (bool, error) {
	keyStr, err := s.key(key)

	if err != nil {
		return false, fmt.Errorf("can not get key to check value in redis: %w", err)
	}

	count, err := s.client.Exists(keyStr)

	if err != nil {
		return false, fmt.Errorf("can not check existence in redis store: %w", err)
	}

	return count > 0, nil
}

func (s *redisKvStore) Get(_ context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr, err := s.key(key)

	if err != nil {
		return false, fmt.Errorf("can not get key to read value from redis: %w", err)
	}

	data, err := s.client.Get(keyStr)

	if err == redis.Nil {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("can not get value from redis store: %w", err)
	}

	err = Unmarshal([]byte(data), value)

	if err != nil {
		return false, fmt.Errorf("can not unmarshal value from redis store: %w", err)
	}

	return true, nil
}

func (s *redisKvStore) GetBatch(ctx context.Context, keys interface{}, result interface{}) ([]interface{}, error) {
	return getBatch(ctx, keys, result, s.getChunk, s.settings.BatchSize)
}

func (s *redisKvStore) getChunk(_ context.Context, resultMap *refl.Map, keys []interface{}) ([]interface{}, error) {
	var err error

	missing := make([]interface{}, 0)
	keyStrings := make([]string, len(keys))

	for i := 0; i < len(keyStrings); i++ {
		keyStrings[i], err = s.key(keys[i])

		if err != nil {
			return nil, fmt.Errorf("can not build string key: %w", err)
		}
	}

	items, err := s.client.MGet(keyStrings...)

	if err != nil {
		return nil, fmt.Errorf("can not get batch from redis: %w", err)
	}

	// redis returns nil if a key is missing, otherwise we don't know which value is missing
	if len(items) != len(keys) {
		return nil, fmt.Errorf("count of returned items does not match key count %d != %d", len(items), len(keys))
	}

	for i, item := range items {
		item, ok := item.(string)

		if !ok {
			missing = append(missing, keys[i])

			continue
		}

		element := resultMap.NewElement()
		err = Unmarshal([]byte(item), element)

		if err != nil {
			return nil, fmt.Errorf("can not unmarshal item: %w", err)
		}

		if err := resultMap.Set(keys[i], element); err != nil {
			return nil, fmt.Errorf("can not set new element on result map: %w", err)
		}
	}

	return missing, nil
}

func (s *redisKvStore) Put(_ context.Context, key interface{}, value interface{}) error {
	bytes, err := Marshal(value)

	if err != nil {
		return fmt.Errorf("can not marshal value %T %v: %w", value, value, err)
	}

	keyStr, err := s.key(key)

	if err != nil {
		return fmt.Errorf("can not get key to write value to redis: %w", err)
	}

	err = s.client.Set(keyStr, bytes, s.settings.Ttl)

	if err != nil {
		return fmt.Errorf("can not set value in redis store: %w", err)
	}

	return nil
}

func (s *redisKvStore) PutBatch(ctx context.Context, values interface{}) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return fmt.Errorf("could not convert values from %T to map[interface{}]interface{}", values)
	}

	for k, v := range mii {
		if err = s.Put(ctx, k, v); err != nil {
			return fmt.Errorf("failed to write batch to redis: %w", err)
		}
	}

	return nil
}

func (s *redisKvStore) EstimateSize() *int64 {
	size, err := s.client.DBSize()

	if err != nil {
		return nil
	}

	return &size
}

func (s *redisKvStore) Delete(_ context.Context, key interface{}) error {
	keyStr, err := s.key(key)

	if err != nil {
		return fmt.Errorf("can not get key to delete value from redis: %w", err)
	}

	_, err = s.client.Del(keyStr)

	if err != nil {
		return fmt.Errorf("can not delete value from redis store: %w", err)
	}

	return nil
}

func (s *redisKvStore) DeleteBatch(_ context.Context, keys interface{}) error {
	si, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return fmt.Errorf("could not convert keys from %T to []interface{}: %w", keys, err)
	}

	redisKeys := make([]string, len(si))

	for i, key := range si {
		keyStr, err := s.key(key)

		if err != nil {
			return fmt.Errorf("can not get key to delete value from redis: %w", err)
		}

		redisKeys[i] = keyStr
	}

	_, err = s.client.Del(redisKeys...)

	if err != nil {
		return fmt.Errorf("can not delete values from redis store: %w", err)
	}

	return nil
}

func (s *redisKvStore) key(key interface{}) (string, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return "", fmt.Errorf("can not cast key %T %v to string: %w", key, key, err)
	}

	keyStr = strings.Join([]string{
		s.settings.Project,
		s.settings.Family,
		s.settings.Application,
		"kvstore",
		s.settings.Name,
		keyStr,
	}, "-")

	return keyStr, nil
}
