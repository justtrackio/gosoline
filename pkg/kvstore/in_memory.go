package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/karlseguin/ccache"
	"reflect"
	"time"
)

type InMemoryKvStore struct {
	cache    *ccache.Cache
	settings *Settings
}

func NewInMemoryKvStore(_ cfg.Config, _ mon.Logger, settings *Settings) KvStore {
	return NewInMemoryKvStoreWithInterfaces(settings)
}

func NewInMemoryKvStoreWithInterfaces(settings *Settings) *InMemoryKvStore {
	cache := ccache.New(ccache.Configure())

	return &InMemoryKvStore{
		cache:    cache,
		settings: settings,
	}
}

func (s *InMemoryKvStore) Contains(_ context.Context, key interface{}) (bool, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return false, err
	}

	item := s.cache.Get(keyStr)

	if item == nil {
		return false, nil
	}

	expired := item.Expired()

	return !expired, nil
}

func (s *InMemoryKvStore) Get(_ context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return false, err
	}

	item := s.cache.Get(keyStr)

	if item == nil {
		return false, nil
	}

	if item.Expired() {
		return false, nil
	}

	itemValue := item.Value()
	ri := reflect.ValueOf(itemValue)
	rv := reflect.ValueOf(value)

	if rv.Kind() != reflect.Ptr {
		return false, fmt.Errorf("the output value has to be a pointer")
	}

	rv = rv.Elem()
	rv.Set(ri)

	return true, nil
}

func (s *InMemoryKvStore) GetBatch(ctx context.Context, keys interface{}, values interface{}) ([]interface{}, error) {
	return getBatch(ctx, keys, values, s.getChunk, s.settings.BatchSize)
}

func (s *InMemoryKvStore) getChunk(ctx context.Context, resultMap *refl.Map, keys []interface{}) ([]interface{}, error) {
	var err error

	missing := make([]interface{}, 0)
	keyStrings := make([]string, len(keys))

	for i := 0; i < len(keyStrings); i++ {
		keyStrings[i], err = CastKeyToString(keys[i])

		if err != nil {
			return nil, fmt.Errorf("can not build string key: %w", err)
		}
	}

	for _, key := range keyStrings {
		element := resultMap.NewElement()
		ok, err := s.Get(ctx, key, element)

		if err != nil {
			return nil, fmt.Errorf("can not get batch element for key %s: %w", key, err)
		}

		if !ok {
			missing = append(missing, key)
			continue
		}

		if err := resultMap.Set(key, element); err != nil {
			return nil, fmt.Errorf("can not set new element on result map: %w", err)
		}
	}

	return missing, nil
}

func (s *InMemoryKvStore) Put(_ context.Context, key interface{}, value interface{}) error {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return err
	}

	s.cache.Set(keyStr, value, time.Hour)

	return nil
}

func (s *InMemoryKvStore) PutBatch(ctx context.Context, values interface{}) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return fmt.Errorf("could not convert values to map[interface{}]interface{}")
	}

	for k, v := range mii {
		if err = s.Put(ctx, k, v); err != nil {
			return fmt.Errorf("can not put value into in_memory store: %w", err)
		}
	}

	return nil
}
