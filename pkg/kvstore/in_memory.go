package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/karlseguin/ccache"
	"reflect"
	"sync/atomic"
	"time"
)

type InMemoryKvStore struct {
	cache     *ccache.Cache
	settings  *Settings
	cacheSize *int64
}

func NewInMemoryKvStore(_ cfg.Config, _ mon.Logger, settings *Settings) KvStore {
	return NewInMemoryKvStoreWithInterfaces(settings)
}

func NewInMemoryKvStoreWithInterfaces(settings *Settings) KvStore {
	cacheSize := new(int64)
	cache := ccache.New(ccache.Configure().OnDelete(func(item *ccache.Item) {
		// track how many items are still in the cache
		atomic.AddInt64(cacheSize, -1)
	}))

	if settings.Ttl.Nanoseconds() == 0 {
		settings.Ttl = time.Hour
	}

	return NewMetricStoreWithInterfaces(&InMemoryKvStore{
		cache:     cache,
		settings:  settings,
		cacheSize: cacheSize,
	}, settings)
}

func (s *InMemoryKvStore) Contains(_ context.Context, key interface{}) (bool, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return false, fmt.Errorf("can not build string key %T %v: %w", key, key, err)
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
		return false, fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	item := s.cache.Get(keyStr)

	if item == nil || item.Expired() {
		return false, nil
	}

	itemValue := item.Value()
	ri := reflect.ValueOf(itemValue)
	rv := reflect.ValueOf(value)

	if rv.Kind() != reflect.Ptr {
		return false, fmt.Errorf("the output value has to be a pointer, was %T", value)
	}

	rv = rv.Elem()
	rv.Set(ri)

	return true, nil
}

func (s *InMemoryKvStore) GetBatch(ctx context.Context, keys interface{}, values interface{}) ([]interface{}, error) {
	return getBatch(ctx, keys, values, s.getChunk, s.settings.BatchSize)
}

func (s *InMemoryKvStore) getChunk(ctx context.Context, resultMap *refl.Map, keys []interface{}) ([]interface{}, error) {
	missing := make([]interface{}, 0)

	for _, key := range keys {
		keyString, err := CastKeyToString(key)

		if err != nil {
			return nil, fmt.Errorf("can not build string key %T %v: %w", key, key, err)
		}

		element := resultMap.NewElement()
		ok, err := s.Get(ctx, key, element)

		if err != nil {
			return nil, fmt.Errorf("can not get batch element for key %s: %w", keyString, err)
		}

		if !ok {
			missing = append(missing, keyString)

			continue
		}

		if err := resultMap.Set(keyString, element); err != nil {
			return nil, fmt.Errorf("can not set new element on result map: %w", err)
		}
	}

	return missing, nil
}

func (s *InMemoryKvStore) Put(_ context.Context, key interface{}, value interface{}) error {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	rv := reflect.ValueOf(value)

	// make sure to store a copy, not a reference
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		value = rv.Interface()
	}

	s.cache.Set(keyStr, value, s.settings.Ttl)

	atomic.AddInt64(s.cacheSize, 1)

	return nil
}

func (s *InMemoryKvStore) PutBatch(ctx context.Context, values interface{}) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return fmt.Errorf("could not convert values from %T to map[interface{}]interface{}", values)
	}

	for k, v := range mii {
		if err = s.Put(ctx, k, v); err != nil {
			return fmt.Errorf("can not put value into in_memory store: %w", err)
		}
	}

	return nil
}

func (s *InMemoryKvStore) EstimateSize() *int64 {
	return mdl.Int64(atomic.LoadInt64(s.cacheSize))
}

func (s *InMemoryKvStore) Delete(_ context.Context, key interface{}) error {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	s.cache.Delete(keyStr)

	return nil
}

func (s *InMemoryKvStore) DeleteBatch(ctx context.Context, keys interface{}) error {
	si, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return fmt.Errorf("could not convert keys from %T to []interface{}: %w", keys, err)
	}

	for _, key := range si {
		if err = s.Delete(ctx, key); err != nil {
			return fmt.Errorf("can not remove value from in_memory store: %w", err)
		}
	}

	return nil
}
