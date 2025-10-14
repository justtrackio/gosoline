package kinesis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type StreamDescription struct {
	FullStreamName string
	StreamArn      string
	StreamName     string
	OpenShardCount int
}

type Service struct {
	logger         log.Logger
	client         Client
	fullStreamName string
}

func NewService(ctx context.Context, config cfg.Config, logger log.Logger, settings StreamNameSettingsAware) (*Service, error) {
	var err error
	var client Client
	var fullStreamName Stream

	if client, err = ProvideClient(ctx, config, logger, settings.GetClientName()); err != nil {
		return nil, fmt.Errorf("can not create kinesis client with name %s: %w", settings.GetClientName(), err)
	}

	if fullStreamName, err = GetStreamName(config, settings); err != nil {
		return nil, fmt.Errorf("can not get full stream name: %w", err)
	}

	return &Service{
		logger:         logger,
		client:         client,
		fullStreamName: string(fullStreamName),
	}, nil
}

func (s *Service) Create(ctx context.Context) error {
	s.logger.Info(ctx, "looking for kinesis stream: %s", s.fullStreamName)

	_, err := s.client.DescribeStreamSummary(ctx, &kinesis.DescribeStreamSummaryInput{
		StreamName: aws.String(s.fullStreamName),
	})

	if err == nil {
		s.logger.Info(ctx, "found kinesis stream: %s", s.fullStreamName)

		return nil
	}

	var errResourceNotFoundException *types.ResourceNotFoundException
	if !errors.As(err, &errResourceNotFoundException) && err != nil {
		return fmt.Errorf("failed to describe kinesis streams: %w", err)
	}

	s.logger.Info(ctx, "trying to create kinesis stream: %s", s.fullStreamName)

	_, err = s.client.CreateStream(ctx, &kinesis.CreateStreamInput{
		ShardCount: aws.Int32(1),
		StreamName: aws.String(s.fullStreamName),
	})

	var errResourceInUseException *types.ResourceInUseException
	if err != nil && errors.As(err, &errResourceInUseException) && strings.Contains(err.Error(), "already exists") {
		s.logger.Info(ctx, "kinesis stream already being created: %s", s.fullStreamName)

		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to create kinesis stream %s: %w", s.fullStreamName, err)
	}

	s.logger.Info(ctx, "created kinesis stream: %s", s.fullStreamName)

	return nil
}

func (s *Service) DescribeStream(ctx context.Context) (*StreamDescription, error) {
	var err error
	var out *kinesis.DescribeStreamSummaryOutput

	input := &kinesis.DescribeStreamSummaryInput{
		StreamName: aws.String(s.fullStreamName),
	}

	if out, err = s.client.DescribeStreamSummary(ctx, input); err != nil {
		return nil, fmt.Errorf("failed to describe kinesis stream %s: %w", s.fullStreamName, err)
	}

	return &StreamDescription{
		FullStreamName: s.fullStreamName,
		StreamArn:      *out.StreamDescriptionSummary.StreamARN,
		StreamName:     *out.StreamDescriptionSummary.StreamName,
		OpenShardCount: int(*out.StreamDescriptionSummary.OpenShardCount),
	}, err
}
