package kvstore

import (
	"context"
	"fmt"
	"reflect"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/refl"
)

type ChainKvStore[T any] interface {
	KvStore[T]
	Add(elementFactory ElementFactory[T]) error
	AddStore(store KvStore[T])
}

type chainKvStore[T any] struct {
	logger   log.Logger
	factory  Factory[T]
	chain    []KvStore[T]
	settings *Settings

	missingCache KvStore[T]
}

func NewChainKvStore[T any](ctx context.Context, config cfg.Config, logger log.Logger, missingCacheEnabled bool, settings *Settings) (ChainKvStore[T], error) {
	if reflect.ValueOf(new(T)).Elem().Kind() == reflect.Pointer {
		return nil, fmt.Errorf("the generic type T should not be a pointer type but is of type %T", *new(T))
	}

	var err error
	if err = settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("can not pad settings from config: %w", err)
	}
	factory := buildFactory[T](ctx, config, logger)

	var missingCache KvStore[T]

	if missingCacheEnabled {
		missingCacheSettings := *settings
		missingCacheSettings.Name = fmt.Sprintf("%s-missingCache", settings.Name)

		if missingCache, err = NewInMemoryKvStore[T](ctx, config, logger, &missingCacheSettings); err != nil {
			return nil, fmt.Errorf("can not create missing cache: %w", err)
		}
	} else {
		missingCache = NewEmptyKvStore[T]()
	}

	return NewChainKvStoreWithInterfaces[T](logger, factory, missingCache, settings), nil
}

func NewChainKvStoreWithInterfaces[T any](logger log.Logger, factory Factory[T], missingCache KvStore[T], settings *Settings) ChainKvStore[T] {
	return &chainKvStore[T]{
		logger:       logger,
		factory:      factory,
		chain:        make([]KvStore[T], 0),
		settings:     settings,
		missingCache: missingCache,
	}
}

func (s *chainKvStore[T]) Add(elementFactory ElementFactory[T]) error {
	store, err := s.factory(elementFactory, s.settings)
	if err != nil {
		return fmt.Errorf("can not create store: %w", err)
	}

	s.AddStore(store)

	return nil
}

func (s *chainKvStore[T]) AddStore(store KvStore[T]) {
	s.chain = append(s.chain, store)
}

func (s *chainKvStore[T]) Contains(ctx context.Context, key any) (bool, error) {
	return s.Get(ctx, key, new(T))
}

func (s *chainKvStore[T]) Get(ctx context.Context, key any, value *T) (bool, error) {
	// check if we can short circuit the whole deal
	exists, err := s.missingCache.Contains(ctx, key)
	if err != nil {
		s.logger.Warn(ctx, "failed to read from missing value cache: %s", err.Error())
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

			s.logger.Warn(ctx, "could not get %s from kvstore %T: %s", key, element, err.Error())
		}

		if exists {
			foundInIndex = i

			break
		}
	}

	// Cache empty value if no result was found
	if !exists {
		if err := s.missingCache.Put(ctx, key, *new(T)); err != nil {
			s.logger.Warn(ctx, "failed to write to missing value cache: %s", err.Error())
		}

		return false, nil
	}

	// propagate to the lower cache levels
	for i := foundInIndex - 1; i >= 0; i-- {
		err := s.chain[i].Put(ctx, key, *value)
		if err != nil {
			s.logger.Warn(ctx, "could not put %s to kvstore %T: %s", key, s.chain[i], err.Error())
		}
	}

	return true, nil
}

func (s *chainKvStore[T]) GetBatch(ctx context.Context, keys any, values any) ([]any, error) {
	todo, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	todo, cachedMissing, err := s.getKeysFromMissingCache(ctx, todo)
	if err != nil {
		return nil, err
	}

	if len(todo) == 0 {
		return cachedMissing, nil
	}

	todo, refill, foundInIndex, err := s.getFromChain(ctx, todo, values)
	if err != nil {
		return nil, err
	}

	err = s.propagateToLowerCacheLevels(ctx, values, refill, foundInIndex)
	if err != nil {
		return nil, err
	}

	s.storeMissingKeys(ctx, todo)

	missing := make([]any, 0, len(todo)+len(cachedMissing))
	missing = append(missing, todo...)
	missing = append(missing, cachedMissing...)

	return missing, nil
}

func (s *chainKvStore[T]) getKeysFromMissingCache(ctx context.Context, todo []any) (remainingTodo []any, cachedMissing []any, err error) {
	cachedMissingMap := make(map[string]any)
	remainingTodo, err = s.missingCache.GetBatch(ctx, todo, cachedMissingMap)
	if err != nil {
		s.logger.Warn(ctx, "failed to read from missing value cache: %s", err.Error())
	}

	cachedMissing = make([]any, 0, len(cachedMissingMap))
	for k := range cachedMissingMap {
		cachedMissing = append(cachedMissing, k)
	}

	return remainingTodo, cachedMissing, nil
}

func (s *chainKvStore[T]) getFromChain(ctx context.Context, todo []any, values any) (remainingTodo []any, refill map[int][]any, foundInIndex int, err error) {
	lastElementIndex := len(s.chain) - 1
	refill = make(map[int][]any)
	foundInIndex = lastElementIndex + 1
	remainingTodo = todo

	for i, element := range s.chain {
		refill[i], err = element.GetBatch(ctx, remainingTodo, values)
		if err != nil {
			// return error only if last element fails
			if i == lastElementIndex {
				return nil, nil, 0, fmt.Errorf("could not get batch from kvstore %T: %w", element, err)
			}

			s.logger.Warn(ctx, "could not get batch from kvstore %T: %s", element, err.Error())
			refill[i] = remainingTodo
		}

		remainingTodo = refill[i]

		if len(remainingTodo) == 0 {
			foundInIndex = i

			break
		}
	}

	return remainingTodo, refill, foundInIndex, nil
}

func (s *chainKvStore[T]) propagateToLowerCacheLevels(ctx context.Context, values any, refill map[int][]any, foundInIndex int) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)
	if err != nil {
		return fmt.Errorf("can not cast result values from %T to map[any]any: %w", values, err)
	}

	// propagate to the lower cache levels
	for i := foundInIndex - 1; i >= 0; i-- {
		if len(refill[i]) == 0 {
			continue
		}

		missingInElement := make(map[any]T)

		for _, key := range refill[i] {
			if val, ok := mii[key]; ok {
				missingInElement[key] = val.(T)
			}
		}

		if len(missingInElement) == 0 {
			continue
		}

		err = s.chain[i].PutBatch(ctx, missingInElement)
		if err != nil {
			s.logger.Warn(ctx, "could not put batch to kvstore %T: %s", s.chain[i], err.Error())
		}
	}

	return nil
}

func (s *chainKvStore[T]) storeMissingKeys(ctx context.Context, todo []any) {
	if len(todo) > 0 {
		missingValues := make(map[any]any, len(todo))

		for _, key := range todo {
			missingValues[key] = *new(T)
		}

		err := s.missingCache.PutBatch(ctx, missingValues)
		if err != nil {
			s.logger.Warn(ctx, "could not put batch to empty value cache: %w", err.Error())
		}
	}
}

func (s *chainKvStore[T]) Put(ctx context.Context, key any, value T) error {
	lastElementIndex := len(s.chain) - 1

	for i := 0; i <= lastElementIndex; i++ {
		err := s.chain[i].Put(ctx, key, value)
		if err != nil {
			// return error only if last element fails
			if i == lastElementIndex {
				return fmt.Errorf("could not put %s to kvstore %T: %w", key, s.chain[i], err)
			}

			s.logger.Warn(ctx, "could not put %s to kvstore %T: %s", key, s.chain[i], err.Error())
		}
	}

	// remove the value from the missing value cache only after we persisted it
	// otherwise, we might remove it, some other thread adds it again and then we insert
	// it into the backing stores
	if err := s.missingCache.Delete(ctx, key); err != nil {
		s.logger.Warn(ctx, "could not erase cached empty value for key %s: %s", key, err.Error())
	}

	return nil
}

func (s *chainKvStore[T]) PutBatch(ctx context.Context, values any) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)
	if err != nil {
		return fmt.Errorf("can not cast values from %T to map[any]any: %w", values, err)
	}

	lastElementIndex := len(s.chain) - 1

	for i := 0; i <= lastElementIndex; i++ {
		err := s.chain[i].PutBatch(ctx, mii)
		if err != nil {
			// return error only if last element fails
			if i == lastElementIndex {
				return fmt.Errorf("could not put batch to kvstore %T: %w", s.chain[i], err)
			}

			s.logger.Warn(ctx, "could not put batch to kvstore %T: %s", s.chain[i], err.Error())
		}
	}

	for key := range mii {
		if err := s.missingCache.Delete(ctx, key); err != nil {
			s.logger.Warn(ctx, "could not erase cached empty value for key %T %v: %s", key, key, err.Error())
		}
	}

	return nil
}

func (s *chainKvStore[T]) Delete(ctx context.Context, key any) error {
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

func (s *chainKvStore[T]) DeleteBatch(ctx context.Context, keys any) error {
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
