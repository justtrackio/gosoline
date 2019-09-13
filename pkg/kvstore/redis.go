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

func (s *RedisKvStore) Contains(ctx context.Context, key interface{}) (bool, error) {
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

func (s *RedisKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
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
