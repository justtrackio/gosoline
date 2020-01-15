package ddb

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
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
	instance *dynamodb.DynamoDB
}{}

func ProvideClient(config cfg.Config, logger mon.Logger, settings *Settings) *dynamodb.DynamoDB {
	c.Lock()
	defer c.Unlock()

	if c.instance != nil {
		return c.instance
	}

	c.instance = NewClient(config, logger, settings)

	return c.instance
}

func NewClient(config cfg.Config, logger mon.Logger, settings *Settings) *dynamodb.DynamoDB {
	if settings.Backoff.Enabled {
		settings.Client.MaxRetries = 0
	}

	awsConfig := cloud.GetAwsConfig(config, logger, "dynamoDb", &settings.Client)
	sess := session.Must(session.NewSession(awsConfig))

	return dynamodb.New(sess)
}
