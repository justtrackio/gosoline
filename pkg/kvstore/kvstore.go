package kvstore

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"time"
)

type Settings struct {
	cfg.AppId
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

//go:generate mockery -name KvStore
type KvStore interface {
	// Check if a key exists in the store.
	Contains(ctx context.Context, key interface{}) (bool, error)
	// Retrieve a value from the store by the given key. If the value does
	// not exist, false is returned and value is not modified.
	// value should be a pointer to the model you want to retrieve.
	Get(ctx context.Context, key interface{}, value interface{}) (bool, error)
	// Retrieve a set of values from the store. Each value is written to the
	// map in values at its key. Returns a list of missing keys in the store.
	GetBatch(ctx context.Context, keys interface{}, values interface{}) ([]interface{}, error)
	// Write a value to the store
	Put(ctx context.Context, key interface{}, value interface{}) error
	// Write a batch of values to the store. Values should be something which
	// can be converted to map[interface{}]interface{}.
	PutBatch(ctx context.Context, values interface{}) error
	// Remove the value with the given key from the store
	Delete(ctx context.Context, key interface{}) error
	// Remove all values with the given keys from the store
	DeleteBatch(ctx context.Context, keys interface{}) error
}

//go:generate mockery -name SizedStore
type SizedStore interface {
	KvStore
	// return an estimate about how many elements are currently in the store
	// returns nil if no estimate could be returned
	EstimateSize() *int64
}

type Factory func(config cfg.Config, logger log.Logger, settings *Settings) (KvStore, error)

func buildFactory(config cfg.Config, logger log.Logger) func(factory Factory, settings *Settings) (KvStore, error) {
	return func(factory Factory, settings *Settings) (KvStore, error) {
		return factory(config, logger, settings)
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
