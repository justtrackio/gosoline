package stream

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoKinesis "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const kinesisBatchSizeMax = 500

type KinesisOutputSettings struct {
	StreamName string
}

type kinesisOutput struct {
	logger   log.Logger
	clock    clock.Clock
	uuidGen  uuid.Uuid
	client   gosoKinesis.Client
	settings *KinesisOutputSettings
}

func NewKinesisOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *KinesisOutputSettings) (Output, error) {
	client, err := gosoKinesis.ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("failed to provide kinesis client: %w", err)
	}

	err = gosoKinesis.CreateKinesisStream(ctx, config, logger, client, settings.StreamName)
	if err != nil {
		return nil, fmt.Errorf("failed to create kinesis stream: %w", err)
	}

	return NewKinesisOutputWithInterfaces(logger, clock.Provider, uuid.New(), client, settings), nil
}

func NewKinesisOutputWithInterfaces(logger log.Logger, clock clock.Clock, uuidGen uuid.Uuid, client gosoKinesis.Client, settings *KinesisOutputSettings) Output {
	return &kinesisOutput{
		logger:   logger,
		clock:    clock,
		uuidGen:  uuidGen,
		client:   client,
		settings: settings,
	}
}

func (o *kinesisOutput) WriteOne(ctx context.Context, record WritableMessage) error {
	return o.Write(ctx, []WritableMessage{record})
}

func (o *kinesisOutput) Write(ctx context.Context, batch []WritableMessage) error {
	if len(batch) == 0 {
		return nil
	}

	ctx = log.AppendLoggerContextField(ctx, log.Fields{
		"stream_name":              o.settings.StreamName,
		"kinesis_write_request_id": o.uuidGen.NewV4(),
	})

	var err, errs error
	var chunks Chunks

	if chunks, err = BuildChunks(batch, kinesisBatchSizeMax); err != nil {
		return fmt.Errorf("could not build batch for messages: %w", err)
	}

	for _, chunk := range chunks {
		if err = o.writeBatch(ctx, chunk); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return fmt.Errorf("can not put messages to stream %s: %w", o.settings.StreamName, errs)
	}

	return nil
}

func (o *kinesisOutput) writeBatch(ctx context.Context, batch [][]byte) error {
	records := make([]types.PutRecordsRequestEntry, 0, len(batch))

	for _, data := range batch {
		req := types.PutRecordsRequestEntry{
			Data:         data,
			PartitionKey: aws.String(o.uuidGen.NewV4()),
		}

		records = append(records, req)
	}

	tries := 0
	start := o.clock.Now()
	for len(records) > 0 {
		tries++
		failedRecords, err := o.putRecordsAndCollectFailed(ctx, records)
		if err != nil {
			return fmt.Errorf("can not write batch to stream %s: %w", o.settings.StreamName, err)
		}

		if len(failedRecords) > 0 {
			o.logger.WithContext(ctx).Warn("%d / %d records failed, retrying", len(failedRecords), len(records))
		} else if tries > 1 {
			took := o.clock.Now().Sub(start)
			o.logger.WithContext(ctx).Info("writeBatch successful after %d retries in %s", tries-1, took)
		}

		records = failedRecords
	}

	return nil
}

func (o *kinesisOutput) putRecordsAndCollectFailed(ctx context.Context, records []types.PutRecordsRequestEntry) ([]types.PutRecordsRequestEntry, error) {
	putRecordsOutput, err := o.client.PutRecords(ctx, &kinesis.PutRecordsInput{
		Records:    records,
		StreamName: aws.String(o.settings.StreamName),
	})
	if err != nil {
		return nil, fmt.Errorf("can execute PutRecordsRequest: %w", err)
	}

	failedRecords := make([]types.PutRecordsRequestEntry, 0, len(records))

	for i, outputRecord := range putRecordsOutput.Records {
		if outputRecord.ErrorCode != nil {
			failedRecords = append(failedRecords, records[i])
		}
	}

	return failedRecords, nil
}
