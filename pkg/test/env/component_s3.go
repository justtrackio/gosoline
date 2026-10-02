package env

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

type s3Component struct {
	baseComponent
	binding ContainerBinding
	region  string
	buckets map[string]string
}

func (c *s3Component) Address() string {
	return fmt.Sprintf("http://%s", c.binding.getAddress())
}

func (c *s3Component) CfgOptions() []cfg.Option {
	options := []cfg.Option{
		cfg.WithConfigSetting("cloud.aws.s3.clients.default", map[string]any{
			"credentials": map[string]any{
				"access_key_id":     DefaultAccessKeyID,
				"secret_access_key": DefaultSecretAccessKey,
				"session_token":     DefaultToken,
			},
			"endpoint":     c.Address(),
			"region":       c.region,
			"usePathStyle": true,
		}),
	}
	for name, bucket := range c.buckets {
		options = append(options, cfg.WithConfigSetting("blob."+name+".bucket", bucket))
	}

	return options
}

func (c *s3Component) S3Client() *s3.Client {
	return s3.NewFromConfig(aws.Config{
		Region: c.region, Credentials: GetDefaultStaticCredentials(),
	}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(c.Address())
		options.UsePathStyle = true
	})
}
