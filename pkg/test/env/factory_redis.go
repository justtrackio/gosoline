package env

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentRedis] = new(redisFactory)
}

const componentRedis = "redis"

type redisSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port int `cfg:"port" default:"0"`
}

type redisFactory struct{}

func (f *redisFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("redis") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentRedis) {
		return nil
	}

	if manager.HasType(componentRedis) {
		return nil
	}

	settings := &redisSettings{}
	UnmarshalSettings(config, settings, componentRedis, "default")
	settings.Type = componentRedis

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default redis component: %w", err)
	}

	return nil
}

func (f *redisFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &redisSettings{}
}

func (f *redisFactory) DescribeContainers(settings interface{}) componentContainerDescriptions {
	return componentContainerDescriptions{
		"main": {
			containerConfig: f.configureContainer(settings),
			healthCheck:     f.healthCheck(),
		},
	}
}

func (f *redisFactory) configureContainer(settings interface{}) *containerConfig {
	s := settings.(*redisSettings)

	return &containerConfig{
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		PortBindings: portBindings{
			"6379/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *redisFactory) healthCheck() ComponentHealthCheck {
	return func(container *container) error {
		client := f.client(container)
		err := client.Ping(context.Background()).Err()

		return err
	}
}

func (f *redisFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*container, _ interface{}) (Component, error) {
	component := &RedisComponent{
		address: f.address(containers["main"]),
		client:  f.client(containers["main"]),
	}

	return component, nil
}

func (f *redisFactory) address(container *container) string {
	binding := container.bindings["6379/tcp"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)

	return address
}

func (f *redisFactory) client(container *container) *redis.Client {
	address := f.address(container)

	client := redis.NewClient(&redis.Options{
		Addr: address,
	})

	return client
}
