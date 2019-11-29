package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/refl"
)

type ChainKvStore struct {
	logger   mon.Logger
	factory  func(factory Factory, settings *Settings) KvStore
	chain    []KvStore
	settings *Settings
}

func NewChainKvStore(config cfg.Config, logger mon.Logger, settings *Settings) *ChainKvStore {
	settings.PadFromConfig(config)
	factory := buildFactory(config, logger)

	return NewChainKvStoreWithInterfaces(logger, factory, settings)
}

func NewChainKvStoreWithInterfaces(logger mon.Logger, factory func(Factory, *Settings) KvStore, settings *Settings) *ChainKvStore {
	return &ChainKvStore{
		logger:   logger,
		factory:  factory,
		chain:    make([]KvStore, 0),
		settings: settings,
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

func (s *ChainKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	var err error
	var i int
	var exists bool

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

	if i == 0 {
		return missing, nil
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

	return missing, nil
}

func (s *ChainKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	lastElementIndex := len(s.chain) - 1

	for i, element := range s.chain {
		err := element.Put(ctx, key, value)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return fmt.Errorf("could not put %s to kvstore %T: %w", key, element, err)
		}
	}

	return nil
}

func (s *ChainKvStore) PutBatch(ctx context.Context, values interface{}) error {
	lastElementIndex := len(s.chain) - 1

	for i, element := range s.chain {
		err := element.PutBatch(ctx, values)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return fmt.Errorf("could not put batch to kvstore %w", err)
		}
	}

	return nil
}
