package env

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

type dynamoDbComponent struct {
	baseComponent
	binding      ContainerBinding
	region       string
	tablePattern string
}

func (c *dynamoDbComponent) Address() string {
	return fmt.Sprintf("http://%s", c.binding.getAddress())
}

func (c *dynamoDbComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigMap(map[string]any{
			"cloud.aws.dynamodb.clients.default": map[string]any{
				"credentials": map[string]any{
					"access_key_id":     DefaultAccessKeyID,
					"secret_access_key": DefaultSecretAccessKey,
					"session_token":     DefaultToken,
				},
				"endpoint":   c.Address(),
				"region":     c.region,
				"purge_type": "drop_table",
				"naming": map[string]any{
					"table_pattern": c.tablePattern,
				},
			},
		}),
	}
}

func (c *dynamoDbComponent) DdbClient() *dynamodb.Client {
	return dynamodb.NewFromConfig(aws.Config{
		Region: c.region, Credentials: GetDefaultStaticCredentials(),
	}, func(options *dynamodb.Options) {
		options.BaseEndpoint = aws.String(c.Address())
	})
}
