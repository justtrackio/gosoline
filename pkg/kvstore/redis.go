package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
	"strings"
)

type RedisKvStore struct {
	client   redis.Client
	settings *Settings
}

func NewRedisKvStore(config cfg.Config, logger mon.Logger, settings *Settings) KvStore {
	settings.PadFromConfig(config)

	redisName := fmt.Sprintf("kvstore_%s", settings.Name)
	client := redis.GetClient(config, logger, redisName)

	return NewRedisKvStoreWithInterfaces(client, settings)
}

func NewRedisKvStoreWithInterfaces(client redis.Client, settings *Settings) *RedisKvStore {
	return &RedisKvStore{
		client:   client,
		settings: settings,
	}
}

func (s *RedisKvStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	keyStr, err := s.key(key)

	if err != nil {
		return false, err
	}

	count, err := s.client.Exists(keyStr)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *RedisKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	bytes, err := Marshal(value)

	if err != nil {
		return err
	}

	keyStr, err := s.key(key)

	if err != nil {
		return err
	}

	err = s.client.Set(keyStr, bytes, s.settings.Ttl)

	return err
}

func (s *RedisKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr, err := s.key(key)

	if err != nil {
		return false, err
	}

	data, err := s.client.Get(keyStr)

	if err == redis.Nil {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	err = Unmarshal([]byte(data), value)

	return true, err
}

func (s *RedisKvStore) key(key interface{}) (string, error) {
	keyStr, err := KeyToString(key)

	if err != nil {
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
