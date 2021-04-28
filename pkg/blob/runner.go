package blob

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/hashicorp/go-multierror"
	"io"
	"sync"
)

const (
	metricName      = "BlobBatchRunner"
	operationRead   = "Read"
	operationWrite  = "Write"
	operationCopy   = "Copy"
	operationDelete = "Delete"
)

type BatchRunnerSettings struct {
	ReaderRunnerCount int `cfg:"reader_runner_count" default:"10"`
	WriterRunnerCount int `cfg:"writer_runner_count" default:"10"`
	CopyRunnerCount   int `cfg:"copy_runner_count" default:"10"`
	DeleteRunnerCount int `cfg:"delete_runner_count" default:"10"`
}

var br = struct {
	sync.Mutex
	instance *batchRunner
}{}

func ProvideBatchRunner() kernel.ModuleFactory {
	br.Lock()
	defer br.Unlock()

	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
		if br.instance != nil {
			return br.instance, nil
		}

		br.instance = NewBatchRunner(config, logger)

		return br.instance, nil
	}
}

//go:generate mockery --name BatchRunner
type BatchRunner interface {
	Run(ctx context.Context) error
}

type batchRunner struct {
	kernel.ForegroundModule
	kernel.ServiceStage

	logger   mon.Logger
	metric   mon.MetricWriter
	client   s3iface.S3API
	channels *BatchRunnerChannels
	settings *BatchRunnerSettings
}

func NewBatchRunner(config cfg.Config, logger mon.Logger) *batchRunner {
	settings := &BatchRunnerSettings{}
	config.UnmarshalKey("blob", settings)

	defaultMetrics := getDefaultRunnerMetrics()
	metricWriter := mon.NewMetricDaemonWriter(defaultMetrics...)

	runner := &batchRunner{
		logger:   logger,
		metric:   metricWriter,
		client:   ProvideS3Client(config),
		channels: ProvideBatchRunnerChannels(config),
		settings: settings,
	}

	return runner
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

			out, err := r.client.GetObject(input)

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
				ACL:    object.ACL,
				Body:   body,
				Bucket: object.bucket,
				Key:    aws.String(key),
			}

			_, err := r.client.PutObject(input)

			if err != nil {
				object.Exists = false
				object.Error = err
			} else {
				object.Exists = true
			}

			if err := body.Close(); err != nil {
				object.Error = multierror.Append(object.Error, err)
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
				ACL:        object.ACL,
				Bucket:     object.bucket,
				Key:        aws.String(key),
				CopySource: aws.String(source),
			}

			_, err := r.client.CopyObject(input)

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

			_, err := r.client.DeleteObject(input)

			if err != nil {
				object.Error = err
			}

			r.writeMetric(operationDelete)

			object.wg.Done()
		}
	}
}

func (r *batchRunner) writeMetric(operation string) {
	r.metric.WriteOne(&mon.MetricDatum{
		MetricName: metricName,
		Priority:   mon.PriorityHigh,
		Dimensions: map[string]string{
			"Operation": operation,
		},
		Unit:  mon.UnitCount,
		Value: 1.0,
	})
}

func getDefaultRunnerMetrics() []*mon.MetricDatum {
	return []*mon.MetricDatum{
		{
			MetricName: metricName,
			Priority:   mon.PriorityHigh,
			Dimensions: map[string]string{
				"Operation": operationRead,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			MetricName: metricName,
			Priority:   mon.PriorityHigh,
			Dimensions: map[string]string{
				"Operation": operationWrite,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			MetricName: metricName,
			Priority:   mon.PriorityHigh,
			Dimensions: map[string]string{
				"Operation": operationCopy,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			MetricName: metricName,
			Priority:   mon.PriorityHigh,
			Dimensions: map[string]string{
				"Operation": operationDelete,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}
