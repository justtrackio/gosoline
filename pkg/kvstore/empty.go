package kvstore

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/refl"
)

type emptyKvStore struct{}

func NewEmptyKvStore() KvStore {
	return NewEmptyKvStoreWithInterfaces()
}

func NewEmptyKvStoreWithInterfaces() KvStore {
	return &emptyKvStore{}
}

func (s *emptyKvStore) Contains(_ context.Context, _ interface{}) (bool, error) {
	return false, nil
}

func (s *emptyKvStore) Get(_ context.Context, _ interface{}, _ interface{}) (bool, error) {
	return false, nil
}

func (s *emptyKvStore) GetBatch(_ context.Context, keys interface{}, _ interface{}) ([]interface{}, error) {
	missing, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return nil, fmt.Errorf("could not convert keys from %T to []interface{}: %w", keys, err)
	}

	return missing, nil
}

func (s *emptyKvStore) Put(_ context.Context, _ interface{}, _ interface{}) error {
	return nil
}

func (s *emptyKvStore) PutBatch(_ context.Context, _ interface{}) error {
	return nil
}

func (s *emptyKvStore) Delete(_ context.Context, _ interface{}) error {
	return nil
}

func (s *emptyKvStore) DeleteBatch(_ context.Context, _ interface{}) error {
	return nil
}
