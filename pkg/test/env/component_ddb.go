package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type ddbComponent struct {
	baseComponent
	binding containerBinding
}

func (c *ddbComponent) AppOptions() []application.Option {
	return []application.Option{
		application.WithConfigMap(map[string]interface{}{
			"aws_dynamoDb_endpoint":   fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port),
			"aws_dynamoDb_autoCreate": true,
		}),
	}
}

func (c *ddbComponent) Address() string {
	return fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port)
}

func (c *ddbComponent) Client() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:   aws.String(c.Address()),
		MaxRetries: aws.Int(0),
	}))

	return dynamodb.New(sess)
}
