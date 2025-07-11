package kvstore

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/refl"
)

type emptyKvStore[T any] struct{}

func NewEmptyKvStore[T any]() KvStore[T] {
	return NewEmptyKvStoreWithInterfaces[T]()
}

func NewEmptyKvStoreWithInterfaces[T any]() KvStore[T] {
	return &emptyKvStore[T]{}
}

func (s *emptyKvStore[T]) Contains(_ context.Context, _ any) (bool, error) {
	return false, nil
}

func (s *emptyKvStore[T]) Get(_ context.Context, _ any, _ *T) (bool, error) {
	return false, nil
}

func (s *emptyKvStore[T]) GetBatch(_ context.Context, keys any, _ any) ([]any, error) {
	missing, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return nil, fmt.Errorf("could not convert keys from %T to []any: %w", keys, err)
	}

	return missing, nil
}

func (s *emptyKvStore[T]) Put(_ context.Context, _ any, _ T) error {
	return nil
}

func (s *emptyKvStore[T]) PutBatch(_ context.Context, _ any) error {
	return nil
}

func (s *emptyKvStore[T]) Delete(_ context.Context, _ any) error {
	return nil
}

func (s *emptyKvStore[T]) DeleteBatch(_ context.Context, _ any) error {
	return nil
}
