package env

import (
	"fmt"
	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/applike/gosoline/pkg/cfg"
	awsExec "github.com/applike/gosoline/pkg/cloud/aws"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type DdbComponent struct {
	baseComponent
	logger     log.Logger
	ddbAddress string
	toxiproxy  *toxiproxy.Proxy
}

func (c *DdbComponent) CfgOptions() []cfg.Option {
	clientEndpointKey := fmt.Sprintf("cloud.aws.dynamodb.clients.%s.endpoint", c.name)

	return []cfg.Option{
		cfg.WithConfigMap(map[string]interface{}{
			"aws_dynamoDb_endpoint":   c.Endpoint(),
			"aws_dynamoDb_autoCreate": true,
		}),
		cfg.WithConfigSetting(clientEndpointKey, c.Endpoint()),
	}
}

func (c *DdbComponent) Address() string {
	return c.ddbAddress
}

func (c *DdbComponent) Endpoint() string {
	return fmt.Sprintf("http://%s", c.ddbAddress)
}

func (c *DdbComponent) Client() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:    aws.String(c.Endpoint()),
		MaxRetries:  aws.Int(0),
		Region:      aws.String(endpoints.EuCentral1RegionID),
		Credentials: gosoAws.GetDefaultCredentials(),
	}))

	return dynamodb.New(sess)
}

func (c *DdbComponent) Repository(settings *ddb.Settings) (ddb.Repository, error) {
	return ddb.NewWithInterfaces(c.logger, tracing.NewNoopTracer(), c.Client(), awsExec.DefaultExecutor{}, settings)
}

func (c *DdbComponent) Toxiproxy() *toxiproxy.Proxy {
	return c.toxiproxy
}
