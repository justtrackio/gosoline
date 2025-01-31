package ddb

import (
	"context"
	"fmt"
)

type LifecycleManager struct {
	service *Service
}

func NewLifecycleManager(service *Service) *LifecycleManager {
	return &LifecycleManager{service}
}

func (l *LifecycleManager) GetId() string {
	return fmt.Sprintf("ddb/%s", l.service.metadataFactory.GetTableName())
}

func (l *LifecycleManager) Create(ctx context.Context) error {
	if _, err := l.service.CreateTable(ctx); err != nil {
		return fmt.Errorf("could not create ddb table %s: %w", l.service.metadataFactory.GetTableName(), err)
	}

	return nil
}

func (l *LifecycleManager) Purge(ctx context.Context) error {
	return l.service.PurgeTable(ctx)
}
