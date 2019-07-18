package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/twinj/uuid"
	"time"
)

const kinesisBatchSizeMax = 500

type KinesisOutputSettings struct {
	StreamName string
}

type kinesisOutput struct {
	logger   mon.Logger
	client   kinesisiface.KinesisAPI
	settings *KinesisOutputSettings
}

func NewKinesisOutput(config cfg.Config, logger mon.Logger, settings *KinesisOutputSettings) Output {
	client := cloud.GetKinesisClient(config, logger)

	return NewKinesisOutputWithInterfaces(logger, client, settings)
}

func NewKinesisOutputWithInterfaces(logger mon.Logger, client kinesisiface.KinesisAPI, settings *KinesisOutputSettings) Output {
	return &kinesisOutput{
		client:   client,
		settings: settings,
		logger:   logger,
	}
}

func (o *kinesisOutput) WriteOne(ctx context.Context, record *Message) error {
	return o.Write(ctx, []*Message{record})
}

func (o *kinesisOutput) Write(ctx context.Context, batch []*Message) error {
	if len(batch) == 0 {
		return nil
	}

	errs := make([]error, 0)
	chunks, err := BuildChunks(batch, kinesisBatchSizeMax)

	if err != nil {
		o.logger.Error(err, "could not batch all messages")
	}

	for _, chunk := range chunks {
		err := o.writeBatch(chunk)

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Wrap(errs[0], fmt.Sprintf("there were %v write errors to %v", len(errs), o.settings.StreamName))
}

func (o *kinesisOutput) writeBatch(batch [][]byte) error {
	records := make([]*kinesis.PutRecordsRequestEntry, 0, len(batch))

	for _, data := range batch {
		request := &kinesis.PutRecordsRequestEntry{
			Data:         data,
			PartitionKey: aws.String(uuid.NewV4().String()),
		}

		records = append(records, request)
	}

	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = 15 * time.Minute

	err := backoff.Retry(func() (err error) {
		records, err = o.putRecordsAndCollectFailed(records)

		return err
	}, backoffConfig)

	if err != nil {
		o.logger.Error(err, "Error putting records")
	}

	return err
}

func (o *kinesisOutput) putRecordsAndCollectFailed(records []*kinesis.PutRecordsRequestEntry) ([]*kinesis.PutRecordsRequestEntry, error) {
	input := kinesis.PutRecordsInput{
		Records:    records,
		StreamName: aws.String(o.settings.StreamName),
	}

	putRecordsOutput, err := o.client.PutRecords(&input)

	if err != nil {
		o.logger.Error(err, "Error putting records")

		return records, err
	}

	failedRecords := make([]*kinesis.PutRecordsRequestEntry, 0, len(records))

	for i, outputRecord := range putRecordsOutput.Records {
		if outputRecord.ErrorCode != nil {
			failedRecords = append(failedRecords, records[i])
		}
	}

	o.logger.WithFields(mon.Fields{
		"failed_records": len(failedRecords),
		"total_records":  len(records),
	}).Debug("put records to stream")

	if len(failedRecords) > 0 {
		return failedRecords, errors.New("some records failed")
	}

	return nil, nil
}
