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

type localstackComponent struct {
	baseComponent
	config            cfg.Config
	logger            log.Logger
	endpointAddress   string
	region            string
	ddbNamingSettings *ddb.TableNamingSettings
	toxiproxy         *toxiproxy.Proxy
}

func (c *localstackComponent) CfgOptions() []cfg.Option {
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
				"dynamodb": map[string]any{
					"clients": map[string]any{
						"default": map[string]any{
							"purge_type": "drop_table",
						},
					},
				},
			},
		}),
	}

	return options
}

func (c *localstackComponent) Address() string {
	return fmt.Sprintf("http://%s", c.endpointAddress)
}

func (c *localstackComponent) DdbClient() *dynamodb.Client {
	return dynamodb.NewFromConfig(
		aws.Config{
			Region:      "eu-central-1",
			Credentials: GetDefaultStaticCredentials(),
		},
		func(options *dynamodb.Options) {
			options.BaseEndpoint = gosoAws.NilIfEmpty(c.Address())
		},
	)
}

func (c *localstackComponent) DdbRepository(settings *ddb.Settings) (ddb.Repository, error) {
	tracer := tracing.NewLocalTracer()
	client := c.DdbClient()

	if err := settings.ModelId.PadFromConfig(c.config); err != nil {
		return nil, fmt.Errorf("failed to pad model id from config: %w", err)
	}

	tableName := ddb.GetTableNameWithSettings(settings, c.ddbNamingSettings)
	metadataFactory := ddb.NewMetadataFactoryWithInterfaces(settings, tableName)

	return ddb.NewWithInterfaces(c.logger, tracer, client, metadataFactory)
}

func (c *localstackComponent) S3Client() *s3.Client {
	awsCfg := aws.Config{
		Region:      "eu-central-1",
		Credentials: GetDefaultStaticCredentials(),
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = gosoAws.NilIfEmpty(c.Address())
	})
}

func (c *localstackComponent) SnsClient() *sns.Client {
	return sns.NewFromConfig(
		aws.Config{
			Region:      "eu-central-1",
			Credentials: GetDefaultStaticCredentials(),
		},
		func(options *sns.Options) {
			options.BaseEndpoint = gosoAws.NilIfEmpty(c.Address())
		},
	)
}

func (c *localstackComponent) Toxiproxy() *toxiproxy.Proxy {
	return c.toxiproxy
}
