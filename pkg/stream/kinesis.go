package stream

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
)

type ResourceNameGetter interface {
	GetResourceName() string
}

func createKinesisStream(config cfg.Config, logger mon.Logger, client kinesisiface.KinesisAPI, settings ResourceNameGetter) {
	autoCreate := config.GetBool("aws_kinesis_autoCreate")
	if !autoCreate {
		return
	}

	streamName := settings.GetResourceName()
	logger.Infof("looking for kinesis stream: %s", streamName)

	streams, err := client.ListStreams(&kinesis.ListStreamsInput{})
	if err != nil {
		logger.WithFields(mon.Fields{
			"stream_name": streamName,
		}).Error(err, "failed to list kinesis streams")

		return
	}

	for _, awsStreamName := range streams.StreamNames {
		if *awsStreamName != streamName {
			continue
		}

		logger.Infof("found kinesis stream: %s", streamName)

		return
	}

	logger.Infof("trying to create kinesis stream: %s", streamName)
	_, err = client.CreateStream(&kinesis.CreateStreamInput{
		ShardCount: aws.Int64(1),
		StreamName: aws.String(streamName),
	})

	if err != nil {
		logger.WithFields(mon.Fields{
			"stream_name": streamName,
		}).Errorf(err, "failed to create kinesis stream: %s", streamName)

		return
	}

	logger.Infof("created kinesis stream: %s", streamName)
}
