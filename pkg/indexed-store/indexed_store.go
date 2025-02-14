package indexed_store

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/spf13/cast"
	"time"
)

// An IndexedStore is something comparable to a dynamo db table with global secondary indices.
// There is a default table with a single primary key (like in a kv store) and arbitrary many indices
// with a main key and zero or more range keys. There might be a limit on the number of range keys supported by an implementation,
// for example an IndexedStore backed by a ddb table will only allow up to a single range key.
//go:generate mockery --name IndexedStore
type IndexedStore interface {
	// Contains checks if a key exists in the store.
	Contains(ctx context.Context, key interface{}) (bool, error)
	// ContainsInIndex looks in an index for a value with the given key and range keys (if any are specified)
	ContainsInIndex(ctx context.Context, index string, key interface{}, rangeKeys ...interface{}) (bool, error)
	// Get retrieves a value from the store by the given key. If the value does not exist, nil is returned.
	Get(ctx context.Context, key interface{}) (BaseValue, error)
	// GetFromIndex retrieves a value from an index by the given key and range keys (if any are specified).
	// If the value does not exist, nil is returned. The value is converted to its base value before it is returned.
	GetFromIndex(ctx context.Context, index string, key interface{}, rangeKeys ...interface{}) (BaseValue, error)
	// GetBatch retrieves a set of values from the store. If an item is not found, it is not added to the list.
	// The order of the list is unspecified and not necessarily the same as the order of the keys.
	GetBatch(ctx context.Context, keys interface{}) ([]BaseValue, error)
	// GetBatchWithMissing is like GetBatch, but also returns a list of values which could not be found.
	GetBatchWithMissing(ctx context.Context, keys interface{}) ([]BaseValue, []interface{}, error)
	// GetBatchFromIndex retrieves a set of values from an index. If an item is not found, it is not added to the list.
	// The order of the list is unspecified and not necessarily the same as the order of the keys.
	// The keys and range keys need to be convertible to []interface{} and of the same length.
	GetBatchFromIndex(ctx context.Context, index string, keys interface{}, rangeKeys ...interface{}) ([]BaseValue, error)
	// GetBatchWithMissingFromIndex is like GetBatchFromIndex, but also returns a list of values which could not be found.
	GetBatchWithMissingFromIndex(ctx context.Context, index string, keys interface{}, rangeKeys ...interface{}) ([]BaseValue, []MissingValue, error)
	// Put writes a value to the store. You can only write to the main store and not to an index directly.
	Put(ctx context.Context, value BaseValue) error
	// PutBatch does the same as calling Put in a loop, but more efficiently.
	PutBatch(ctx context.Context, values interface{}) error
	// Delete removes the value with the given key from the store. You can only remove values from the main store and not
	// an index directly.
	Delete(ctx context.Context, key interface{}) error
	// DeleteBatch does the same as calling Delete in a loop, but more efficiently.
	DeleteBatch(ctx context.Context, keys interface{}) error
}

//go:generate mockery --name SizedStore
type SizedStore interface {
	IndexedStore
	// EstimateSize return an estimate about how many elements are currently in the store.
	// It returns nil if no estimate can be returned.
	EstimateSize() *int64
}

type MissingValue struct {
	Key       interface{}
	RangeKeys []interface{}
}

type BaseValue interface {
	GetId() interface{}
}

type IndexValue interface {
	ToBaseValue() BaseValue
}

type Settings struct {
	cfg.AppId
	Name           string
	Model          BaseValue
	Indices        []IndexSettings
	Ttl            time.Duration
	BatchSize      int
	MetricsEnabled bool
	InMemorySettings
}

type IndexSettings struct {
	Name  string
	Model IndexValue
}

type InMemorySettings struct {
	MaxSize        int64
	Buckets        uint32
	ItemsToPrune   uint32
	DeleteBuffer   uint32
	PromoteBuffer  uint32
	GetsPerPromote int32
}

func CastKeyToString(key interface{}) (string, error) {
	str, err := cast.ToStringE(key)

	if err == nil {
		return str, nil
	}

	return "", fmt.Errorf("unknown type [%T] for indexed store key: %w", key, err)
}
