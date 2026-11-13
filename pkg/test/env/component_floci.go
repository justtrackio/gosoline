package env

import (
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type flociComponent struct {
	baseComponent
	config          cfg.Config
	logger          log.Logger
	endpointAddress string
	region          string
	toxiproxy       *toxiproxy.Proxy
	dynamodb        *dynamoDbComponent
	s3              *s3Component
}

func (c *flociComponent) CfgOptions() []cfg.Option {
	options := []cfg.Option{
		cfg.WithConfigMap(map[string]any{
			"cloud.aws": map[string]any{
				"credentials": map[string]any{
					"access_key_id":     DefaultAccessKeyID,
					"secret_access_key": DefaultSecretAccessKey,
					"session_token":     DefaultToken,
				},
				"defaults": map[string]any{
					"region":   c.region,
					"endpoint": c.Address(),
				},
			},
		}),
	}

	options = append(options, c.dynamodb.CfgOptions()...)
	options = append(options, c.s3.CfgOptions()...)

	return options
}

func (c *flociComponent) Address() string {
	return fmt.Sprintf("http://%s", c.endpointAddress)
}

func (c *flociComponent) DdbClient() *dynamodb.Client {
	return c.dynamodb.DdbClient()
}

func (c *flociComponent) DdbRepository(settings *ddb.Settings) (ddb.Repository, error) {
	tracer := tracing.NewLocalTracer()
	client := c.DdbClient()

	if err := settings.ModelId.PadFromConfig(c.config); err != nil {
		return nil, fmt.Errorf("failed to pad model id from config: %w", err)
	}

	metadataFactory, err := ddb.NewMetadataFactory(c.config, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create ddb metadata factory: %w", err)
	}

	return ddb.NewWithInterfaces(c.logger, tracer, client, metadataFactory)
}

func (c *flociComponent) S3Client() *s3.Client {
	return c.s3.S3Client()
}

func (c *flociComponent) SnsClient() *sns.Client {
	return sns.NewFromConfig(
		aws.Config{
			Region:      c.region,
			Credentials: GetDefaultStaticCredentials(),
		},
		func(options *sns.Options) {
			options.BaseEndpoint = gosoAws.NilIfEmpty(c.Address())
		},
	)
}

func (c *flociComponent) Toxiproxy() *toxiproxy.Proxy {
	return c.toxiproxy
}
