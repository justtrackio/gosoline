package fixtures

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoDynamodb "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

type dynamodbPurger struct {
	logger    log.Logger
	settings  *ddb.Settings
	client    gosoDynamodb.Client
	tableName string
}

func NewDynamodbPurger(ctx context.Context, config cfg.Config, logger log.Logger, settings *ddb.Settings) (*dynamodbPurger, error) {
	var err error
	var client gosoDynamodb.Client

	if client, err = gosoDynamodb.ProvideClient(ctx, config, logger, "default"); err != nil {
		return nil, fmt.Errorf("can not create dynamodb client: %w", err)
	}

	tableName := ddb.TableName(config, settings)

	return &dynamodbPurger{
		logger:    logger,
		settings:  settings,
		client:    client,
		tableName: tableName,
	}, nil
}

func (p *dynamodbPurger) Purge(ctx context.Context) error {
	p.logger.Info("purging table %s", p.tableName)
	_, err := p.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(p.tableName)})
	p.logger.Info("purging table %s done", p.tableName)

	return err
}
