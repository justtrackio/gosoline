package env

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	baseRedis "github.com/redis/go-redis/v9"
)

type RedisComponent struct {
	baseComponent
	address string
	client  *baseRedis.Client
}

func (c *RedisComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigSetting("redis", map[string]any{
			"default": map[string]any{
				"dialer":  "tcp",
				"address": c.address,
			},
		}),
	}
}

func (c *RedisComponent) Address() string {
	return c.address
}

func (c *RedisComponent) Client() *baseRedis.Client {
	return c.client
}
