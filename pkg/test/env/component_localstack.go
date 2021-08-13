package env

import (
	"fmt"

	"github.com/applike/gosoline/pkg/cfg"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/thoas/go-funk"
)

type localstackComponent struct {
	baseComponent
	services []string
	binding  containerBinding
	region   string
}

func (c *localstackComponent) CfgOptions() []cfg.Option {
	options := []cfg.Option{
		cfg.WithConfigMap(map[string]interface{}{
			"cloud.aws": map[string]interface{}{
				"credentials:": map[string]interface{}{
					"access_key_id":     DefaultAccessKeyID,
					"secret_access_key": DefaultSecretAccessKey,
					"session_token":     DefaultToken,
				},
				"defaults": map[string]interface{}{
					"region":   c.region,
					"endpoint": c.Address(),
				},
			},
		}),
	}

	if funk.ContainsString(c.services, localstackServiceS3) {
		options = append(options, cfg.WithConfigMap(map[string]interface{}{
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
	return sns.NewFromConfig(aws.Config{
		EndpointResolver: gosoAws.EndpointResolver(c.Address()),
		Region:           "eu-central-1",
		Credentials:      GetDefaultStaticCredentials(),
	})
}
