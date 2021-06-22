package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type dynamodbPurger struct {
	logger   log.Logger
	settings *ddb.Settings
	client   dynamodbiface.DynamoDBAPI
}

func newDynamodbPurger(config cfg.Config, logger log.Logger, settings *ddb.Settings) *dynamodbPurger {
	client := ddb.ProvideClient(config, logger, settings)

	return &dynamodbPurger{
		logger:   logger,
		settings: settings,
		client:   client,
	}
}

func (p *dynamodbPurger) purgeDynamodb() error {
	tableName := ddb.TableName(p.settings)
	p.logger.Info("purging table %s", tableName)
	_, err := p.client.DeleteTable(&dynamodb.DeleteTableInput{TableName: aws.String(tableName)})
	p.logger.Info("purging table %s done", tableName)

	return err
}
