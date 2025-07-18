package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricName      = "BlobBatchRunner"
	operationCopy   = "Copy"
	operationDelete = "Delete"
	operationRead   = "Read"
	operationWrite  = "Write"
)

type BatchRunnerSettings struct {
	ClientName        string `cfg:"client_name" default:"default"`
	CopyRunnerCount   int    `cfg:"copy_runner_count" default:"10"`
	DeleteRunnerCount int    `cfg:"delete_runner_count" default:"10"`
	ReaderRunnerCount int    `cfg:"reader_runner_count" default:"10"`
	WriterRunnerCount int    `cfg:"writer_runner_count" default:"10"`
}

var br = struct {
	sync.Mutex
	instance BatchRunner
}{}

func ProvideBatchRunner(name string) kernel.ModuleFactory {
	br.Lock()
	defer br.Unlock()

	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		if br.instance != nil {
			return br.instance, nil
		}

		var err error
		br.instance, err = NewBatchRunner(ctx, config, logger, name)

		return br.instance, err
	}
}

//go:generate go run github.com/vektra/mockery/v2 --name BatchRunner
type BatchRunner interface {
	Run(ctx context.Context) error
}

type batchRunner struct {
	kernel.ForegroundModule
	kernel.ServiceStage

	logger   log.Logger
	channels *BatchRunnerChannels
	client   gosoS3.Client
	metric   metric.Writer
	settings *BatchRunnerSettings
}

func NewBatchRunner(ctx context.Context, config cfg.Config, logger log.Logger, name string) (BatchRunner, error) {
	settings := &BatchRunnerSettings{}
	if err := config.UnmarshalKey(fmt.Sprintf("blob.%s", name), settings); err != nil {
		return nil, err
	}

	defaultMetrics := getDefaultRunnerMetrics()
	metricWriter := metric.NewWriter(defaultMetrics...)

	s3Client, err := gosoS3.ProvideClient(ctx, config, logger, settings.ClientName)
	if err != nil {
		return nil, fmt.Errorf("can not create s3 client with name %s: %w", settings.ClientName, err)
	}

	runnerChannels, err := ProvideBatchRunnerChannels(config)
	if err != nil {
		return nil, fmt.Errorf("can not create batch runner channels: %w", err)
	}

	runner := &batchRunner{
		logger:   logger,
		metric:   metricWriter,
		client:   s3Client,
		channels: runnerChannels,
		settings: settings,
	}

	return runner, nil
}

func (r *batchRunner) Run(ctx context.Context) error {
	for i := 0; i < r.settings.ReaderRunnerCount; i++ {
		go r.executeRead(ctx)
	}

	for i := 0; i < r.settings.WriterRunnerCount; i++ {
		go r.executeWrite(ctx)
	}

	for i := 0; i < r.settings.CopyRunnerCount; i++ {
		go r.executeCopy(ctx)
	}

	for i := 0; i < r.settings.DeleteRunnerCount; i++ {
		go r.executeDelete(ctx)
	}

	<-ctx.Done()

	return nil
}

func (r *batchRunner) executeRead(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case object := <-r.channels.read:
			var body io.ReadCloser
			var err error

			key := object.GetFullKey()
			exists := true

			input := &s3.GetObjectInput{
				Bucket: object.bucket,
				Key:    aws.String(key),
			}

			out, err := r.client.GetObject(ctx, input)

			if err != nil {
				if awsErr, ok := err.(awserr.RequestFailure); ok && awsErr.StatusCode() == 404 {
					exists = false
					err = nil
				}
			} else {
				body = out.Body
			}

			r.writeMetric(operationRead)

			object.Body = StreamReader(body)
			object.Exists = exists
			object.Error = err
			object.wg.Done()
		}
	}
}

func (r *batchRunner) executeWrite(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case object := <-r.channels.write:
			key := object.GetFullKey()
			body := CloseOnce(object.Body.AsReader())

			input := &s3.PutObjectInput{
				ACL:             object.ACL,
				Body:            body,
				Bucket:          object.bucket,
				Key:             aws.String(key),
				ContentEncoding: object.ContentEncoding,
				ContentType:     object.ContentType,
			}

			_, err := r.client.PutObject(ctx, input)

			if err != nil {
				object.Exists = false
				object.Error = err
			} else {
				object.Exists = true
			}

			if err := body.Close(); err != nil {
				object.Error = errors.Join(object.Error, err)
			}

			r.writeMetric(operationWrite)

			object.wg.Done()
		}
	}
}

func (r *batchRunner) executeCopy(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case object := <-r.channels.copy:
			key := object.GetFullKey()
			source := object.getSource()

			input := &s3.CopyObjectInput{
				ACL:             object.ACL,
				Bucket:          object.bucket,
				Key:             aws.String(key),
				CopySource:      aws.String(source),
				ContentEncoding: object.ContentEncoding,
				ContentType:     object.ContentType,
			}

			_, err := r.client.CopyObject(ctx, input)
			if err != nil {
				object.Error = err
			}

			r.writeMetric(operationCopy)

			object.wg.Done()
		}
	}
}

func (r *batchRunner) executeDelete(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case object := <-r.channels.delete:
			key := object.GetFullKey()

			input := &s3.DeleteObjectInput{
				Bucket: object.bucket,
				Key:    aws.String(key),
			}

			_, err := r.client.DeleteObject(ctx, input)
			if err != nil {
				object.Error = err
			}

			r.writeMetric(operationDelete)

			object.wg.Done()
		}
	}
}

func (r *batchRunner) writeMetric(operation string) {
	r.metric.WriteOne(&metric.Datum{
		MetricName: metricName,
		Priority:   metric.PriorityHigh,
		Dimensions: map[string]string{
			"Operation": operation,
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})
}

func getDefaultRunnerMetrics() []*metric.Datum {
	return []*metric.Datum{
		{
			MetricName: metricName,
			Priority:   metric.PriorityHigh,
			Dimensions: map[string]string{
				"Operation": operationRead,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			MetricName: metricName,
			Priority:   metric.PriorityHigh,
			Dimensions: map[string]string{
				"Operation": operationWrite,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			MetricName: metricName,
			Priority:   metric.PriorityHigh,
			Dimensions: map[string]string{
				"Operation": operationCopy,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			MetricName: metricName,
			Priority:   metric.PriorityHigh,
			Dimensions: map[string]string{
				"Operation": operationDelete,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
