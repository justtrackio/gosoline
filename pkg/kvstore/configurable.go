package kvstore

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"sync"
	"time"
)

const (
	TypeChain    = "chain"
	TypeDdb      = "ddb"
	TypeInMemory = "inMemory"
	TypeRedis    = "redis"
)

type ChainConfiguration struct {
	Project             string                `cfg:"project"`
	Family              string                `cfg:"family"`
	Application         string                `cfg:"application"`
	Type                string                `cfg:"type" default:"chain" validate:"eq=chain"`
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

func NewConfigurableKvStore(config cfg.Config, logger log.Logger, name string) (KvStore, error) {
	key := fmt.Sprintf("kvstore.%s.type", name)
	t := config.GetString(key)

	switch t {
	case TypeChain:
		return newKvStoreChainFromConfig(config, logger, name)
	}

	return nil, fmt.Errorf("invalid kvstore %s of type %s", name, t)
}

func newKvStoreChainFromConfig(config cfg.Config, logger log.Logger, name string) (KvStore, error) {
	key := GetConfigurableKey(name)

	configuration := ChainConfiguration{}
	config.UnmarshalKey(key, &configuration)

	store, err := NewChainKvStore(config, logger, configuration.MissingCacheEnabled, &Settings{
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
	})
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

func ProvideConfigurableKvStore(config cfg.Config, logger log.Logger, name string) (KvStore, error) {
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
