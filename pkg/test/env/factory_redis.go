package env

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	baseRedis "github.com/redis/go-redis/v9"
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
	lck     sync.Mutex
	clients map[string]*baseRedis.Client
}

func (f *redisFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("redis") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentRedis) {
		return nil
	}

	if has, err := manager.HasType(componentRedis); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	settings := &redisSettings{}
	if err := UnmarshalSettings(config, settings, componentRedis, "default"); err != nil {
		return fmt.Errorf("can not unmarshal redis settings: %w", err)
	}
	settings.Type = componentRedis

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default redis component: %w", err)
	}

	return nil
}

func (f *redisFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &redisSettings{}
}

func (f *redisFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	return ComponentContainerDescriptions{
		"main": {
			ContainerConfig: f.configureContainer(settings),
			HealthCheck:     f.healthCheck(),
		},
	}
}

func (f *redisFactory) configureContainer(settings any) *ContainerConfig {
	s := settings.(*redisSettings)

	return &ContainerConfig{
		Auth:       s.Image.Auth,
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		PortBindings: PortBindings{
			"main": {
				ContainerPort: 6379,
				HostPort:      s.Port,
				Protocol:      "tcp",
			},
		},
	}
}

func (f *redisFactory) healthCheck() ComponentHealthCheck {
	return func(container *Container) error {
		client := f.client(container)
		err := client.Ping(context.Background()).Err()

		return err
	}
}

func (f *redisFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*Container, _ any) (Component, error) {
	component := &RedisComponent{
		address: f.address(containers["main"]),
		client:  f.client(containers["main"]),
	}

	return component, nil
}

func (f *redisFactory) address(container *Container) string {
	binding := container.bindings["main"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)

	return address
}

func (f *redisFactory) client(container *Container) *baseRedis.Client {
	address := f.address(container)

	f.lck.Lock()
	defer f.lck.Unlock()

	if f.clients == nil {
		f.clients = make(map[string]*baseRedis.Client)
	}

	if _, ok := f.clients[address]; !ok {
		f.clients[address] = baseRedis.NewClient(&baseRedis.Options{
			Addr: address,
		})
	}

	return f.clients[address]
}
