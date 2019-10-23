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
	"strings"
	"sync"
)

var br = struct {
	sync.Mutex
	instance *BatchRunner
}{}

func ProvideBatchRunner() *BatchRunner {
	br.Lock()
	defer br.Unlock()

	if br.instance != nil {
		return br.instance
	}

	br.instance = &BatchRunner{}

	return br.instance
}

type BatchRunner struct {
	kernel.ForegroundModule

	logger mon.Logger
	metric mon.MetricWriter
	client s3iface.S3API
	read   chan *Object
	write  chan *Object
}

func (r *BatchRunner) Boot(config cfg.Config, logger mon.Logger) error {
	appId := cfg.GetAppIdFromConfig(config)
	defaultMetrics := getDefaultRunnerMetrics(appId)

	r.logger = logger
	r.client = ProvideS3Client(config)
	r.metric = mon.NewMetricDaemonWriter(defaultMetrics...)
	r.read = make(chan *Object, 100)
	r.write = make(chan *Object, 100)

	return nil
}

func (r *BatchRunner) Run(ctx context.Context) error {
	for i := 0; i < 100; i++ {
		go r.executeRead()
	}

	for i := 0; i < 100; i++ {
		go r.executeWrite()
	}

	<-ctx.Done()

	return nil
}

func (r *BatchRunner) executeRead() {
	for object := range r.read {
		var body io.ReadCloser
		var err error

		key := strings.Join([]string{*object.prefix, *object.Key}, "/")
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

		object.Body = StreamReader(body)
		object.Exists = exists
		object.Error = err
		object.wg.Done()
	}
}

func (r *BatchRunner) executeWrite() {
	for object := range r.write {
		key := strings.Join([]string{*object.prefix, *object.Key}, "/")
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

		object.wg.Done()
	}
}

func getDefaultRunnerMetrics(appId cfg.AppId) []*mon.MetricDatum {
	name := appId.String()

	return []*mon.MetricDatum{
		{
			MetricName: name,
			Dimensions: map[string]string{
				"Operation": "Read",
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			MetricName: name,
			Dimensions: map[string]string{
				"Operation": "Write",
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}
