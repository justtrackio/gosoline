package env

import (
	"github.com/applike/gosoline/pkg/application"
	"github.com/go-redis/redis"
)

type redisComponent struct {
	baseComponent
	address string
	client  *redis.Client
}

func (c *redisComponent) AppOptions() []application.Option {
	return []application.Option{
		application.WithConfigSetting("redis", map[string]interface{}{
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
