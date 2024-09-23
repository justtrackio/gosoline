package env

import (
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type localstackComponent struct {
	baseComponent
	services []string
	binding  containerBinding
	region   string
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

	if slices.Contains(c.services, localstackServiceS3) {
		options = append(options, cfg.WithConfigMap(map[string]any{
			"aws_s3_endpoint":   c.Address(),
			"aws_s3_autoCreate": true,
		}))
	}

	return options
}

func (c *localstackComponent) Address() string {
	return fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port)
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
