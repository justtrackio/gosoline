package kinesis

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
)

//go:generate mockery -name ResourceNameGetter
type ResourceNameGetter interface {
	GetResourceName() string
}

// Ensure a kinesis stream exists if autoCreate for kinesis is set to true.
func CreateKinesisStream(config cfg.Config, logger log.Logger, client kinesisiface.KinesisAPI, settings ResourceNameGetter) error {
	autoCreate := config.GetBool("aws_kinesis_autoCreate")
	if !autoCreate {
		return nil
	}

	streamName := settings.GetResourceName()
	logger.Info("looking for kinesis stream: %s", streamName)

	streams, err := client.ListStreams(&kinesis.ListStreamsInput{})
	if err != nil {
		return fmt.Errorf("failed to list kinesis streams: %w", err)
	}

	for _, awsStreamName := range streams.StreamNames {
		if *awsStreamName != streamName {
			continue
		}

		logger.Info("found kinesis stream: %s", streamName)

		return nil
	}

	logger.Info("trying to create kinesis stream: %s", streamName)
	_, err = client.CreateStream(&kinesis.CreateStreamInput{
		ShardCount: aws.Int64(1),
		StreamName: aws.String(streamName),
	})

	if err != nil {
		return fmt.Errorf("failed to create kinesis stream %s: %w", streamName, err)
	}

	logger.Info("created kinesis stream: %s", streamName)

	return nil
}
