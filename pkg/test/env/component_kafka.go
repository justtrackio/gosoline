package env

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/admin"
)

type KafkaComponent struct {
	baseComponent
	client                admin.Client
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

func (c KafkaComponent) Client() admin.Client {
	return c.client
}

func (c KafkaComponent) BrokerAddress() string {
	return c.brokerAddress
}

func (c KafkaComponent) SchemaRegistryAddress() string {
	return c.schemaRegistryAddress
}
