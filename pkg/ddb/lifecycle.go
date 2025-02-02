package ddb

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoDynamodb "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

type LifecycleManager struct {
	service *Service
}

func NewLifecycleManager(settings *Settings, optFns ...gosoDynamodb.ClientOption) func() (string, reslife.LifeCycleerFactory) {
	return func() (string, reslife.LifeCycleerFactory) {
		id := fmt.Sprintf("ddb/%s", settings.ModelId.String())

		return id, func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
			var err error
			var svc *Service

			if svc, err = NewService(ctx, config, logger, settings, optFns...); err != nil {
				return nil, fmt.Errorf("could not create ddb service: %w", err)
			}

			return &LifecycleManager{svc}, nil
		}
	}
}

func (l *LifecycleManager) Create(ctx context.Context) error {
	if _, err := l.service.CreateTable(ctx); err != nil {
		return fmt.Errorf("could not create ddb table %s: %w", l.service.metadataFactory.GetTableName(), err)
	}

	return nil
}

func (l *LifecycleManager) Register(ctx context.Context) (string, any, error) {
	return "", nil, nil
}

func (l *LifecycleManager) Purge(ctx context.Context) error {
	return l.service.PurgeTable(ctx)
}
