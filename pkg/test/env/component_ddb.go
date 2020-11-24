package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	awsExec "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type ddbComponent struct {
	baseComponent
	logger  mon.Logger
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
		Region:     aws.String(endpoints.EuCentral1RegionID),
	}))

	return dynamodb.New(sess)
}

func (c *ddbComponent) Repository(settings *ddb.Settings) ddb.Repository {
	return ddb.NewWithInterfaces(c.logger, tracing.NewNoopTracer(), c.Client(), awsExec.DefaultExecutor{}, settings)
}
