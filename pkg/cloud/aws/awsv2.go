package aws

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/smithy-go/logging"
	"github.com/aws/smithy-go/middleware"
	"github.com/cenkalti/backoff"
	"time"
)

type ClientHttpSettings struct {
	Timeout time.Duration `cfg:"timeout" default:"0"`
}

type ClientRetrySettings struct {
	InitialInterval time.Duration `cfg:"initial_interval" default:"50ms"`
	MaxInterval     time.Duration `cfg:"max_interval" default:"10s"`
	MaxElapsedTime  time.Duration `cfg:"max_elapsed_time" default:"15m"`
}

type ClientSettings struct {
	Region     string              `cfg:"region" default:"eu-central-1"`
	Endpoint   string              `cfg:"endpoint" default:"http://localhost:4566"`
	Retry      ClientRetrySettings `cfg:"retry"`
	HttpClient ClientHttpSettings  `cfg:"http_client"`
}

func DefaultClientOptions(logger log.Logger, clock clock.Clock, settings ClientSettings, optFns ...func(options *awsCfg.LoadOptions) error) []func(options *awsCfg.LoadOptions) error {
	options := []func(options *awsCfg.LoadOptions) error{
		awsCfg.WithRegion(settings.Region),
		awsCfg.WithEndpointResolver(EndpointResolver(settings.Endpoint)),
		awsCfg.WithLogger(NewLogger(logger)),
		awsCfg.WithClientLogMode(aws.ClientLogMode(0)),
		awsCfg.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(options *retry.StandardOptions) {
				options.Backoff = NewExponentialBackoffDelayer(clock, &settings.Retry)
			})
		}),
	}
	options = append(options, optFns...)

	return options
}

func DefaultClientConfig(ctx context.Context, logger log.Logger, clock clock.Clock, settings ClientSettings, optFns ...func(options *awsCfg.LoadOptions) error) (aws.Config, error) {
	var err error
	var awsConfig aws.Config
	var options = DefaultClientOptions(logger, clock, settings, optFns...)

	if awsConfig, err = awsCfg.LoadDefaultConfig(ctx, options...); err != nil {
		return awsConfig, fmt.Errorf("can not initialize config: %w", err)
	}

	awsConfig.APIOptions = append(awsConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Initialize.Add(AttemptLoggerInitMiddleware(logger, clock, settings.Retry.MaxElapsedTime), middleware.After)
	})
	awsConfig.APIOptions = append(awsConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Finalize.Insert(AttemptLoggerRetryMiddleware(logger, clock), "Retry", middleware.After)
	})

	if settings.HttpClient.Timeout > 0 {
		awsConfig.HTTPClient = awsHttp.NewBuildableClient().WithTimeout(settings.HttpClient.Timeout)
	}

	return awsConfig, nil
}

func WithEndpoint(url string) func(options *awsCfg.LoadOptions) error {
	return func(o *awsCfg.LoadOptions) error {
		o.EndpointResolver = EndpointResolver(url)
		return nil
	}
}

func EndpointResolver(url string) aws.EndpointResolverFunc {
	return func(service, region string) (aws.Endpoint, error) {
		if url == "" {
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		}

		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           url,
			SigningRegion: region,
		}, nil
	}
}

type Logger struct {
	base log.Logger
}

func NewLogger(base log.Logger) *Logger {
	return &Logger{
		base: base,
	}
}

func (l Logger) Logf(classification logging.Classification, format string, v ...interface{}) {
	switch classification {
	case logging.Warn:
		l.base.Warn(format, v...)
	default:
		l.base.Info(format, v...)
	}
}

func (l Logger) WithContext(ctx context.Context) logging.Logger {
	return &Logger{
		base: l.base.WithContext(ctx),
	}
}

type ExponentialBackoffDelayer struct {
	backoff *backoff.ExponentialBackOff
}

func NewExponentialBackoffDelayer(clock clock.Clock, settings *ClientRetrySettings) *ExponentialBackoffDelayer {
	backoff := backoff.NewExponentialBackOff()
	backoff.Clock = clock
	backoff.InitialInterval = settings.InitialInterval
	backoff.MaxInterval = settings.MaxInterval
	backoff.MaxElapsedTime = settings.MaxElapsedTime

	return &ExponentialBackoffDelayer{
		backoff: backoff,
	}
}

func (d *ExponentialBackoffDelayer) BackoffDelay(attempt int, err error) (time.Duration, error) {
	return d.backoff.NextBackOff(), nil
}
