package kvstore

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"sync"
	"time"
)

const (
	TypeChain    = "chain"
	TypeDdb      = "ddb"
	TypeInMemory = "inMemory"
	TypeRedis    = "redis"
)

type Configuration struct {
	Project             string                `cfg:"project"`
	Family              string                `cfg:"family"`
	Application         string                `cfg:"application"`
	Type                string                `cfg:"type" default:"chain" validate:"oneof=chain redis ddb inMemory"`
	Elements            []string              `cfg:"elements" validate:"min=1"`
	Ttl                 time.Duration         `cfg:"ttl"`
	BatchSize           int                   `cfg:"batch_size" default:"100" validate:"min=1"`
	MissingCacheEnabled bool                  `cfg:"missing_cache_enabled" default:"false"`
	MetricsEnabled      bool                  `cfg:"metrics_enabled" default:"false"`
	InMemory            InMemoryConfiguration `cfg:"in_memory"`
}

type InMemoryConfiguration struct {
	MaxSize        int64  `cfg:"max_size" default:"5000"`
	Buckets        uint32 `cfg:"buckets" default:"16"`
	ItemsToPrune   uint32 `cfg:"items_to_prune" default:"500"`
	DeleteBuffer   uint32 `cfg:"delete_buffer" default:"1024"`
	PromoteBuffer  uint32 `cfg:"promote_buffer" default:"1024"`
	GetsPerPromote int32  `cfg:"gets_per_promote" default:"3"`
}

func NewConfigurableKvStore(config cfg.Config, logger mon.Logger, name string) (KvStore, error) {
	key := fmt.Sprintf("kvstore.%s.type", name)
	t := config.GetString(key)

	switch t {
	case TypeChain:
		return newKvStoreChainFromConfig(config, logger, name)
	case TypeDdb:
		return newKvStoreDdbFromConfig(config, logger, name)
	case TypeRedis:
		return newKvStoreRedisFromConfig(config, logger, name)
	case TypeInMemory:
		return newKvStoreInMemoryFromConfig(config, logger, name)
	}

	return nil, fmt.Errorf("invalid kvstore %s of type %s", name, t)
}

func newKvStoreChainFromConfig(config cfg.Config, logger mon.Logger, name string) (KvStore, error) {
	configuration, settings := getConfiguration(config, name)

	store, err := NewChainKvStore(config, logger, configuration.MissingCacheEnabled, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create chain store: %w", err)
	}

	for _, element := range configuration.Elements {
		switch element {
		case TypeDdb:
			if err := store.Add(NewDdbKvStore); err != nil {
				return nil, fmt.Errorf("can not add ddb store: %w", err)
			}
		case TypeInMemory:
			if err := store.Add(NewInMemoryKvStore); err != nil {
				return nil, fmt.Errorf("can not add inMemory store: %w", err)
			}
		case TypeRedis:
			if err := store.Add(NewRedisKvStore); err != nil {
				return nil, fmt.Errorf("can not add redis store: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid element type %s for kvstore chain", element)
		}

	}

	return store, nil
}

func newKvStoreDdbFromConfig(config cfg.Config, logger mon.Logger, name string) (KvStore, error) {
	_, settings := getConfiguration(config, name)

	store, err := NewDdbKvStore(config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create ddb store: %w", err)
	}

	return store, nil
}

func newKvStoreRedisFromConfig(config cfg.Config, logger mon.Logger, name string) (KvStore, error) {
	_, settings := getConfiguration(config, name)

	store, err := NewRedisKvStore(config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create ddb store: %w", err)
	}

	return store, nil
}

func newKvStoreInMemoryFromConfig(config cfg.Config, logger mon.Logger, name string) (KvStore, error) {
	_, settings := getConfiguration(config, name)

	store, err := NewInMemoryKvStore(config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create ddb store: %w", err)
	}

	return store, nil
}

func getConfiguration(config cfg.Config, name string) (Configuration, *Settings) {
	key := GetConfigurableKey(name)

	configuration := Configuration{}
	config.UnmarshalKey(key, &configuration)

	return configuration, &Settings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Application: configuration.Application,
		},
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
	}
}

func GetConfigurableKey(name string) string {
	return fmt.Sprintf("kvstore.%s", name)
}

var configurableKvStoreLock = sync.Mutex{}
var configurableKvStores = map[string]KvStore{}

func ResetConfigurableKvStores() {
	configurableKvStoreLock.Lock()
	defer configurableKvStoreLock.Unlock()

	configurableKvStores = map[string]KvStore{}
}

func ProvideConfigurableKvStore(config cfg.Config, logger mon.Logger, name string) (KvStore, error) {
	configurableKvStoreLock.Lock()
	defer configurableKvStoreLock.Unlock()

	if _, ok := configurableKvStores[name]; ok {
		return configurableKvStores[name], nil
	}

	var err error
	configurableKvStores[name], err = NewConfigurableKvStore(config, logger, name)

	if err != nil {
		return nil, err
	}

	return configurableKvStores[name], nil
}
