package fixtures

import (
	"context"
	"fmt"

	"github.com/applike/gosoline/pkg/cfg"
	gosoDynamodb "github.com/applike/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type dynamodbPurger struct {
	logger   log.Logger
	settings *ddb.Settings
	client   gosoDynamodb.Client
}

func newDynamodbPurger(ctx context.Context, config cfg.Config, logger log.Logger, settings *ddb.Settings) (*dynamodbPurger, error) {
	var err error
	var client gosoDynamodb.Client

	if client, err = gosoDynamodb.ProvideClient(ctx, config, logger, "default"); err != nil {
		return nil, fmt.Errorf("can not create dynamodb client: %w", err)
	}

	return &dynamodbPurger{
		logger:   logger,
		settings: settings,
		client:   client,
	}, nil
}

func (p *dynamodbPurger) purgeDynamodb(ctx context.Context) error {
	tableName := ddb.TableName(p.settings)
	p.logger.Info("purging table %s", tableName)
	_, err := p.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(tableName)})
	p.logger.Info("purging table %s done", tableName)

	return err
}
