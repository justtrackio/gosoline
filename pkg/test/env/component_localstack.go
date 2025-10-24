package env

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type localstackComponent struct {
	baseComponent
	binding ContainerBinding
	region  string
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
			},
		}),
	}

	return options
}

func (c *localstackComponent) Address() string {
	return fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port)
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
