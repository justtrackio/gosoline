package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"strconv"
	"time"
)

type Settings struct {
	Name string
	Ttl  time.Duration
}

type Factory func(config cfg.Config, logger mon.Logger, settings *Settings) KvStore

func buildFactory(config cfg.Config, logger mon.Logger) func(factory Factory, settings *Settings) KvStore {
	return func(factory Factory, settings *Settings) KvStore {
		return factory(config, logger, settings)
	}
}

type KvStore interface {
	Contains(ctx context.Context, key interface{}) (bool, error)
	Get(ctx context.Context, key interface{}, value interface{}) (bool, error)
	Put(ctx context.Context, key interface{}, value interface{}) error
}

func KeyToString(key interface{}) string {
	switch v := key.(type) {
	case int:
		return strconv.Itoa(v)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case string:
		return v
	}

	panic(fmt.Errorf("unknown type [%T] for kvstore key", key))
}
