package env

import (
	"github.com/go-redis/redis/v8"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

type RedisComponent struct {
	baseComponent
	address string
	client  *redis.Client
}

func (c *RedisComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigSetting("redis", map[string]interface{}{
			"default": map[string]interface{}{
				"dialer":  "tcp",
				"address": c.address,
			},
		}),
	}
}

func (c *RedisComponent) Address() string {
	return c.address
}

func (c *RedisComponent) Client() *redis.Client {
	return c.client
}
