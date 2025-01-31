package kvstore

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/fixtures"
)

func NewNamedKvStoreFixture[T any](name any, value T) *fixtures.NamedFixture[*KvStoreFixture] {
	return &fixtures.NamedFixture[*KvStoreFixture]{
		Name: fmt.Sprint(name),
		Value: &KvStoreFixture{
			Key:   name,
			Value: value,
		},
	}
}
