package env

import (
	"context"
	"fmt"
	"strconv"
	"sync"

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
	ContainerBindingSettings
	UseExternalContainer bool `cfg:"use_external_container" default:"false"`
}

type redisFactory struct {
	lck     sync.Mutex
	clients map[string]*redis.Client
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

func (f *redisFactory) DescribeContainers(settings any) componentContainerDescriptions {
	return componentContainerDescriptions{
		"main": {
			containerConfig: f.configureContainer(settings),
			healthCheck:     f.healthCheck(),
		},
	}
}

func (f *redisFactory) configureContainer(settings any) *containerConfig {
	s := settings.(*redisSettings)

	if s.UseExternalContainer {
		return &containerConfig{
			UseExternalContainer: true,
			ContainerBindings: containerBindings{
				"6379/tcp": containerBinding{
					host: s.Host,
					port: strconv.Itoa(s.Port),
				},
			},
		}
	}

	// ensure to use a free port for the new container
	s.Port = 0

	return &containerConfig{
		Auth:       s.Image.Auth,
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

func (f *redisFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*container, _ any) (Component, error) {
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

	f.lck.Lock()
	defer f.lck.Unlock()

	if f.clients == nil {
		f.clients = make(map[string]*redis.Client)
	}

	if _, ok := f.clients[address]; !ok {
		f.clients[address] = redis.NewClient(&redis.Options{
			Addr: address,
		})
	}

	return f.clients[address]
}
