package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type dynamodbPurger struct {
	logger   mon.Logger
	settings *ddb.Settings
	client   dynamodbiface.DynamoDBAPI
}

func newDynamodbPurger(config cfg.Config, logger mon.Logger, settings *ddb.Settings) *dynamodbPurger {
	client := ddb.ProvideClient(config, logger, settings)

	return &dynamodbPurger{
		logger:   logger,
		settings: settings,
		client:   client,
	}
}

func (p *dynamodbPurger) purgeDynamodb() error {
	tableName := ddb.TableName(p.settings.ModelId)
	p.logger.Infof("purging table %s", tableName)
	_, err := p.client.DeleteTable(&dynamodb.DeleteTableInput{TableName: aws.String(tableName)})
	p.logger.Infof("purging table %s done", tableName)

	return err
}
