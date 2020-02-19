package test

import (
	"fmt"
	"github.com/go-redis/redis"
	"log"
)

type redisConfig struct {
	Debug bool   `mapstructure:"debug"`
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
}

var redisConfigs map[string]*redisConfig
var redisClients simpleCache

func init() {
	redisClients = simpleCache{}
	redisConfigs = make(map[string]*redisConfig)
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
	redisConfigs[name] = config

	containerName := fmt.Sprintf("gosoline_test_redis_%s", name)

	runContainer(containerName, ContainerConfig{
		Repository: "redis",
		Tag:        "5-alpine",
		PortBindings: PortBinding{
			"6379/tcp": fmt.Sprint(config.Port),
		},

		HealthCheck: func() error {
			client := ProvideRedisClient(name)
			_, err := client.Ping().Result()

			return err
		},
		PrintLogs: config.Debug,
	})
}

func ProvideRedisClient(name string) *redis.Client {
	return redisClients.New(name, func() interface{} {
		addr := fmt.Sprintf("%s:%d", redisConfigs[name].Host, redisConfigs[name].Port)

		return redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0, // use default DB
		})
	}).(*redis.Client)
}
