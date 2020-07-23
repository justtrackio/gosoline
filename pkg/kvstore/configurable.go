package kvstore

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

const (
	TypeChain    = "chain"
	TypeDdb      = "ddb"
	TypeInMemory = "inMemory"
	TypeRedis    = "redis"
)

func NewConfigurableKvStore(config cfg.Config, logger mon.Logger, name string) KvStore {
	key := fmt.Sprintf("kvstore.%s.type", name)
	t := config.GetString(key)

	switch t {
	case TypeChain:
		return newKvStoreChainFromConfig(config, logger, name)
	default:
		logger.Fatalf(fmt.Errorf("invalid kvstore %s of type %s", name, t), "invalid kvstore %s of type %s", name, t)
	}

	return nil
}

type ChainConfiguration struct {
	Project     string        `cfg:"project"`
	Family      string        `cfg:"family"`
	Application string        `cfg:"application"`
	Elements    []string      `cfg:"elements" validate:"min=1"`
	Ttl         time.Duration `cfg:"ttl"`
	BatchSize   int           `cfg:"batch_size" default:"100" validate:"min=1"`
}

func newKvStoreChainFromConfig(config cfg.Config, logger mon.Logger, name string) KvStore {
	key := GetConfigurableKey(name)

	configuration := ChainConfiguration{}
	config.UnmarshalKey(key, &configuration)

	store := NewChainKvStore(config, logger, &Settings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		Name:      name,
		Ttl:       configuration.Ttl,
		BatchSize: configuration.BatchSize,
	})

	for _, element := range configuration.Elements {
		switch element {
		case TypeDdb:
			store.Add(NewDdbKvStore)
		case TypeInMemory:
			store.Add(NewInMemoryKvStore)
		case TypeRedis:
			store.Add(NewRedisKvStore)
		default:
			err := fmt.Errorf("invalid element type %s for kvstore chain", element)
			logger.Fatalf(err, err.Error())
		}

	}

	return store
}

func GetConfigurableKey(name string) string {
	return fmt.Sprintf("kvstore.%s", name)
}
