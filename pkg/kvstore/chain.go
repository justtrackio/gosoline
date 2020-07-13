package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/refl"
	"reflect"
)

type ChainKvStore struct {
	logger   mon.Logger
	factory  func(factory Factory, settings *Settings) KvStore
	chain    []KvStore
	settings *Settings

	missingCacheEnabled bool
	missingCache        *InMemoryKvStore
}

func NewChainKvStore(config cfg.Config, logger mon.Logger, missingCacheEnabled bool, settings *Settings) *ChainKvStore {
	settings.PadFromConfig(config)
	factory := buildFactory(config, logger)

	var missingCache *InMemoryKvStore
	if missingCacheEnabled {
		missingCache = NewInMemoryKvStore(config, logger, settings).(*InMemoryKvStore)
	}

	return NewChainKvStoreWithInterfaces(logger, factory, missingCacheEnabled, missingCache, settings)
}

func NewChainKvStoreWithInterfaces(logger mon.Logger, factory func(Factory, *Settings) KvStore, missingCacheEnabled bool, missingCache *InMemoryKvStore, settings *Settings) *ChainKvStore {
	return &ChainKvStore{
		logger:              logger,
		factory:             factory,
		chain:               make([]KvStore, 0),
		settings:            settings,
		missingCache:        missingCache,
		missingCacheEnabled: missingCacheEnabled,
	}
}

func (s *ChainKvStore) Add(elementFactory Factory) {
	store := s.factory(elementFactory, s.settings)
	s.AddStore(store)
}

func (s *ChainKvStore) AddStore(store KvStore) {
	s.chain = append(s.chain, store)
}

func (s *ChainKvStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	lastElementIndex := len(s.chain) - 1

	if s.missingCacheEnabled {
		// don't care about an error as there are more elements to come
		if exists, _ := s.missingCache.Contains(ctx, key); exists {
			return true, nil
		}
	}

	for i, element := range s.chain {
		exists, err := element.Contains(ctx, key)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return false, err
		}

		if exists {
			return true, nil
		}
	}

	return false, nil
}

// Get fills the passed value variable with the value from the underlying store.
// Returns a boolean whether the element has been found, and if an error occurred.
// If caching of missing values is enabled, always true is returned.
// The caller needs to check the passed value, if it was modified properly.
func (s *ChainKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	var err error
	var i int
	var exists bool

	if s.missingCacheEnabled {
		// don't care about an error as there are more elements to come
		if exists, _ = s.missingCache.Get(ctx, key, value); exists {
			return true, nil
		}
	}

	lastElementIndex := len(s.chain) - 1

	for i = 0; i < len(s.chain); i++ {
		exists, err = s.chain[i].Get(ctx, key, value)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return false, fmt.Errorf("could not get %s from kvstore %T: %w", key, s.chain[i], err)
		}

		if exists {
			break
		}
	}

	// Cache empty value if no result was found
	if s.missingCacheEnabled && !exists {
		// value must be a pointer, otherwise this fails
		element := reflect.New(reflect.TypeOf(value).Elem()).Interface()
		err = s.missingCache.Put(ctx, key, element)

		if err != nil {
			return false, fmt.Errorf("could not put %s to empty value cache %T, %w", key, s.chain[0], err)
		}

		return s.missingCache.Get(ctx, key, value)
	}

	if !exists {
		return false, nil
	}

	for i--; i >= 0; i-- {
		err = s.chain[i].Put(ctx, key, value)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return false, fmt.Errorf("could not put %s to kvstore %T: %w", key, s.chain[i], err)
		}
	}

	return true, nil
}

// GetBatch fills the given value map with values from the store.
// It returns an array of missing keys, and an error if one occurred.
// If caching of missing values is enabled, the array of missing keys has length 0.
// The value map will then contain nil for every key which was missing.
func (s *ChainKvStore) GetBatch(ctx context.Context, keys interface{}, values interface{}) ([]interface{}, error) {
	var i int
	var err error
	var missing []interface{}
	var lastElementIndex = len(s.chain) - 1

	refill := make(map[int][]interface{})
	missing, err = refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	if s.missingCacheEnabled {
		missing, _ = s.missingCache.GetBatch(ctx, missing, values)
	}

	if len(missing) == 0 {
		return missing, nil
	}

	for i = 0; i < len(s.chain); i++ {
		refill[i], err = s.chain[i].GetBatch(ctx, missing, values)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return nil, fmt.Errorf("could not get batch from kvstore %T: %w", s.chain[i], err)
		}

		if err != nil {
			s.logger.WithContext(ctx).Warnf("could not get batch from kvstore %T: %s", s.chain[i], err.Error())
			refill[i] = missing
		}

		missing = refill[i]

		if len(missing) == 0 {
			break
		}
	}

	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return nil, fmt.Errorf("can not cast result values to map[interface{}]interface{}: %w", err)
	}

	for i--; i >= 0; i-- {
		if len(refill[i]) == 0 {
			continue
		}

		missingInElement := make(map[interface{}]interface{})

		for _, key := range refill[i] {
			if val, ok := mii[key]; ok {
				missingInElement[key] = val
			}
		}

		if len(missingInElement) == 0 {
			continue
		}

		err = s.chain[i].PutBatch(ctx, missingInElement)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return nil, fmt.Errorf("could not put batch to kvstore %T: %w", s.chain[i], err)
		}
	}

	// store missing keys if enabled
	if s.missingCacheEnabled && len(missing) > 0 {
		missingValues := make(map[interface{}]interface{}, len(missing))

		resultMap, err := refl.MapOf(values)
		if err != nil {
			s.logger.WithContext(ctx).Warnf("could not interpret value map %T, %w", resultMap, err)
			return nil, err
		}

		for _, key := range missing {
			element := resultMap.NewElement()

			missingValues[key] = element
			err = resultMap.Set(key, element)

			if err != nil {
				s.logger.WithContext(ctx).Warnf("could not set empty value in value map %T, %w", resultMap, err)

				return nil, err
			}
		}

		err = s.missingCache.PutBatch(ctx, missingValues)

		if err != nil {
			s.logger.WithContext(ctx).Warnf("could not put batch to empty value cache %T, %w", s.chain[0], err.Error())

			return nil, err
		}

		missing = nil
	}

	return missing, nil
}

func (s *ChainKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	lastElementIndex := len(s.chain) - 1

	if s.missingCacheEnabled {
		err := s.missingCache.Delete(ctx, key)

		if err != nil {
			return fmt.Errorf("could not erase cached empty value for key %s: %w", key, err)
		}
	}

	for i := 0; i <= lastElementIndex; i++ {
		err := s.chain[i].Put(ctx, key, value)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return fmt.Errorf("could not put %s to kvstore %T: %w", key, s.chain[i], err)
		}
	}

	return nil
}

func (s *ChainKvStore) PutBatch(ctx context.Context, values interface{}) error {
	lastElementIndex := len(s.chain) - 1

	if s.missingCacheEnabled {
		mii, err := refl.InterfaceToMapInterfaceInterface(values)

		if err != nil {
			return fmt.Errorf("can not cast values to map[interface{}]interface{}: %w", err)
		}

		keys := make([]interface{}, 0, len(mii))
		for key := range mii {
			keys = append(keys, key)
		}

		err = s.missingCache.DeleteBatch(ctx, keys)

		if err != nil {
			return fmt.Errorf("could not erase cached empty values for key: %w", err)
		}
	}

	for i := 0; i <= lastElementIndex; i++ {
		err := s.chain[i].PutBatch(ctx, values)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return fmt.Errorf("could not put batch to kvstore %w", err)
		}
	}

	return nil
}
