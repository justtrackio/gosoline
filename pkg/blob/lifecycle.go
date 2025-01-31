package blob

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

type lifecycleManager struct {
	service  *Service
	settings *Settings
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Purger
}

var _ LifecycleManager = (*lifecycleManager)(nil)

func NewLifecycleManager(settings *Settings) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var service *Service

		if service, err = NewService(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("can not create blob service: %w", err)
		}

		return &lifecycleManager{
			service:  service,
			settings: settings,
		}, nil
	}
}

func (l *lifecycleManager) GetId() string {
	return fmt.Sprintf("blob/%s", l.settings.Bucket)
}

func (l *lifecycleManager) Create(ctx context.Context) error {
	return l.service.CreateBucket(ctx)
}

func (l *lifecycleManager) Purge(ctx context.Context) error {
	return l.service.Purge(ctx)
}
