package env

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

type S3Component struct {
	baseComponent
	s3Address string
}

func (c *S3Component) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigMap(map[string]interface{}{
			"cloud.aws.defaults.credentials": map[string]interface{}{
				"access_key_id":     DefaultAccessKeyID,
				"secret_access_key": DefaultSecretAccessKey,
			},
			"cloud.aws.s3.clients.default": map[string]interface{}{
				"endpoint":     c.s3Address,
				"usePathStyle": true,
			},
		}),
	}
}

func (c *S3Component) Client() *s3.Client {
	awsCfg := aws.Config{
		Region:      "eu-central-1",
		Credentials: GetDefaultStaticCredentials(),
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(c.s3Address)
	})
}
