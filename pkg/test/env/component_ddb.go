package env

import (
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/applike/gosoline/pkg/cfg"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DdbComponent struct {
	baseComponent
	logger     log.Logger
	ddbAddress string
	toxiproxy  *toxiproxy.Proxy
}

func (c *DdbComponent) CfgOptions() []cfg.Option {
	clientEndpointKey := fmt.Sprintf("cloud.aws.dynamodb.clients.%s.endpoint", c.name)

	return []cfg.Option{
		cfg.WithConfigMap(map[string]interface{}{
			"cloud.aws.credentials": map[string]interface{}{
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
	return dynamodb.NewFromConfig(aws.Config{
		EndpointResolver: gosoAws.EndpointResolver(c.Endpoint()),
		Region:           "eu-central-1",
		Credentials:      GetDefaultStaticCredentials(),
	})
}

func (c *DdbComponent) Repository(settings *ddb.Settings) (ddb.Repository, error) {
	tracer := tracing.NewNoopTracer()
	client := c.Client()

	return ddb.NewWithInterfaces(c.logger, tracer, client, settings)
}

func (c *DdbComponent) Toxiproxy() *toxiproxy.Proxy {
	return c.toxiproxy
}
