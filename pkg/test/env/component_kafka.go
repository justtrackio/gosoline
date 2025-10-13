package env

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/admin"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
)

type KafkaComponent struct {
	baseComponent
	adminClient           admin.Client
	schemaRegistryClient  schemaRegistry.Client
	brokerAddress         string
	schemaRegistryAddress string
}

func (c KafkaComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigSetting(
			"kafka", map[string]any{
				"connection": map[string]any{
					"default": map[string]any{
						"brokers":                 []string{c.brokerAddress},
						"schema_registry_address": c.schemaRegistryAddress,
						"tls_enabled":             false,
					},
				},
			}),
	}
}

func (c KafkaComponent) AdminClient() admin.Client {
	return c.adminClient
}

func (c KafkaComponent) SchemaRegistryClient() schemaRegistry.Client {
	return c.schemaRegistryClient
}

func (c KafkaComponent) BrokerAddress() string {
	return c.brokerAddress
}

func (c KafkaComponent) SchemaRegistryAddress() string {
	return c.schemaRegistryAddress
}
