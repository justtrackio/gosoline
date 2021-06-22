package stream

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	gosoKinesis "github.com/applike/gosoline/pkg/cloud/aws/kinesis"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/hashicorp/go-multierror"
)

const kinesisBatchSizeMax = 500

type KinesisOutputSettings struct {
	StreamName string
	Backoff    exec.BackoffSettings
}

func (k *KinesisOutputSettings) GetResourceName() string {
	return k.StreamName
}

type kinesisOutput struct {
	logger      log.Logger
	uuidGen     uuid.Uuid
	client      kinesisiface.KinesisAPI
	batchExec   exec.Executor
	requestExec gosoAws.Executor
	settings    *KinesisOutputSettings
}

func NewKinesisOutput(config cfg.Config, logger log.Logger, settings *KinesisOutputSettings) (Output, error) {
	client := cloud.GetKinesisClient(config, logger)
	err := gosoKinesis.CreateKinesisStream(config, logger, client, settings)

	if err != nil {
		return nil, fmt.Errorf("failed to create kinesis stream: %w", err)
	}

	res := &exec.ExecutableResource{
		Type: "kinesis.request",
		Name: settings.StreamName,
	}
	requestExec := gosoAws.NewExecutor(logger, res, &settings.Backoff)

	return NewKinesisOutputWithInterfaces(logger, client, requestExec, settings), nil
}

func NewKinesisOutputWithInterfaces(logger log.Logger, client kinesisiface.KinesisAPI, requestExec gosoAws.Executor, settings *KinesisOutputSettings) Output {
	uuidGen := uuid.New()

	res := &exec.ExecutableResource{
		Type: "kinesis.batch",
		Name: settings.StreamName,
	}
	batchExec := exec.NewBackoffExecutor(logger, res, &settings.Backoff, CheckRecordsFailedError)

	return &kinesisOutput{
		logger:      logger,
		uuidGen:     uuidGen,
		client:      client,
		batchExec:   batchExec,
		requestExec: requestExec,
		settings:    settings,
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
	var err error
	var records = make([]*kinesis.PutRecordsRequestEntry, 0, len(batch))

	for _, data := range batch {
		req := &kinesis.PutRecordsRequestEntry{
			Data:         data,
			PartitionKey: aws.String(o.uuidGen.NewV4()),
		}

		records = append(records, req)
	}

	_, err = o.batchExec.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		records, err = o.putRecordsAndCollectFailed(ctx, records)
		return records, err
	})

	if err != nil {
		return fmt.Errorf("can not write batch to stream %s: %w", o.settings.StreamName, err)
	}

	return nil
}

func (o *kinesisOutput) putRecordsAndCollectFailed(ctx context.Context, records []*kinesis.PutRecordsRequestEntry) ([]*kinesis.PutRecordsRequestEntry, error) {
	input := kinesis.PutRecordsInput{
		Records:    records,
		StreamName: aws.String(o.settings.StreamName),
	}

	output, err := o.requestExec.Execute(ctx, func() (*request.Request, interface{}) {
		return o.client.PutRecordsRequest(&input)
	})

	if err != nil {
		return records, fmt.Errorf("can execute PutRecordsRequest: %w", err)
	}

	putRecordsOutput := output.(*kinesis.PutRecordsOutput)
	failedRecords := make([]*kinesis.PutRecordsRequestEntry, 0, len(records))

	for i, outputRecord := range putRecordsOutput.Records {
		if outputRecord.ErrorCode != nil {
			failedRecords = append(failedRecords, records[i])
		}
	}

	if len(failedRecords) > 0 {
		return failedRecords, NewRecordsFailedError(records, failedRecords)
	}

	return nil, nil
}

type RecordsFailedError struct {
	total  []*kinesis.PutRecordsRequestEntry
	failed []*kinesis.PutRecordsRequestEntry
}

func NewRecordsFailedError(total []*kinesis.PutRecordsRequestEntry, failed []*kinesis.PutRecordsRequestEntry) RecordsFailedError {
	return RecordsFailedError{
		total:  total,
		failed: failed,
	}
}

func (r RecordsFailedError) Error() string {
	return fmt.Sprintf("%d out of %d records failed", len(r.failed), len(r.total))
}

func IsRecordsFailedError(err error) bool {
	return errors.As(err, &RecordsFailedError{})
}

func CheckRecordsFailedError(_ interface{}, err error) exec.ErrorType {
	if IsRecordsFailedError(err) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}
