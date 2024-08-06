package kvstore

import (
	"context"
	"fmt"
	"reflect"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	TypeChain    = "chain"
	TypeDdb      = "ddb"
	TypeInMemory = "inMemory"
	TypeRedis    = "redis"
)

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
	t := config.GetString(key)

	switch t {
	case TypeChain:
		return newKvStoreChainFromConfig[T](ctx, config, logger, name)
	}

	return nil, fmt.Errorf("invalid kvstore %s of type %s", name, t)
}

func newKvStoreChainFromConfig[T any](ctx context.Context, config cfg.Config, logger log.Logger, name string) (KvStore[T], error) {
	key := GetConfigurableKey(name)

	configuration := ChainConfiguration{}
	config.UnmarshalKey(key, &configuration)

	store, err := NewChainKvStore[T](ctx, config, logger, configuration.MissingCacheEnabled, &Settings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Group:       configuration.Group,
			Application: configuration.Application,
		},
		DdbSettings:    configuration.Ddb,
		Name:           name,
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
