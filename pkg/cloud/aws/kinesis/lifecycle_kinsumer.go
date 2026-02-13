package kinesis

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const (
	MetadataKeyKinsumers = "cloud.aws.kinesis.kinsumers"
)

type KinsumerMetadata struct {
	AwsClientName  string       `json:"aws_client_name"`
	ClientId       ClientId     `json:"client_id"`
	Name           string       `json:"name"`
	OpenShardCount int          `json:"open_shard_count"`
	StreamAppId    cfg.Identity `json:"stream_app_id"`
	StreamArn      string       `json:"stream_arn"`
	StreamName     string       `json:"stream_name"`
	StreamNameFull Stream       `json:"stream_name_full"`
}

type lifecycleManagerKinsumer struct {
	service  *Service
	settings *Settings
	clientId ClientId
}

type LifecycleManagerKinsumer interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Registerer
}

var _ LifecycleManagerKinsumer = &lifecycleManagerKinsumer{}

func NewLifecycleManagerKinsumer(settings *Settings, clientId ClientId) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var service *Service

		if service, err = NewService(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("failed to create kinesis service: %w", err)
		}

		return &lifecycleManagerKinsumer{
			service:  service,
			settings: settings,
			clientId: clientId,
		}, nil
	}
}

func (l *lifecycleManagerKinsumer) GetId() string {
	return fmt.Sprintf("kinesis/%s/kinsumer", l.settings.StreamName)
}

func (l *lifecycleManagerKinsumer) Create(ctx context.Context) error {
	return l.service.Create(ctx)
}

func (l *lifecycleManagerKinsumer) Register(ctx context.Context) (key string, metadata any, err error) {
	var desc *StreamDescription

	if desc, err = l.service.DescribeStream(ctx); err != nil {
		return "", nil, fmt.Errorf("failed to describe stream: %w", err)
	}

	metadata = KinsumerMetadata{
		AwsClientName:  l.settings.ClientName,
		ClientId:       l.clientId,
		Name:           l.settings.Name,
		OpenShardCount: desc.OpenShardCount,
		StreamAppId:    l.settings.Identity,
		StreamArn:      desc.StreamArn,
		StreamName:     desc.StreamName,
		StreamNameFull: Stream(desc.FullStreamName),
	}

	return MetadataKeyKinsumers, metadata, nil
}
