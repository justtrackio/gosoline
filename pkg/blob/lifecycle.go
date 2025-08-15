package blob

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const MetadataKey = "blob.stores"

type Metadata struct {
	AwsClientName string `json:"aws_client_name"`
	Bucket        string `json:"bucket"`
	Name          string `json:"name"`
	Prefix        string `json:"prefix"`
}

type lifecycleManager struct {
	service  *Service
	settings *Settings
	name     string
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Registerer
	reslife.Purger
}

var _ LifecycleManager = (*lifecycleManager)(nil)

func NewLifecycleManager(settings *Settings, name string) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var service *Service

		if service, err = NewService(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("can not create blob service: %w", err)
		}

		return &lifecycleManager{
			service:  service,
			settings: settings,
			name:     name,
		}, nil
	}
}

func (l *lifecycleManager) GetId() string {
	return fmt.Sprintf("blob/%s", l.name)
}

func (l *lifecycleManager) Create(ctx context.Context) error {
	return l.service.CreateBucket(ctx)
}

func (l *lifecycleManager) Register(ctx context.Context) (key string, metadata any, err error) {
	metadata = Metadata{
		AwsClientName: l.settings.ClientName,
		Bucket:        l.settings.Bucket,
		Name:          l.name,
		Prefix:        l.settings.Prefix,
	}

	return MetadataKey, metadata, nil
}

func (l *lifecycleManager) Purge(ctx context.Context) error {
	return l.service.Purge(ctx)
}
