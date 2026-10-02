package env

import (
	"context"
	"fmt"
	"strconv"
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
	ContainerBindingSettings
	DB     int  `cfg:"db" default:"0" validate:"min=0"`
	AutoDB bool `cfg:"auto_db" default:"false"`
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
	ports := PortBindings{
		"main": {ContainerPort: 6379, HostPort: s.Port, Protocol: "tcp"},
	}
	if s.isExternal() {
		return externalContainer(s.Host, ports)
	}

	return &ContainerConfig{
		Auth:         s.Image.Auth,
		Repository:   s.Image.Repository,
		Tag:          s.Image.Tag,
		PortBindings: ports,
	}
}

func (f *redisFactory) healthCheck() ComponentHealthCheck {
	return func(container *Container) error {
		client := f.client(container, 0)
		err := client.Ping(context.Background()).Err()

		return err
	}
}

func (f *redisFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*Container, settings any) (Component, error) {
	s := settings.(*redisSettings)
	if s.AutoDB && s.isExternal() && s.DB == 0 {
		db, err := f.allocateDatabase(containers["main"])
		if err != nil {
			return nil, err
		}
		s.DB = db
	}

	component := &RedisComponent{
		address: f.address(containers["main"]),
		client:  f.client(containers["main"], s.DB),
		db:      s.DB,
	}

	return component, nil
}

func (f *redisFactory) allocateDatabase(container *Container) (int, error) {
	client := f.client(container, 0)
	ctx := context.Background()
	databases, err := client.ConfigGet(ctx, "databases").Result()
	if err != nil {
		return 0, fmt.Errorf("can not read Redis database count: %w", err)
	}
	count, err := strconv.Atoi(databases["databases"])
	if err != nil {
		return 0, fmt.Errorf("invalid Redis database count: %w", err)
	}
	db, err := client.Incr(ctx, "gosoline:test:next-db").Result()
	if err != nil {
		return 0, fmt.Errorf("can not allocate Redis database: %w", err)
	}
	if db >= int64(count) {
		return 0, fmt.Errorf("Redis database allocation exhausted: database %d, configured count %d", db, count)
	}

	return int(db), nil
}

func (f *redisFactory) address(container *Container) string {
	binding := container.bindings["main"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)

	return address
}

func (f *redisFactory) client(container *Container, db int) *baseRedis.Client {
	address := f.address(container)
	key := fmt.Sprintf("%s/%d", address, db)

	f.lck.Lock()
	defer f.lck.Unlock()

	if f.clients == nil {
		f.clients = make(map[string]*baseRedis.Client)
	}

	if _, ok := f.clients[key]; !ok {
		f.clients[key] = baseRedis.NewClient(&baseRedis.Options{
			Addr: address,
			DB:   db,
		})
	}

	return f.clients[key]
}
