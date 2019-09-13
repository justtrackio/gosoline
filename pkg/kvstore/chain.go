package kvstore

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/pkg/errors"
)

type ChainKvStore struct {
	factory  func(factory Factory, settings *Settings) KvStore
	chain    []KvStore
	settings *Settings
}

func NewChainKvStore(config cfg.Config, logger mon.Logger, settings *Settings) *ChainKvStore {
	return &ChainKvStore{
		factory:  buildFactory(config, logger),
		chain:    make([]KvStore, 0),
		settings: settings,
	}
}

func (s *ChainKvStore) Add(elementFactory Factory) {
	element := s.factory(elementFactory, s.settings)
	s.chain = append(s.chain, element)
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

func (s *ChainKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	lastElementIndex := len(s.chain) - 1

	for i, element := range s.chain {
		err := element.Put(ctx, key, value)

		// return error only if last element fails
		if err != nil && i == lastElementIndex {
			return errors.Wrapf(err, "could not put %s to kvstore %T", key, element)
		}
	}

	return nil
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
			return false, errors.Wrapf(err, "could not get %s from kvstore %T", key, s.chain[i])
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
			return false, errors.Wrapf(err, "could not put %s to kvstore %T", key, s.chain[i])
		}
	}

	return true, nil
}
