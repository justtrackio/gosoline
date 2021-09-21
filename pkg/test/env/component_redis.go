package env

import (
	"github.com/go-redis/redis/v8"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

type redisComponent struct {
	baseComponent
	address string
	client  *redis.Client
}

func (c *redisComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigSetting("redis", map[string]interface{}{
			"default": map[string]interface{}{
				"dialer":  "tcp",
				"address": c.address,
			},
		}),
	}
}

func (c *redisComponent) Address() string {
	return c.address
}

func (c *redisComponent) Client() *redis.Client {
	return c.client
}
