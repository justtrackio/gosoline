package kafka

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/twmb/franz-go/pkg/sr"
)

// WithRegisteredSchema registers a schema before the integration test starts.
// This is necessary because gosoline expects all used schemas to be registered externally and to already exist in the schema registry.
func WithRegisteredSchema(s suite.TestingSuite, subject string, schema string, schemaType sr.SchemaType) suite.Option {
	return suite.WithEnvSetup(
		func() error {
			_, err := s.Env().Kafka("default").SchemaRegistryClient().
				CreateSchema(context.Background(), subject, sr.Schema{
					Schema: schema,
					Type:   schemaType,
				})

			return err
		},
	)
}

func WithKafkaBrokerPort(brokerPort int) suite.Option {
	return suite.WithConfigMap(
		map[string]any{
			"test": map[string]any{
				"components": map[string]any{
					"kafka": map[string]any{
						"default": map[string]any{
							"broker_port": brokerPort,
						},
					},
				},
			},
		},
	)
}
