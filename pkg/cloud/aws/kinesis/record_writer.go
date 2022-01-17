package kinesis

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/thoas/go-funk"
)

const (
	kinesisBatchSizeMax           = 500
	metadataKeyRecordWriters      = "cloud.aws.kinesis.record_writers"
	metricNamePutRecords          = "PutRecords"
	metricNamePutRecordsFailure   = "PutRecordsFailure"
	metricNamePutRecordsBatchSize = "PutRecordsBatchSize"
)

type RecordWriterMetadata struct {
	StreamName string `json:"stream_name"`
}

type RecordWriterSettings struct {
	StreamName string
}

type RecordWriter interface {
	PutRecord(ctx context.Context, record []byte) error
	PutRecords(ctx context.Context, batch [][]byte) error
}

type recordWriter struct {
	logger       log.Logger
	metricWriter metric.Writer
	clock        clock.Clock
	uuidGen      uuid.Uuid
	client       Client
	settings     *RecordWriterSettings
}

func NewRecordWriter(ctx context.Context, config cfg.Config, logger log.Logger, settings *RecordWriterSettings) (RecordWriter, error) {
	defaultMetrics := getRecordWriterDefaultMetrics(settings.StreamName)
	metricWriter := metric.NewDaemonWriter(defaultMetrics...)

	client, err := ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("failed to provide kinesis client: %w", err)
	}

	err = CreateKinesisStream(ctx, config, logger, client, settings.StreamName)
	if err != nil {
		return nil, fmt.Errorf("failed to create kinesis stream: %w", err)
	}

	metadata := RecordWriterMetadata{
		StreamName: settings.StreamName,
	}
	if err = appctx.MetadataAppend(ctx, metadataKeyRecordWriters, metadata); err != nil {
		return nil, fmt.Errorf("can not access the appctx metadata: %w", err)
	}

	return NewRecordWriterWithInterfaces(logger, metricWriter, clock.Provider, uuid.New(), client, settings), nil
}

func NewRecordWriterWithInterfaces(logger log.Logger, metricWriter metric.Writer, clock clock.Clock, uuidGen uuid.Uuid, client Client, settings *RecordWriterSettings) RecordWriter {
	return &recordWriter{
		logger:       logger,
		metricWriter: metricWriter,
		clock:        clock,
		uuidGen:      uuidGen,
		client:       client,
		settings:     settings,
	}
}

func (o *recordWriter) PutRecord(ctx context.Context, record []byte) error {
	return o.PutRecords(ctx, [][]byte{record})
}

func (o *recordWriter) PutRecords(ctx context.Context, records [][]byte) error {
	if len(records) == 0 {
		return nil
	}

	ctx = log.AppendLoggerContextField(ctx, log.Fields{
		"stream_name":              o.settings.StreamName,
		"kinesis_write_request_id": o.uuidGen.NewV4(),
	})

	var err, errs error
	chunks := funk.Chunk(records, kinesisBatchSizeMax).([][][]byte)

	for _, chunk := range chunks {
		if err = o.putRecordsBatch(ctx, chunk); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return fmt.Errorf("can not put records to stream %s: %w", o.settings.StreamName, errs)
	}

	return nil
}

func (o *recordWriter) putRecordsBatch(ctx context.Context, batch [][]byte) error {
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

		o.writeMetrics(len(records), len(failedRecords))

		if len(failedRecords) > 0 {
			o.logger.WithContext(ctx).Warn("%d / %d records failed, retrying", len(failedRecords), len(records))
		} else if tries > 1 {
			took := o.clock.Now().Sub(start)
			o.logger.WithContext(ctx).Warn("PutRecords successful after %d retries in %s", tries-1, took)
		}

		records = failedRecords
	}

	return nil
}

func (o *recordWriter) putRecordsAndCollectFailed(ctx context.Context, records []types.PutRecordsRequestEntry) ([]types.PutRecordsRequestEntry, error) {
	putRecordsOutput, err := o.client.PutRecords(ctx, &kinesis.PutRecordsInput{
		Records:    records,
		StreamName: aws.String(o.settings.StreamName),
	})
	if err != nil {
		return nil, fmt.Errorf("can not execute PutRecordsRequest: %w", err)
	}

	failedRecords := make([]types.PutRecordsRequestEntry, 0, len(records))

	for i, outputRecord := range putRecordsOutput.Records {
		if outputRecord.ErrorCode != nil {
			failedRecords = append(failedRecords, records[i])
		}
	}

	return failedRecords, nil
}

func (o *recordWriter) writeMetrics(records int, failed int) {
	o.metricWriter.Write(metric.Data{
		&metric.Datum{
			MetricName: metricNamePutRecords,
			Dimensions: map[string]string{
				"StreamName": o.settings.StreamName,
			},
			Value: float64(records),
		},
		&metric.Datum{
			MetricName: metricNamePutRecordsFailure,
			Dimensions: map[string]string{
				"StreamName": o.settings.StreamName,
			},
			Value: float64(failed),
		},
		&metric.Datum{
			MetricName: metricNamePutRecordsBatchSize,
			Dimensions: map[string]string{
				"StreamName": o.settings.StreamName,
			},
			Value: float64(records),
		},
	})
}

func getRecordWriterDefaultMetrics(streamName string) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNamePutRecords,
			Dimensions: map[string]string{
				"StreamName": streamName,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNamePutRecordsFailure,
			Dimensions: map[string]string{
				"StreamName": streamName,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNamePutRecordsBatchSize,
			Dimensions: map[string]string{
				"StreamName": streamName,
			},
			Unit:  metric.UnitCountAverage,
			Value: 0.0,
		},
	}
}
