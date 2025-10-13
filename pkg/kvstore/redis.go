package kvstore

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/redis"
	"github.com/justtrackio/gosoline/pkg/refl"
)

type redisKvStore[T any] struct {
	client   redis.Client
	settings *Settings
}

func RedisBasename(name string) string {
	return fmt.Sprintf("kvstore-%s", name)
}

func NewRedisKvStore[T any](ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (KvStore[T], error) {
	if reflect.ValueOf(new(T)).Elem().Kind() == reflect.Pointer {
		return nil, fmt.Errorf("the generic type T should not be a pointer type but is of type %T", *new(T))
	}

	if err := settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("failed to pad settings from config: %w", err)
	}

	redisName := RedisBasename(settings.Name)

	client, err := redis.ProvideClient(ctx, config, logger, redisName)
	if err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	return NewRedisKvStoreWithInterfaces[T](client, settings), nil
}

func NewRedisKvStoreWithInterfaces[T any](client redis.Client, settings *Settings) KvStore[T] {
	return NewMetricStoreWithInterfaces[T](&redisKvStore[T]{
		client:   client,
		settings: settings,
	}, settings)
}

func (s *redisKvStore[T]) Contains(ctx context.Context, key any) (bool, error) {
	keyStr, err := s.key(key)
	if err != nil {
		return false, fmt.Errorf("can not get key to check value in redis: %w", err)
	}

	count, err := s.client.Exists(ctx, keyStr)
	if err != nil {
		return false, fmt.Errorf("can not check existence in redis store: %w", err)
	}

	return count > 0, nil
}

func (s *redisKvStore[T]) Get(ctx context.Context, key any, value *T) (bool, error) {
	keyStr, err := s.key(key)
	if err != nil {
		return false, fmt.Errorf("can not get key to read value from redis: %w", err)
	}

	data, err := s.client.Get(ctx, keyStr)

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

func (s *redisKvStore[T]) GetBatch(ctx context.Context, keys any, result any) ([]any, error) {
	return getBatch(ctx, keys, result, s.getChunk, s.settings.BatchSize)
}

func (s *redisKvStore[T]) getChunk(ctx context.Context, resultMap *refl.Map, keys []any) ([]any, error) {
	var err error

	missing := make([]any, 0)
	keyStrings := make([]string, len(keys))

	for i := 0; i < len(keyStrings); i++ {
		keyStrings[i], err = s.key(keys[i])
		if err != nil {
			return nil, fmt.Errorf("can not build string key: %w", err)
		}
	}

	items, err := s.client.MGet(ctx, keyStrings...)
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

func (s *redisKvStore[T]) Put(ctx context.Context, key any, value T) error {
	keyStr, bytes, err := s.marshalKeyValue(key, value)
	if err != nil {
		return fmt.Errorf("can not get key/value to write to redis: %w", err)
	}

	err = s.client.Set(ctx, keyStr, bytes, s.settings.Ttl)
	if err != nil {
		return fmt.Errorf("can not set value in redis store: %w", err)
	}

	return nil
}

func (s *redisKvStore[T]) marshalKeyValue(key any, value any) (keyStr string, bytes []byte, err error) {
	if bytes, err = Marshal(value); err != nil {
		return "", nil, fmt.Errorf("can not marshal value %T %v: %w", value, value, err)
	}

	if keyStr, err = s.key(key); err != nil {
		return "", nil, fmt.Errorf("can not get key to write value to redis: %w", err)
	}

	return keyStr, bytes, nil
}

func (s *redisKvStore[T]) PutBatch(ctx context.Context, values any) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)
	if err != nil {
		return fmt.Errorf("could not convert values from %T to map[any]any", values)
	}

	chunkSize := s.settings.BatchSize
	pairs := make([]any, 0, 2*chunkSize)
	for k, v := range mii {
		key, value, err := s.marshalKeyValue(k, v)
		if err != nil {
			return fmt.Errorf("PutBatch could not marshal key/value: %w", err)
		}
		pairs = append(pairs, key, value)

		if len(pairs) >= 2*chunkSize {
			err = s.flushChunk(ctx, pairs)
			if err != nil {
				return fmt.Errorf("failed to write batch to redis: %w", err)
			}
			pairs = make([]any, 0, chunkSize)
		}
	}

	return s.flushChunk(ctx, pairs)
}

func (s *redisKvStore[T]) flushChunk(ctx context.Context, pairs []any) error {
	if len(pairs) < 1 {
		return nil
	}

	pipe := s.client.Pipeline().TxPipeline()
	pipe.MSet(ctx, pairs)

	// setting ttl
	if s.settings.Ttl != 0 {
		for i := 0; i < len(pairs); i += 2 {
			keyStr, ok := pairs[i].(string)
			if !ok {
				return fmt.Errorf("setting ttl, failed to cast key to string: %v", pairs[i])
			}
			pipe.Expire(ctx, keyStr, s.settings.Ttl)
		}
	}

	_, err := pipe.Exec(ctx)

	return err
}

func (s *redisKvStore[T]) EstimateSize() *int64 {
	size, err := s.client.DBSize(context.Background())
	if err != nil {
		return nil
	}

	return &size
}

func (s *redisKvStore[T]) Delete(ctx context.Context, key any) error {
	keyStr, err := s.key(key)
	if err != nil {
		return fmt.Errorf("can not get key to delete value from redis: %w", err)
	}

	_, err = s.client.Del(ctx, keyStr)
	if err != nil {
		return fmt.Errorf("can not delete value from redis store: %w", err)
	}

	return nil
}

func (s *redisKvStore[T]) DeleteBatch(ctx context.Context, keys any) error {
	si, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return fmt.Errorf("could not convert keys from %T to []any: %w", keys, err)
	}

	redisKeys := make([]string, len(si))

	for i, key := range si {
		keyStr, err := s.key(key)
		if err != nil {
			return fmt.Errorf("can not get key to delete value from redis: %w", err)
		}

		redisKeys[i] = keyStr
	}

	_, err = s.client.Del(ctx, redisKeys...)
	if err != nil {
		return fmt.Errorf("can not delete values from redis store: %w", err)
	}

	return nil
}

func (s *redisKvStore[T]) key(key any) (string, error) {
	keyStr, err := CastKeyToString(key)
	if err != nil {
		return "", fmt.Errorf("can not cast key %T %v to string: %w", key, key, err)
	}

	keyStr = strings.Join([]string{
		s.settings.Project,
		s.settings.Family,
		s.settings.Group,
		"kvstore",
		s.settings.Name,
		keyStr,
	}, "-")

	return keyStr, nil
}
