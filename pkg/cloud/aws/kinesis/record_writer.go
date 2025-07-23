package kinesis

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	kinesisBatchSizeMax = 500

	metricNamePutRecords          = "PutRecords"
	metricNamePutRecordsFailure   = "PutRecordsFailure"
	metricNamePutRecordsBatchSize = "PutRecordsBatchSize"
)

type Record struct {
	Data            []byte
	PartitionKey    *string
	ExplicitHashKey *string
}

type RecordWriterSettings struct {
	cfg.AppId
	ClientName string
	StreamName string
	Backoff    exec.BackoffSettings
}

func (r RecordWriterSettings) GetAppId() cfg.AppId {
	return r.AppId
}

func (r RecordWriterSettings) GetClientName() string {
	return r.ClientName
}

func (r RecordWriterSettings) GetStreamName() string {
	return r.StreamName
}

//go:generate go run github.com/vektra/mockery/v2 --name RecordWriter
type RecordWriter interface {
	PutRecord(ctx context.Context, record *Record) error
	PutRecords(ctx context.Context, batch []*Record) error
}

type recordWriter struct {
	logger         log.Logger
	metricWriter   metric.Writer
	clock          clock.Clock
	uuidGen        uuid.Uuid
	client         Client
	settings       *RecordWriterSettings
	fullStreamName string
}

func NewRecordWriter(ctx context.Context, config cfg.Config, logger log.Logger, settings *RecordWriterSettings) (RecordWriter, error) {
	var err error
	if err = settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("can not pad settings from config: %w", err)
	}

	var fullStreamName Stream
	var client *kinesis.Client

	if fullStreamName, err = GetStreamName(config, settings); err != nil {
		return nil, fmt.Errorf("can not get full stream name: %w", err)
	}

	defaultMetrics := getRecordWriterDefaultMetrics(string(fullStreamName))
	metricWriter := metric.NewWriter(defaultMetrics...)

	if client, err = ProvideClient(ctx, config, logger, settings.ClientName); err != nil {
		return nil, fmt.Errorf("failed to provide kinesis client: %w", err)
	}

	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManagerWriter(settings)); err != nil {
		return nil, fmt.Errorf("failed to add kinesis lifecycle manager: %w", err)
	}

	return NewRecordWriterWithInterfaces(logger, metricWriter, clock.Provider, uuid.New(), client, settings, string(fullStreamName)), nil
}

func NewRecordWriterWithInterfaces(
	logger log.Logger,
	metricWriter metric.Writer,
	clock clock.Clock,
	uuidGen uuid.Uuid,
	client Client,
	settings *RecordWriterSettings,
	fullStreamName string,
) RecordWriter {
	return &recordWriter{
		logger:         logger,
		metricWriter:   metricWriter,
		clock:          clock,
		uuidGen:        uuidGen,
		client:         client,
		settings:       settings,
		fullStreamName: fullStreamName,
	}
}

func (w *recordWriter) PutRecord(ctx context.Context, record *Record) error {
	return w.PutRecords(ctx, []*Record{record})
}

func (w *recordWriter) PutRecords(ctx context.Context, records []*Record) error {
	if len(records) == 0 {
		return nil
	}

	ctx = log.AppendContextFields(ctx, log.Fields{
		"stream_name":              w.fullStreamName,
		"kinesis_write_request_id": w.uuidGen.NewV4(),
	})

	var err, errs error
	chunks := funk.Chunk(records, kinesisBatchSizeMax)

	for _, chunk := range chunks {
		if err = w.putRecordsBatch(ctx, chunk); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return fmt.Errorf("can not put records to stream %s: %w", w.fullStreamName, errs)
	}

	return nil
}

func (w *recordWriter) putRecordsBatch(ctx context.Context, batch []*Record) error {
	records := make([]types.PutRecordsRequestEntry, 0, len(batch))

	for _, rec := range batch {
		if rec.PartitionKey == nil && rec.ExplicitHashKey == nil {
			rec.PartitionKey = aws.String(w.uuidGen.NewV4())
		}

		req := types.PutRecordsRequestEntry{
			Data:            rec.Data,
			PartitionKey:    rec.PartitionKey,
			ExplicitHashKey: rec.ExplicitHashKey,
		}

		records = append(records, req)
	}

	var err error
	var failedRecords []types.PutRecordsRequestEntry
	var reason string

	attempt := 1
	start := w.clock.Now()
	batchId := w.uuidGen.NewV4()

	backoff := exec.NewExponentialBackOff(&w.settings.Backoff)

	for {
		if failedRecords, reason, err = w.putRecordsAndCollectFailed(ctx, records); err != nil {
			return fmt.Errorf("can not write batch to stream: %w", err)
		}

		w.writeMetrics(ctx, len(records), len(failedRecords))
		took := w.clock.Now().Sub(start)

		if len(failedRecords) == 0 && attempt == 1 {
			break
		}

		logger := w.logger.WithFields(log.Fields{
			"batch_id": batchId,
		})

		if len(failedRecords) == 0 && attempt > 1 {
			logger.Warn(ctx, "PutRecords successful after %d attempts in %s", attempt, took)

			break
		}

		logger.Warn(
			ctx,
			"PutRecords failed %d of %d records with reason: %s: after %d attempts in %s",
			len(failedRecords),
			len(records),
			reason,
			attempt,
			took,
		)
		records = failedRecords

		// sleep some time before retrying to give the stream some time to recover from a ProvisionedThroughputExceededException
		sleep := backoff.NextBackOff()
		w.clock.Sleep(sleep)
		attempt++
	}

	return nil
}

func (w *recordWriter) putRecordsAndCollectFailed(
	ctx context.Context,
	records []types.PutRecordsRequestEntry,
) ([]types.PutRecordsRequestEntry, string, error) {
	putRecordsOutput, err := w.client.PutRecords(ctx, &kinesis.PutRecordsInput{
		Records:    records,
		StreamName: aws.String(w.fullStreamName),
	})
	if err != nil {
		return nil, "", fmt.Errorf("can not execute PutRecordsRequest: %w", err)
	}

	failedRecords := make([]types.PutRecordsRequestEntry, 0, len(records))
	errors := make(map[string]int)

	for i, outputRecord := range putRecordsOutput.Records {
		if outputRecord.ErrorCode == nil {
			continue
		}

		failedRecords = append(failedRecords, records[i])

		if _, ok := errors[*outputRecord.ErrorCode]; !ok {
			errors[*outputRecord.ErrorCode] = 0
		}

		errors[*outputRecord.ErrorCode]++
	}

	if len(failedRecords) == 0 {
		return failedRecords, "", nil
	}

	reasons := make([]string, 0)
	for errCode, count := range errors {
		reasons = append(reasons, fmt.Sprintf("%d %s errors", count, errCode))
	}
	reason := strings.Join(reasons, ", ")

	return failedRecords, reason, nil
}

func (w *recordWriter) writeMetrics(ctx context.Context, records int, failed int) {
	w.metricWriter.Write(ctx, metric.Data{
		&metric.Datum{
			MetricName: metricNamePutRecords,
			Dimensions: map[string]string{
				"StreamName": w.fullStreamName,
			},
			Value: float64(records - failed),
		},
		&metric.Datum{
			MetricName: metricNamePutRecordsFailure,
			Dimensions: map[string]string{
				"StreamName": w.fullStreamName,
			},
			Value: float64(failed),
		},
		&metric.Datum{
			MetricName: metricNamePutRecordsBatchSize,
			Dimensions: map[string]string{
				"StreamName": w.fullStreamName,
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
