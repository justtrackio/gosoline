package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

type snsComponent struct {
	baseComponent
	binding containerBinding
}

func (c *snsComponent) AppOptions() []application.Option {
	return []application.Option{
		application.WithConfigMap(map[string]interface{}{
			"aws_sns_endpoint":      c.Address(),
			"aws_sns_autoSubscribe": true,
		}),
	}
}

func (c *snsComponent) Address() string {
	return fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port)
}

func (c *snsComponent) Client() *sns.SNS {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:   aws.String(c.Address()),
		MaxRetries: aws.Int(0),
	}))

	return sns.New(sess)
}
