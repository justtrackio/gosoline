package ddb

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"sync"
)

//go:generate mockery -name Client
type Client interface {
	dynamodbiface.DynamoDBAPI
}

var c = struct {
	sync.Mutex
	instance map[string]dynamodbiface.DynamoDBAPI
}{}

func ProvideClient(config cfg.Config, logger log.Logger, settings *Settings) dynamodbiface.DynamoDBAPI {
	c.Lock()
	defer c.Unlock()

	if c.instance == nil {
		c.instance = map[string]dynamodbiface.DynamoDBAPI{}
	}

	endpoint := config.GetString("aws_dynamoDb_endpoint")
	if c.instance[endpoint] != nil {
		return c.instance[endpoint]
	}

	c.instance[endpoint] = NewClient(config, logger, settings)

	return c.instance[endpoint]
}

func NewClient(config cfg.Config, logger log.Logger, settings *Settings) *dynamodb.DynamoDB {
	if settings.Backoff.Enabled {
		settings.Client.MaxRetries = 0
	}

	awsConfig := cloud.GetAwsConfig(config, logger, "dynamoDb", &settings.Client)
	sess := session.Must(session.NewSession(awsConfig))

	return dynamodb.New(sess)
}
