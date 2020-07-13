package kvstore

import (
	"context"
	"github.com/applike/gosoline/pkg/refl"
)

type EmptyKvStore struct {
}

func NewEmptyKvStore() KvStore {
	return NewEmptyKvStoreWithInterfaces()
}

func NewEmptyKvStoreWithInterfaces() *EmptyKvStore {
	return &EmptyKvStore{}
}

func (s *EmptyKvStore) Contains(_ context.Context, _ interface{}) (bool, error) {
	return false, nil
}

func (s *EmptyKvStore) Get(_ context.Context, _ interface{}, _ interface{}) (bool, error) {
	return false, nil
}

func (s *EmptyKvStore) GetBatch(_ context.Context, keys interface{}, _ interface{}) ([]interface{}, error) {
	missing, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return nil, err
	}

	return missing, nil
}

func (s *EmptyKvStore) Put(_ context.Context, key interface{}, value interface{}) error {
	return nil
}

func (s *EmptyKvStore) PutBatch(ctx context.Context, values interface{}) error {
	return nil
}
