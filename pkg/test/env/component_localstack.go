package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

type localstackComponent struct {
	baseComponent
	binding containerBinding
	region  string
}

func (c *localstackComponent) AppOptions() []application.Option {
	return []application.Option{
		application.WithConfigMap(map[string]interface{}{
			"aws_sns_endpoint":      c.Address(),
			"aws_sns_autoSubscribe": true,
			"aws_sqs_endpoint":      c.Address(),
			"aws_sqs_autoCreate":    true,
			"aws_s3_endpoint":       c.Address(),
			"aws_s3_autoCreate":     true,
		}),
	}
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
