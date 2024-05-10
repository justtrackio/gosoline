package kvstore

import (
	"context"
	"fmt"
	"math/bits"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cache"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/karlseguin/ccache"
)

type InMemoryKvStore[T any] struct {
	cache     cache.Cache[T]
	settings  *Settings
	cacheSize *int64
}

func NewInMemoryKvStore[T any](_ context.Context, _ cfg.Config, _ log.Logger, settings *Settings) (KvStore[T], error) {
	if reflect.ValueOf(new(T)).Elem().Kind() == reflect.Pointer {
		return nil, fmt.Errorf("the generic type T should not be a pointer type but is of type %T", *new(T))
	}

	return NewInMemoryKvStoreWithInterfaces[T](settings), nil
}

func NewInMemoryKvStoreWithInterfaces[T any](settings *Settings) KvStore[T] {
	// make sure the config has some sensible values
	if settings.MaxSize <= 0 {
		settings.MaxSize = 5000
	}
	if settings.Buckets == 0 {
		settings.Buckets = 16
	} else if bits.OnesCount32(settings.Buckets) != 1 {
		// buckets needs to be a power of two
		exponent := 32 - bits.LeadingZeros32(settings.Buckets)
		if exponent == 32 {
			// user requested more than 2 billion buckets... hope the user knows what he is doing. give as many buckets as possible
			exponent = 31
		}
		settings.Buckets = 1 << exponent
	}
	if settings.ItemsToPrune == 0 {
		settings.ItemsToPrune = uint32(settings.MaxSize / 10)
	}
	if settings.DeleteBuffer == 0 {
		settings.DeleteBuffer = 1024
	}
	if settings.PromoteBuffer == 0 {
		settings.PromoteBuffer = 1024
	}
	if settings.GetsPerPromote <= 0 {
		settings.GetsPerPromote = 3
	}

	cacheSize := new(int64)
	trackDeletes := func(item *ccache.Item) {
		// track how many items are still in the cache
		atomic.AddInt64(cacheSize, -1)
	}

	cacheConfig := ccache.Configure().
		OnDelete(trackDeletes).
		MaxSize(settings.MaxSize).
		Buckets(settings.Buckets).
		ItemsToPrune(settings.ItemsToPrune).
		DeleteBuffer(settings.DeleteBuffer).
		PromoteBuffer(settings.PromoteBuffer).
		GetsPerPromote(settings.GetsPerPromote)

	ttl := settings.Ttl
	if ttl == 0 {
		ttl = time.Hour
	}

	baseCache := cache.NewWithConfiguration[T](*cacheConfig, ttl)

	return NewMetricStoreWithInterfaces[T](&InMemoryKvStore[T]{
		cache:     baseCache,
		settings:  settings,
		cacheSize: cacheSize,
	}, settings)
}

func (s *InMemoryKvStore[T]) Contains(_ context.Context, key any) (bool, error) {
	keyStr, err := CastKeyToString(key)
	if err != nil {
		return false, fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	return s.cache.Contains(keyStr), nil
}

func (s *InMemoryKvStore[T]) Get(_ context.Context, key any, value *T) (bool, error) {
	keyStr, err := CastKeyToString(key)
	if err != nil {
		return false, fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	item, ok := s.cache.Get(keyStr)
	if !ok {
		return false, nil
	}

	*value = item

	return true, nil
}

func (s *InMemoryKvStore[T]) GetBatch(ctx context.Context, keys any, values any) ([]any, error) {
	return getBatch(ctx, keys, values, s.getChunk, s.settings.BatchSize)
}

func (s *InMemoryKvStore[T]) getChunk(ctx context.Context, resultMap *refl.Map, keys []any) ([]any, error) {
	missing := make([]any, 0)

	for _, key := range keys {
		keyString, err := CastKeyToString(key)
		if err != nil {
			return nil, fmt.Errorf("can not build string key %T %v: %w", key, key, err)
		}

		element := new(T)
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

func (s *InMemoryKvStore[T]) Put(_ context.Context, key any, value T) error {
	keyStr, err := CastKeyToString(key)
	if err != nil {
		return fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	s.cache.Set(keyStr, value)

	atomic.AddInt64(s.cacheSize, 1)

	return nil
}

func (s *InMemoryKvStore[T]) PutBatch(ctx context.Context, values any) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)
	if err != nil {
		return fmt.Errorf("could not convert values from %T to map[any]any", values)
	}

	for k, v := range mii {
		if err = s.Put(ctx, k, v.(T)); err != nil {
			return fmt.Errorf("can not put value into in_memory store: %w", err)
		}
	}

	return nil
}

func (s *InMemoryKvStore[T]) EstimateSize() *int64 {
	return mdl.Box(atomic.LoadInt64(s.cacheSize))
}

func (s *InMemoryKvStore[T]) Delete(_ context.Context, key any) error {
	keyStr, err := CastKeyToString(key)
	if err != nil {
		return fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	s.cache.Expire(keyStr)

	return nil
}

func (s *InMemoryKvStore[T]) DeleteBatch(ctx context.Context, keys any) error {
	si, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return fmt.Errorf("could not convert keys from %T to []any: %w", keys, err)
	}

	for _, key := range si {
		if err = s.Delete(ctx, key); err != nil {
			return fmt.Errorf("can not remove value from in_memory store: %w", err)
		}
	}

	return nil
}
