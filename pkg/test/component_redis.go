package test

import (
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"sync"
)

type redisConfig struct {
	Port int `mapstructure:"port"`
}

func runRedis(name string, config configInput) {
	wait.Add(1)
	go doRunRedis(name, config)
}

func doRunRedis(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type redis is ready", name)

	config := &redisConfig{}
	unmarshalConfig(configMap, config)

	containerName := fmt.Sprintf("gosoline_test_%s_redis", name)
	runContainer(containerName, ContainerConfig{
		Repository: "redis",
		Tag:        "5-alpine",
		Env: []string{
			"discovery.type=single-node",
		},
		PortBindings: PortBinding{
			"6379/tcp": fmt.Sprint(config.Port),
		},

		HealthCheck: func() error {
			client := getRedisClient(config)
			_, err := client.Ping().Result()

			return err
		},
	})
}

var redisClient = struct {
	sync.Mutex
	instance *redis.Client
}{}

func getRedisClient(config *redisConfig) *redis.Client {
	redisClient.Lock()
	defer redisClient.Unlock()

	if redisClient.instance != nil {
		return redisClient.instance
	}

	addr := fmt.Sprintf("localhost:%d", config.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	redisClient.instance = client

	return redisClient.instance
}
