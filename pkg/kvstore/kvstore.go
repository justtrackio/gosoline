package kvstore

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

type Settings struct {
	cfg.AppId
	DdbSettings    DdbSettings
	Name           string
	Ttl            time.Duration
	BatchSize      int
	MetricsEnabled bool
	InMemorySettings
}

type InMemorySettings struct {
	MaxSize        int64
	Buckets        uint32
	ItemsToPrune   uint32
	DeleteBuffer   uint32
	PromoteBuffer  uint32
	GetsPerPromote int32
}

//go:generate go run github.com/vektra/mockery/v2 --name KvStore
type KvStore[T any] interface {
	// Check if a key exists in the store.
	Contains(ctx context.Context, key any) (bool, error)
	// Retrieve a value from the store by the given key. If the value does
	// not exist, false is returned and value is not modified.
	// value should be a pointer to the model you want to retrieve.
	Get(ctx context.Context, key any, value *T) (bool, error)
	// Retrieve a set of values from the store. Each value is written to the
	// map in values at its key.  Values should be something which can be converted to map[interface{}]T.
	// Returns a list of missing keys in the store.
	GetBatch(ctx context.Context, keys any, values any) ([]any, error)
	// Write a value to the store
	Put(ctx context.Context, key any, value T) error
	// Write a batch of values to the store. Values should be something which
	// can be converted to map[interface{}]T.
	PutBatch(ctx context.Context, values any) error
	// Remove the value with the given key from the store
	Delete(ctx context.Context, key any) error
	// Remove all values with the given keys from the store
	DeleteBatch(ctx context.Context, keys any) error
}

//go:generate go run github.com/vektra/mockery/v2 --name SizedStore
type SizedStore[T any] interface {
	KvStore[T]
	// return an estimate about how many elements are currently in the store
	// returns nil if no estimate could be returned
	EstimateSize() *int64
}

type Factory[T any] func(elementFactory ElementFactory[T], settings *Settings) (KvStore[T], error)

type ElementFactory[T any] func(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (KvStore[T], error)

func buildFactory[T any](ctx context.Context, config cfg.Config, logger log.Logger) Factory[T] {
	return func(elementFactory ElementFactory[T], settings *Settings) (KvStore[T], error) {
		return elementFactory(ctx, config, logger, settings)
	}
}

func CastKeyToString(key interface{}) (string, error) {
	str, err := cast.ToStringE(key)

	if err == nil {
		return str, nil
	}

	return "", errors.Wrapf(err, "unknown type [%T] for kvstore key", key)
}

func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
