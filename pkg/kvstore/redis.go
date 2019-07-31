package kvstore

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/msgpack"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
	"strings"
)

type RedisKvStore struct {
	client     redis.Client
	keyBuilder func(key interface{}) string
	settings   *Settings
}

func NewRedisKvStore(config cfg.Config, logger mon.Logger, settings *Settings) KvStore {
	client := redis.GetClient(config, logger, "kvstore")
	keyBuilder := redisKeyBuilder(config, settings)

	return NewRedisKvStoreWithInterfaces(client, keyBuilder, settings)
}

func NewRedisKvStoreWithInterfaces(client redis.Client, keyBuilder func(key interface{}) string, settings *Settings) *RedisKvStore {
	return &RedisKvStore{
		client:     client,
		settings:   settings,
		keyBuilder: keyBuilder,
	}
}

func (s *RedisKvStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	keyStr := s.keyBuilder(key)
	count, err := s.client.Exists(keyStr)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *RedisKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	bytes, err := msgpack.Marshal(value)

	if err != nil {
		return err
	}

	keyStr := s.keyBuilder(key)
	err = s.client.Set(keyStr, bytes, s.settings.Ttl)

	return err
}

func (s *RedisKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr := s.keyBuilder(key)
	data, err := s.client.Get(keyStr)

	if err == redis.Nil {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	err = msgpack.Unmarshal([]byte(data), value)

	return true, err
}

func redisKeyBuilder(config cfg.Config, settings *Settings) func(key interface{}) string {
	appId := cfg.GetAppIdFromConfig(config)

	return func(key interface{}) string {
		keyStr := KeyToString(key)

		return strings.Join([]string{
			appId.Project,
			appId.Family,
			appId.Application,
			"kvstore",
			settings.Name,
			keyStr,
		}, "-")
	}
}
