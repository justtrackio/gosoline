package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
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
		cfg.WithConfigSetting("cloud.aws.defaults", map[string]interface{}{
			"region":   c.region,
			"endpoint": c.Address(),
		}),
	}

	if funk.ContainsString(c.services, localstackServiceCloudWatch) {
		options = append(options, cfg.WithConfigSetting("cloud.aws.cloudwatch.clients.default", map[string]interface{}{
			"endpoint": c.Address(),
		}))
	}

	if funk.ContainsString(c.services, localstackServiceSns) {
		options = append(options, cfg.WithConfigMap(map[string]interface{}{
			"aws_sns_endpoint":      c.Address(),
			"aws_sns_autoSubscribe": true,
		}))
	}

	if funk.ContainsString(c.services, localstackServiceSqs) {
		options = append(options, cfg.WithConfigMap(map[string]interface{}{
			"aws_sqs_endpoint":   c.Address(),
			"aws_sqs_autoCreate": true,
		}))
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

func (c *localstackComponent) SnsClient() *sns.SNS {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(c.region),
		Endpoint:   aws.String(c.Address()),
		MaxRetries: aws.Int(0),
	}))

	return sns.New(sess)
}
