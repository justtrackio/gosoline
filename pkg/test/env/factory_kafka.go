package env

import (
	"context"
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	kafkaAdmin "github.com/justtrackio/gosoline/pkg/kafka/admin"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentKafka] = new(kafkaFactory)
}

const componentKafka = "kafka"

type kafkaFactory struct{}

type kafkaSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	BrokerPort         int `cfg:"broker_port" default:"9092"` // we can't set this to 0 to get a random port because we need a specific port in the container run config
	SchemaRegistryPort int `cfg:"schema_registry_port" default:"0"`
}

var _ componentFactory = &kafkaFactory{}

func (f *kafkaFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("test.components.kafka") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentKafka) {
		return nil
	}

	if has, err := manager.HasType(componentKafka); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	settings := &kafkaSettings{}
	if err := UnmarshalSettings(config, settings, componentKafka, "default"); err != nil {
		return fmt.Errorf("can not unmarshal kafka settings: %w", err)
	}
	settings.Type = componentKafka

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default kafka component: %w", err)
	}

	return nil
}

func (f *kafkaFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &kafkaSettings{}
}

func (f *kafkaFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	descriptions := ComponentContainerDescriptions{
		"main": {
			ContainerConfig:  f.configureContainer(settings),
			HealthCheck:      f.healthCheck(),
			ShutdownCallback: nil,
		},
	}

	return descriptions
}

func (f *kafkaFactory) healthCheck() ComponentHealthCheck {
	return func(container *Container) error {
		ctx := context.Background()

		client, err := kafkaAdmin.NewClient(ctx, log.NewLogger(), []string{f.brokerAddress(container)})
		if err != nil {
			return fmt.Errorf("failed to create kafka admin client: %w", err)
		}

		list, err := client.ListTopics(ctx)
		if err != nil {
			return fmt.Errorf("failed to list topics: %w", err)
		}

		if list.Error() != nil {
			return fmt.Errorf("failed to list topics: %w", list.Error())
		}

		resp, err := http.Get(fmt.Sprintf("http://%s/config", f.schemaRegistryAddress(container)))
		if err != nil {
			return fmt.Errorf("can not connect to schema registry: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("schema registry status code error: %d %s", resp.StatusCode, resp.Status)
		}

		return nil
	}
}

func (f *kafkaFactory) configureContainer(settings any) *ContainerConfig {
	s := settings.(*kafkaSettings)
	hostName := "redpanda"

	return &ContainerConfig{
		RunnerType: RunnerTypeLocal,
		Hostname:   hostName,
		Auth:       s.Image.Auth,
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		PortBindings: PortBindings{
			"main": {
				ContainerPort: 9092,
				HostPort:      s.BrokerPort,
				Protocol:      "tcp",
			},
			"schema-registry": {
				ContainerPort: 8081,
				HostPort:      s.SchemaRegistryPort,
				Protocol:      "tcp",
			},
		},
		Cmd: []string{
			"redpanda start",
			"--smp 1",
			"--overprovisioned",
			fmt.Sprintf("--kafka-addr internal://0.0.0.0:19092,external://0.0.0.0:%d", 9092),
			fmt.Sprintf("--advertise-kafka-addr internal://%s:19092,external://localhost:%d", hostName, s.BrokerPort),
			"--schema-registry-addr internal://0.0.0.0:18081,external://0.0.0.0:8081",
			"--mode dev-container",
		},
	}
}

func (f *kafkaFactory) brokerAddress(container *Container) string {
	binding := container.bindings["main"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)

	return address
}

func (f *kafkaFactory) schemaRegistryAddress(container *Container) string {
	binding := container.bindings["schema-registry"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)

	return address
}

func (f *kafkaFactory) Component(_ cfg.Config, logger log.Logger, containers map[string]*Container, _ any) (Component, error) {
	main := containers["main"]

	adminClient, err := kafkaAdmin.NewClient(context.Background(), logger, []string{f.brokerAddress(main)})
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka admin client: %w", err)
	}

	schemaRegistryClient, err := schemaRegistry.NewClient(f.schemaRegistryAddress(main))
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka admin client: %w", err)
	}

	return &KafkaComponent{
		baseComponent:         baseComponent{},
		adminClient:           adminClient,
		schemaRegistryClient:  schemaRegistryClient,
		brokerAddress:         f.brokerAddress(main),
		schemaRegistryAddress: f.schemaRegistryAddress(main),
	}, nil
}
