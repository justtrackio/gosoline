package kinesis

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/dx"
	"github.com/justtrackio/gosoline/pkg/log"
)

// CreateKinesisStream ensures a kinesis stream exists if dx.auto_create is set to true.
func CreateKinesisStream(ctx context.Context, config cfg.Config, logger log.Logger, client Client, streamName string) error {
	logger.Info("looking for kinesis stream: %s", streamName)

	_, err := client.DescribeStreamSummary(ctx, &kinesis.DescribeStreamSummaryInput{
		StreamName: aws.String(streamName),
	})

	if err == nil {
		logger.Info("found kinesis stream: %s", streamName)
		return nil
	}

	var errResourceNotFoundException *types.ResourceNotFoundException
	if !errors.As(err, &errResourceNotFoundException) && err != nil {
		return fmt.Errorf("failed to describe kinesis streams: %w", err)
	}

	if !dx.ShouldAutoCreate(config) {
		return fmt.Errorf("kinesis stream does not exist and auto create is disabled")
	}

	logger.Info("trying to create kinesis stream: %s", streamName)
	_, err = client.CreateStream(ctx, &kinesis.CreateStreamInput{
		ShardCount: aws.Int32(1),
		StreamName: aws.String(streamName),
	})

	if err != nil {
		return fmt.Errorf("failed to create kinesis stream %s: %w", streamName, err)
	}

	logger.Info("created kinesis stream: %s", streamName)

	return nil
}
