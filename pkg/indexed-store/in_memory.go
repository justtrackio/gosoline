package indexed_store

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/karlseguin/ccache"
	"math/bits"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

type InMemoryIndexedStore struct {
	cache     *ccache.Cache
	settings  *Settings
	cacheSize *int64
}

func NewInMemoryIndexedStore(_ cfg.Config, _ log.Logger, settings *Settings) (IndexedStore, error) {
	return NewInMemoryIndexedStoreWithInterfaces(settings), nil
}

func NewInMemoryIndexedStoreWithInterfaces(settings *Settings) IndexedStore {
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
	cache := ccache.New(cacheConfig)

	if settings.Ttl.Nanoseconds() == 0 {
		settings.Ttl = time.Hour
	}

	return NewMetricStoreWithInterfaces(&InMemoryIndexedStore{
		cache:     cache,
		settings:  settings,
		cacheSize: cacheSize,
	}, settings)
}

func (s *InMemoryIndexedStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	value, err := s.Get(ctx, key)

	return value != nil, err
}

func (s *InMemoryIndexedStore) ContainsInIndex(ctx context.Context, index string, key interface{}, rangeKeys ...interface{}) (bool, error) {
	value, err := s.GetFromIndex(ctx, index, key, rangeKeys...)

	return value != nil, err
}

func (s *InMemoryIndexedStore) Get(_ context.Context, key interface{}) (BaseValue, error) {
	keyStr, err := s.buildKey(nil, key)

	if err != nil {
		return nil, err
	}

	item := s.cache.Get(keyStr)

	if item == nil || item.Expired() {
		return nil, nil
	}

	return item.Value().(BaseValue), nil
}

func (s *InMemoryIndexedStore) GetFromIndex(_ context.Context, index string, key interface{}, rangeKeys ...interface{}) (BaseValue, error) {
	keyStr, err := s.buildKey(&index, key, rangeKeys...)

	if err != nil {
		return nil, err
	}

	item := s.cache.Get(keyStr)

	if item == nil || item.Expired() {
		return nil, nil
	}

	return item.Value().(BaseValue), nil
}

func (s *InMemoryIndexedStore) GetBatch(ctx context.Context, keys interface{}) ([]BaseValue, error) {
	values, _, err := s.GetBatchWithMissing(ctx, keys)

	return values, err
}

func (s *InMemoryIndexedStore) GetBatchWithMissing(ctx context.Context, keys interface{}) ([]BaseValue, []interface{}, error) {
	panic("implement me")
}

func (s *InMemoryIndexedStore) GetBatchFromIndex(ctx context.Context, index string, keys interface{}, rangeKeys ...interface{}) ([]BaseValue, error) {
	values, _, err := s.GetBatchWithMissingFromIndex(ctx, index, keys, rangeKeys...)

	return values, err
}

func (s *InMemoryIndexedStore) GetBatchWithMissingFromIndex(ctx context.Context, index string, keys interface{}, rangeKeys ...interface{}) ([]BaseValue, []MissingValue, error) {
	panic("implement me")
}

func (s *InMemoryIndexedStore) Put(_ context.Context, value BaseValue) error {
	key := value.GetId()
	keyStr, err := s.buildKey(nil, key)

	if err != nil {
		return fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	rv := reflect.ValueOf(value)

	// make sure to store a copy, not a reference
	if rv.Kind() == reflect.Ptr {
		// TODO: does this actually work?
		rv = rv.Elem()
		newPtr := reflect.New(rv.Type())
		newPtr.Elem().Set(rv)
		value = newPtr.Interface().(BaseValue)
	}

	s.cache.Set(keyStr, value, s.settings.Ttl)

	atomic.AddInt64(s.cacheSize, 1)

	return nil
}

func (s *InMemoryIndexedStore) PutBatch(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)

	if err != nil {
		return fmt.Errorf("could not convert values from %T to []interface{}", values)
	}

	for _, v := range valuesSlice {
		if bv, ok := v.(BaseValue); !ok {
			return fmt.Errorf("can not cast %T to BaseValue", v)
		} else if err = s.Put(ctx, bv); err != nil {
			return fmt.Errorf("can not put value into in_memory store: %w", err)
		}
	}

	return nil
}

func (s *InMemoryIndexedStore) Delete(_ context.Context, key interface{}) error {
	keyStr, err := s.buildKey(nil, key)

	if err != nil {
		return fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	}

	s.cache.Delete(keyStr)

	return nil
}

func (s *InMemoryIndexedStore) DeleteBatch(ctx context.Context, keys interface{}) error {
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

func (s *InMemoryIndexedStore) EstimateSize() *int64 {
	return mdl.Int64(atomic.LoadInt64(s.cacheSize))
}

func (s *InMemoryIndexedStore) buildKey(index *string, key interface{}, rangeKeys ...interface{}) (string, error) {
	var parts []string
	if index == nil {
		parts = append(parts, "main")
	} else {
		parts = append(parts, "index", *index)
	}

	if keyStr, err := CastKeyToString(key); err != nil {
		return "", fmt.Errorf("can not build string key %T %v: %w", key, key, err)
	} else {
		parts = append(parts, keyStr)
	}

	for _, rangeKey := range rangeKeys {
		if rangeKeyStr, err := CastKeyToString(rangeKey); err != nil {
			return "", fmt.Errorf("can not build string range key %T %v: %w", rangeKey, rangeKey, err)
		} else {
			parts = append(parts, rangeKeyStr)
		}
	}

	// U+2063 is INVISIBLE SEPARATOR and should normally not occur in keys or index names and therefore make collisions quite rare
	return strings.Join(parts, "\u2063"), nil
}
