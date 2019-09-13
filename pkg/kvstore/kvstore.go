package kvstore

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"time"
)

type Settings struct {
	cfg.AppId
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
