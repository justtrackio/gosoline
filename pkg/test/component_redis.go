package test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/go-redis/redis/v8"
)

type redisSettings struct {
	*mockSettings
	Port int `cfg:"port" default:"0"`
}

type redisComponent struct {
	mockComponentBase
	settings *redisSettings
	clients  *simpleCache
}

func (r *redisComponent) Boot(config cfg.Config, _ log.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string) {
	r.name = name
	r.runner = runner
	r.settings = &redisSettings{
		mockSettings: settings,
	}
	r.clients = &simpleCache{}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, r.settings)
}

func (r *redisComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_redis_%s", r.name)

	return r.runner.Run(containerName, &containerConfigLegacy{
		Repository: "redis",
		Tag:        "5-alpine",
		PortBindings: portBindingLegacy{
			"6379/tcp": fmt.Sprint(r.settings.Port),
		},
		PortMappings: portMappingLegacy{
			"6379/tcp": &r.settings.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &r.settings.Port,
			setHost:  &r.settings.Host,
		},
		HealthCheck: func() error {
			client := r.provideRedisClient()
			_, err := client.Ping(context.Background()).Result()

			return err
		},
		PrintLogs:   r.settings.Debug,
		ExpireAfter: r.settings.ExpireAfter,
	})
}

func (r *redisComponent) provideRedisClient() *redis.Client {
	return r.clients.New(r.name, func() interface{} {
		addr := fmt.Sprintf("%s:%d", r.settings.Host, r.settings.Port)

		return redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0, // use default DB
		})
	}).(*redis.Client)
}
