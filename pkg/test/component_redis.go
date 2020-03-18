package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/go-redis/redis"
)

type redisSettings struct {
	*mockSettings
	Port int `cfg:"port"`
}

type redisComponent struct {
	name     string
	settings *redisSettings
	clients  *simpleCache
	runner   *dockerRunner
}

func (r *redisComponent) Boot(config cfg.Config, runner *dockerRunner, settings *mockSettings, name string) {
	r.name = name
	r.runner = runner
	r.settings = &redisSettings{
		mockSettings: settings,
	}
	r.clients = &simpleCache{}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, r.settings)
}

func (r *redisComponent) Start() {
	containerName := fmt.Sprintf("gosoline_test_redis_%s", r.name)

	r.runner.Run(containerName, containerConfig{
		Repository: "redis",
		Tag:        "5-alpine",
		PortBindings: portBinding{
			"6379/tcp": fmt.Sprint(r.settings.Port),
		},

		HealthCheck: func() error {
			client := r.provideRedisClient()
			_, err := client.Ping().Result()

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
