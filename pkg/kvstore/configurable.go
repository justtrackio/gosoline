package kvstore

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

const (
	TypeChain    = "chain"
	TypeDdb      = "ddb"
	TypeInMemory = "inMemory"
	TypeRedis    = "redis"
)

type ChainConfiguration struct {
	ModelId             mdl.ModelId           `cfg:"model_id"`
	Type                string                `cfg:"type" default:"chain" validate:"eq=chain"`
	Elements            []string              `cfg:"elements" validate:"min=1"`
	Ddb                 DdbSettings           `cfg:"ddb"`
	Ttl                 time.Duration         `cfg:"ttl"`
	BatchSize           int                   `cfg:"batch_size" default:"100" validate:"min=1"`
	MissingCacheEnabled bool                  `cfg:"missing_cache_enabled" default:"false"`
	MetricsEnabled      bool                  `cfg:"metrics_enabled" default:"false"`
	InMemory            InMemoryConfiguration `cfg:"in_memory"`
	Redis               RedisConfiguration    `cfg:"redis"`
}

type InMemoryConfiguration struct {
	MaxSize        int64  `cfg:"max_size" default:"5000"`
	Buckets        uint32 `cfg:"buckets" default:"16"`
	ItemsToPrune   uint32 `cfg:"items_to_prune" default:"500"`
	DeleteBuffer   uint32 `cfg:"delete_buffer" default:"1024"`
	PromoteBuffer  uint32 `cfg:"promote_buffer" default:"1024"`
	GetsPerPromote int32  `cfg:"gets_per_promote" default:"3"`
}

type RedisConfiguration struct {
	DB         int    `cfg:"db" default:"0"`
	KeyPattern string `cfg:"key_pattern,nodecode" default:"{app.namespace}-kvstore-{store}-{key}"`
}

type kvStoreAppCtxKey string

func ProvideConfigurableKvStore[T any](ctx context.Context, config cfg.Config, logger log.Logger, name string) (KvStore[T], error) {
	key := fmt.Sprintf("%s-kvStore", name)

	return appctx.Provide(ctx, kvStoreAppCtxKey(key), func() (KvStore[T], error) {
		return NewConfigurableKvStore[T](ctx, config, logger, name)
	})
}

func NewConfigurableKvStore[T any](ctx context.Context, config cfg.Config, logger log.Logger, name string) (KvStore[T], error) {
	if reflect.ValueOf(new(T)).Elem().Kind() == reflect.Pointer {
		return nil, fmt.Errorf("the generic type T should not be a pointer type but is of type %T", *new(T))
	}

	key := fmt.Sprintf("kvstore.%s.type", name)
	t, err := config.GetString(key)
	if err != nil {
		return nil, fmt.Errorf("could not get type for kvstore %s: %w", name, err)
	}

	if t == TypeChain {
		return newKvStoreChainFromConfig[T](ctx, config, logger, name)
	}

	return nil, fmt.Errorf("invalid kvstore %s of type %s", name, t)
}

func newKvStoreChainFromConfig[T any](ctx context.Context, config cfg.Config, logger log.Logger, name string) (KvStore[T], error) {
	key := GetConfigurableKey(name)

	configuration := ChainConfiguration{}
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kvstore configuration for %s: %w", name, err)
	}

	// Set the ModelId.Name to the kvstore name
	modelId := configuration.ModelId
	modelId.Name = name

	store, err := NewChainKvStore[T](ctx, config, logger, configuration.MissingCacheEnabled, &Settings{
		ModelId:        modelId,
		DdbSettings:    configuration.Ddb,
		Ttl:            configuration.Ttl,
		BatchSize:      configuration.BatchSize,
		MetricsEnabled: configuration.MetricsEnabled,
		InMemorySettings: InMemorySettings{
			MaxSize:        configuration.InMemory.MaxSize,
			Buckets:        configuration.InMemory.Buckets,
			ItemsToPrune:   configuration.InMemory.ItemsToPrune,
			DeleteBuffer:   configuration.InMemory.DeleteBuffer,
			PromoteBuffer:  configuration.InMemory.PromoteBuffer,
			GetsPerPromote: configuration.InMemory.GetsPerPromote,
		},
		RedisSettings: RedisSettings{
			KeyPrefixPattern: configuration.Redis.KeyPattern,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("can not create chain store: %w", err)
	}

	for _, element := range configuration.Elements {
		switch element {
		case TypeDdb:
			if err := store.Add(NewDdbKvStore[T]); err != nil {
				return nil, fmt.Errorf("can not add ddb store: %w", err)
			}
		case TypeInMemory:
			if err := store.Add(NewInMemoryKvStore[T]); err != nil {
				return nil, fmt.Errorf("can not add inMemory store: %w", err)
			}
		case TypeRedis:
			if err := store.Add(NewRedisKvStore[T]); err != nil {
				return nil, fmt.Errorf("can not add redis store: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid element type %s for kvstore chain", element)
		}
	}

	return store, nil
}

func GetConfigurableKey(name string) string {
	return fmt.Sprintf("kvstore.%s", name)
}
