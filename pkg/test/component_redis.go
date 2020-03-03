package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/go-redis/redis"
	"log"
)

type redisSettings struct {
	*mockSettings
	Port int `cfg:"port"`
}

type redisComponent struct {
	name     string
	settings *redisSettings
	clients  *simpleCache
}

func (r *redisComponent) Boot(name string, config cfg.Config, settings *mockSettings) {
	r.name = name
	r.settings = &redisSettings{
		mockSettings: settings,
	}
	r.clients = &simpleCache{}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, r.settings)
}

func (r *redisComponent) Run(runner *dockerRunner) {
	defer log.Printf("%s component of type redis is ready", r.name)

	containerName := fmt.Sprintf("gosoline_test_redis_%s", r.name)

	runner.Run(containerName, containerConfig{
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
		PrintLogs: r.settings.Debug,
	})
}

func (r *redisComponent) ProvideClient(string) interface{} {
	return r.provideRedisClient()
}

func (r *redisComponent) provideRedisClient() *redis.Client {
	return r.clients.New("redis", func() interface{} {
		addr := fmt.Sprintf("%s:%d", r.settings.Host, r.settings.Port)

		return redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0, // use default DB
		})
	}).(*redis.Client)
}
