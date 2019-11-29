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

type RedisKvStore struct {
	logger   mon.Logger
	client   redis.Client
	settings *Settings
}

func NewRedisKvStore(config cfg.Config, logger mon.Logger, settings *Settings) KvStore {
	settings.PadFromConfig(config)

	redisName := fmt.Sprintf("kvstore_%s", settings.Name)
	client := redis.GetClient(config, logger, redisName)

	return NewRedisKvStoreWithInterfaces(logger, client, settings)
}

func NewRedisKvStoreWithInterfaces(logger mon.Logger, client redis.Client, settings *Settings) *RedisKvStore {
	return &RedisKvStore{
		logger:   logger,
		client:   client,
		settings: settings,
	}
}

func (s *RedisKvStore) Contains(_ context.Context, key interface{}) (bool, error) {
	keyStr, err := s.key(key)

	if err != nil {
		return false, err
	}

	count, err := s.client.Exists(keyStr)

	if err != nil {
		s.logger.Error(err, "can not check existence in redis store")
		return false, err
	}

	return count > 0, nil
}

func (s *RedisKvStore) Get(_ context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr, err := s.key(key)

	if err != nil {
		return false, err
	}

	data, err := s.client.Get(keyStr)

	if err == redis.Nil {
		return false, nil
	}

	if err != nil {
		s.logger.Error(err, "can not get value from redis store")
		return false, err
	}

	err = Unmarshal([]byte(data), value)

	if err != nil {
		s.logger.Error(err, "can not unmarshal value")
		return false, err
	}

	return true, nil
}

func (s *RedisKvStore) GetBatch(ctx context.Context, keys interface{}, result interface{}) ([]interface{}, error) {
	return getBatch(ctx, keys, result, s.getChunk, s.settings.BatchSize)
}

func (s *RedisKvStore) getChunk(_ context.Context, resultMap *refl.Map, keys []interface{}) ([]interface{}, error) {
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

	if len(items) != len(keys) {
		return nil, fmt.Errorf("count of returned items does not match key count %d != %d", len(items), len(keys))
	}

	for i, item := range items {
		if _, ok := item.(string); !ok {
			missing = append(missing, keys[i])
			continue
		}

		element := resultMap.NewElement()
		err = Unmarshal([]byte(item.(string)), element)

		if err != nil {
			return nil, fmt.Errorf("can not unmarshal item: %w", err)
		}

		if err := resultMap.Set(keys[i], element); err != nil {
			return nil, fmt.Errorf("can not set new element on result map: %w", err)
		}
	}

	return missing, nil
}

func (s *RedisKvStore) Put(_ context.Context, key interface{}, value interface{}) error {
	bytes, err := Marshal(value)

	if err != nil {
		s.logger.Error(err, "can not marshal value")
		return err
	}

	keyStr, err := s.key(key)

	if err != nil {
		return err
	}

	err = s.client.Set(keyStr, bytes, s.settings.Ttl)

	if err != nil {
		s.logger.Error(err, "can not set value in redis store")
		return err
	}

	return nil
}

func (s *RedisKvStore) PutBatch(ctx context.Context, values interface{}) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return fmt.Errorf("could not convert values to map[interface{}]interface{}")
	}

	for k, v := range mii {
		if err = s.Put(ctx, k, v); err != nil {
			return fmt.Errorf("can not put value into redis: %w", err)
		}
	}

	return nil
}

func (s *RedisKvStore) key(key interface{}) (string, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		s.logger.Error(err, "can not cast key to string")
		return "", err
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
