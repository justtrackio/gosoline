package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/go-redis/redis"
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

type redisFactory struct {
}

func (f *redisFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("redis") {
		return nil
	}

	if manager.HasType(componentRedis) {
		return nil
	}

	if err := manager.Add(componentRedis, "default"); err != nil {
		return fmt.Errorf("can not add default ddb component: %w", err)
	}

	return nil
}

func (f *redisFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &redisSettings{}
}

func (f *redisFactory) ConfigureContainer(settings interface{}) *containerConfig {
	s := settings.(*redisSettings)

	return &containerConfig{
		Repository: "redis",
		Tag:        "6-alpine",
		PortBindings: portBindings{
			"6379/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *redisFactory) HealthCheck(_ interface{}) ComponentHealthCheck {
	return func(container *container) error {
		client := f.client(container)
		err := client.Ping().Err()

		return err
	}
}

func (f *redisFactory) Component(_ cfg.Config, _ mon.Logger, container *container, _ interface{}) (Component, error) {
	component := &redisComponent{
		address: f.address(container),
		client:  f.client(container),
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
