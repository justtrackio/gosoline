package kinesis

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const (
	metadataKeyRecordWriters = "cloud.aws.kinesis.record_writers"
)

type RecordWriterMetadata struct {
	AwsClientName  string `json:"aws_client_name"`
	OpenShardCount int    `json:"open_shard_count"`
	StreamArn      string `json:"stream_arn"`
	StreamName     string `json:"stream_name"`
}

type lifecycleManagerWriter struct {
	service  *Service
	settings StreamNameSettingsAware
}

type LifecycleManagerWriter interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Registerer
}

var _ LifecycleManagerWriter = &lifecycleManagerWriter{}

func NewLifecycleManagerWriter(settings StreamNameSettingsAware) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var service *Service

		if service, err = NewService(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("failed to create kinesis service: %w", err)
		}

		return &lifecycleManagerWriter{
			service:  service,
			settings: settings,
		}, nil
	}
}

func (l *lifecycleManagerWriter) GetId() string {
	return fmt.Sprintf("kinesis/%s/writer", l.settings.GetStreamName())
}

func (l *lifecycleManagerWriter) Create(ctx context.Context) error {
	return l.service.Create(ctx)
}

func (l *lifecycleManagerWriter) Register(ctx context.Context) (key string, metadata any, err error) {
	var desc *StreamDescription

	if desc, err = l.service.DescribeStream(ctx); err != nil {
		return "", nil, fmt.Errorf("failed to describe stream: %w", err)
	}

	metadata = RecordWriterMetadata{
		AwsClientName:  l.settings.GetClientName(),
		OpenShardCount: desc.OpenShardCount,
		StreamArn:      desc.StreamArn,
		StreamName:     desc.StreamName,
	}

	return metadataKeyRecordWriters, metadata, nil
}
