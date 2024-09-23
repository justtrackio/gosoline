package env

import (
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type DdbComponent struct {
	baseComponent
	logger         log.Logger
	ddbAddress     string
	namingSettings *ddb.TableNamingSettings
	toxiproxy      *toxiproxy.Proxy
}

func (c *DdbComponent) CfgOptions() []cfg.Option {
	clientEndpointKey := fmt.Sprintf("cloud.aws.dynamodb.clients.%s.endpoint", c.name)

	return []cfg.Option{
		cfg.WithConfigMap(map[string]any{
			"cloud.aws.defaults.credentials": map[string]any{
				"access_key_id":     DefaultAccessKeyID,
				"secret_access_key": DefaultSecretAccessKey,
				"session_token":     DefaultToken,
			},
		}),
		cfg.WithConfigSetting(clientEndpointKey, c.Endpoint()),
	}
}

func (c *DdbComponent) Address() string {
	return c.ddbAddress
}

func (c *DdbComponent) Endpoint() string {
	return fmt.Sprintf("http://%s", c.ddbAddress)
}

func (c *DdbComponent) Client() *dynamodb.Client {
	return dynamodb.NewFromConfig(
		aws.Config{
			Region:      "eu-central-1",
			Credentials: GetDefaultStaticCredentials(),
		},
		func(options *dynamodb.Options) {
			options.BaseEndpoint = gosoAws.NilIfEmpty(c.Endpoint())
		},
	)
}

func (c *DdbComponent) Repository(settings *ddb.Settings) (ddb.Repository, error) {
	tracer := tracing.NewNoopTracer()
	client := c.Client()
	tableName := ddb.GetTableNameWithSettings(settings, c.namingSettings)
	metadataFactory := ddb.NewMetadataFactoryWithInterfaces(settings, tableName)

	return ddb.NewWithInterfaces(c.logger, tracer, client, metadataFactory)
}

func (c *DdbComponent) Toxiproxy() *toxiproxy.Proxy {
	return c.toxiproxy
}
