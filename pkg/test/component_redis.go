package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/go-redis/redis"
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

func (r *redisComponent) Boot(config cfg.Config, _ mon.Logger, runner *dockerRunner, settings *mockSettings, name string) {
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

	return r.runner.Run(containerName, containerConfig{
		Repository: "redis",
		Tag:        "5-alpine",
		PortBindings: portBinding{
			"6379/tcp": fmt.Sprint(r.settings.Port),
		},
		PortMappings: map[string]*int{
			"6379/tcp": &r.settings.Port,
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
