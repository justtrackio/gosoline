package env

import "github.com/justtrackio/gosoline/pkg/cfg"

type KafkaComponent struct {
	baseComponent
	brokerAddress string
}

func (c KafkaComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigSetting(
			"kafka", map[string]any{
				"connection": map[string]any{
					"default": map[string]any{
						"bootstrap":   []string{c.brokerAddress},
						"tls_enabled": false,
					},
				},
			}),
	}
}

func (c KafkaComponent) Address() string {
	return c.brokerAddress
}
