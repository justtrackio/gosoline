package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/refl"
)

type chainKvStore struct {
	logger   log.Logger
	factory  func(factory Factory, settings *Settings) (KvStore, error)
	chain    []KvStore
	settings *Settings

	missingCache KvStore
}

var noValue = &struct{}{}

func NewChainKvStore(config cfg.Config, logger log.Logger, missingCacheEnabled bool, settings *Settings) (*chainKvStore, error) {
	settings.PadFromConfig(config)
	factory := buildFactory(config, logger)

	var err error
	var missingCache KvStore

	if missingCacheEnabled {
		missingCacheSettings := *settings
		missingCacheSettings.Name = fmt.Sprintf("%s-missingCache", settings.Name)

		if missingCache, err = NewInMemoryKvStore(config, logger, &missingCacheSettings); err != nil {
			return nil, fmt.Errorf("can not create missing cache: %w", err)
		}
	} else {
		missingCache = NewEmptyKvStore()
	}

	return NewChainKvStoreWithInterfaces(logger, factory, missingCache, settings), nil
}

func NewChainKvStoreWithInterfaces(logger log.Logger, factory func(Factory, *Settings) (KvStore, error), missingCache KvStore, settings *Settings) *chainKvStore {
	return &chainKvStore{
		logger:       logger,
		factory:      factory,
		chain:        make([]KvStore, 0),
		settings:     settings,
		missingCache: missingCache,
	}
}

func (s *chainKvStore) Add(elementFactory Factory) error {
	store, err := s.factory(elementFactory, s.settings)
	if err != nil {
		return fmt.Errorf("can not create store: %w", err)
	}

	s.AddStore(store)

	return nil
}

func (s *chainKvStore) AddStore(store KvStore) {
	s.chain = append(s.chain, store)
}

func (s *chainKvStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	lastElementIndex := len(s.chain) - 1

	// check if we can short circuit the whole deal
	exists, err := s.missingCache.Contains(ctx, key)

	if err != nil {
		s.logger.WithContext(ctx).Warn("failed to read from missing value cache: %s", err.Error())
	}

	if exists {
		return false, nil
	}

	for i, element := range s.chain {
		exists, err := element.Contains(ctx, key)

		if err != nil {
			// return error only if last element fails
			if i == lastElementIndex {
				return false, fmt.Errorf("could not check existence of %s from kvstore %T: %w", key, element, err)
			}

			s.logger.WithContext(ctx).Warn("could not check existence of %s from kvstore %T: %s", key, element, err.Error())
		}

		if exists {
			return true, nil
		}
	}

	// Cache empty value if no result was found
	if err := s.missingCache.Put(ctx, key, noValue); err != nil {
		s.logger.WithContext(ctx).Warn("failed to write to missing value cache: %s", err.Error())
	}

	return false, nil
}

func (s *chainKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	// check if we can short circuit the whole deal
	exists, err := s.missingCache.Contains(ctx, key)

	if err != nil {
		s.logger.WithContext(ctx).Warn("failed to read from missing value cache: %s", err.Error())
	}

	if exists {
		return false, nil
	}

	lastElementIndex := len(s.chain) - 1
	foundInIndex := lastElementIndex + 1

	for i, element := range s.chain {
		var err error
		exists, err = element.Get(ctx, key, value)

		if err != nil {
			// return error only if last element fails
			if i == lastElementIndex {
				return false, fmt.Errorf("could not get %s from kvstore %T: %w", key, element, err)
			}

			s.logger.WithContext(ctx).Warn("could not get %s from kvstore %T: %s", key, element, err.Error())
		}

		if exists {
			foundInIndex = i

			break
		}
	}

	// Cache empty value if no result was found
	if !exists {
		if err := s.missingCache.Put(ctx, key, noValue); err != nil {
			s.logger.WithContext(ctx).Warn("failed to write to missing value cache: %s", err.Error())
		}

		return false, nil
	}

	// propagate to the lower cache levels
	for i := foundInIndex - 1; i >= 0; i-- {
		err := s.chain[i].Put(ctx, key, value)

		if err != nil {
			s.logger.WithContext(ctx).Warn("could not put %s to kvstore %T: %s", key, s.chain[i], err.Error())
		}
	}

	return true, nil
}

func (s *chainKvStore) GetBatch(ctx context.Context, keys interface{}, values interface{}) ([]interface{}, error) {
	todo, err := refl.InterfaceToInterfaceSlice(keys)
	var cachedMissing []interface{}

	if err != nil {
		return nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	cachedMissingMap := make(map[string]interface{})
	todo, err = s.missingCache.GetBatch(ctx, todo, cachedMissingMap)

	if err != nil {
		s.logger.WithContext(ctx).Warn("failed to read from missing value cache: %s", err.Error())
	}

	for k := range cachedMissingMap {
		cachedMissing = append(cachedMissing, k)
	}

	if len(todo) == 0 {
		return cachedMissing, nil
	}

	lastElementIndex := len(s.chain) - 1
	refill := make(map[int][]interface{})
	foundInIndex := lastElementIndex + 1

	for i, element := range s.chain {
		var err error
		refill[i], err = element.GetBatch(ctx, todo, values)

		if err != nil {
			// return error only if last element fails
			if i == lastElementIndex {
				return nil, fmt.Errorf("could not get batch from kvstore %T: %w", element, err)
			}

			s.logger.WithContext(ctx).Warn("could not get batch from kvstore %T: %s", element, err.Error())
			refill[i] = todo
		}

		todo = refill[i]

		if len(todo) == 0 {
			foundInIndex = i

			break
		}
	}

	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return nil, fmt.Errorf("can not cast result values from %T to map[interface{}]interface{}: %w", values, err)
	}

	// propagate to the lower cache levels
	for i := foundInIndex - 1; i >= 0; i-- {
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

		if err != nil {
			s.logger.WithContext(ctx).Warn("could not put batch to kvstore %T: %s", s.chain[i], err.Error())
		}
	}

	// store missing keys
	if len(todo) > 0 {
		missingValues := make(map[interface{}]interface{}, len(todo))

		for _, key := range todo {
			missingValues[key] = noValue
		}

		err = s.missingCache.PutBatch(ctx, missingValues)

		if err != nil {
			s.logger.WithContext(ctx).Warn("could not put batch to empty value cache: %w", err.Error())
		}
	}

	missing := make([]interface{}, 0, len(todo)+len(cachedMissing))
	missing = append(missing, todo...)
	missing = append(missing, cachedMissing...)

	return missing, nil
}

func (s *chainKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	lastElementIndex := len(s.chain) - 1

	for i := 0; i <= lastElementIndex; i++ {
		err := s.chain[i].Put(ctx, key, value)

		if err != nil {
			// return error only if last element fails
			if i == lastElementIndex {
				return fmt.Errorf("could not put %s to kvstore %T: %w", key, s.chain[i], err)
			}

			s.logger.WithContext(ctx).Warn("could not put %s to kvstore %T: %s", key, s.chain[i], err.Error())
		}
	}

	// remove the value from the missing value cache only after we persisted it
	// otherwise, we might remove it, some other thread adds it again and then we insert
	// it into the backing stores
	if err := s.missingCache.Delete(ctx, key); err != nil {
		s.logger.WithContext(ctx).Warn("could not erase cached empty value for key %s: %s", key, err.Error())
	}

	return nil
}

func (s *chainKvStore) PutBatch(ctx context.Context, values interface{}) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return fmt.Errorf("can not cast values from %T to map[interface{}]interface{}: %w", values, err)
	}

	lastElementIndex := len(s.chain) - 1

	for i := 0; i <= lastElementIndex; i++ {
		err := s.chain[i].PutBatch(ctx, mii)

		if err != nil {
			// return error only if last element fails
			if i == lastElementIndex {
				return fmt.Errorf("could not put batch to kvstore %T: %w", s.chain[i], err)
			}

			s.logger.WithContext(ctx).Warn("could not put batch to kvstore %T: %s", s.chain[i], err.Error())
		}
	}

	for key := range mii {
		if err := s.missingCache.Delete(ctx, key); err != nil {
			s.logger.WithContext(ctx).Warn("could not erase cached empty value for key %T %v: %s", key, key, err.Error())
		}
	}

	return nil
}

func (s *chainKvStore) Delete(ctx context.Context, key interface{}) error {
	for _, store := range s.chain {
		err := store.Delete(ctx, key)

		if err != nil {
			// even if we do not fail at the last index, we can't leave something
			// in a cache but not in the backend store

			return fmt.Errorf("could not delete %s from kvstore %T: %w", key, store, err)
		}
	}

	return nil
}

func (s *chainKvStore) DeleteBatch(ctx context.Context, keys interface{}) error {
	for _, store := range s.chain {
		err := store.DeleteBatch(ctx, keys)

		if err != nil {
			// even if we do not fail at the last index, we can't leave something
			// in a cache but not in the backend store

			return fmt.Errorf("could not batch delete from kvstore %T: %w", store, err)
		}
	}

	return nil
}
