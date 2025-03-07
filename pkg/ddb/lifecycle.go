package ddb

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoDynamodb "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const MetadataKeyTables = "cloud.aws.dynamodb.tables"

type TableMetadata struct {
	AwsClientName string `json:"aws_client_name"`
	TableName     string `json:"table_name"`
}

type lifecycleManager struct {
	service         *Service
	metadataFactory *MetadataFactory
	settings        *Settings
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Registerer
	reslife.Purger
}

var _ LifecycleManager = &lifecycleManager{}

func NewLifecycleManager(settings *Settings, optFns ...gosoDynamodb.ClientOption) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var svc *Service

		if svc, err = NewService(ctx, config, logger, settings, optFns...); err != nil {
			return nil, fmt.Errorf("could not create ddb service: %w", err)
		}

		return &lifecycleManager{
			service:         svc,
			metadataFactory: NewMetadataFactory(config, settings),
			settings:        settings,
		}, nil
	}
}

func (l *lifecycleManager) GetId() string {
	return fmt.Sprintf("ddb/%s", l.settings.ModelId.String())
}

func (l *lifecycleManager) Create(ctx context.Context) error {
	if _, err := l.service.CreateTable(ctx); err != nil {
		return fmt.Errorf("could not create ddb table %s: %w", l.service.metadataFactory.GetTableName(), err)
	}

	return nil
}

func (l *lifecycleManager) Register(ctx context.Context) (key string, metadata any, err error) {
	metadata = TableMetadata{
		AwsClientName: l.settings.ClientName,
		TableName:     l.metadataFactory.GetTableName(),
	}

	return MetadataKeyTables, metadata, nil
}

func (l *lifecycleManager) Purge(ctx context.Context) error {
	return l.service.PurgeTable(ctx)
}
