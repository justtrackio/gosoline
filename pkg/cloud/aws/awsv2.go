package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/smithy-go/logging"
	"github.com/aws/smithy-go/middleware"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ClientConfigAware interface {
	GetSettings() ClientSettings
	GetLoadOptions() []func(options *awsCfg.LoadOptions) error
	GetRetryOptions() []func(*retry.StandardOptions)
}

type ClientSettingsAware interface {
	SetBackoff(backoff exec.BackoffSettings)
}

type Credentials struct {
	AccessKeyID     string `cfg:"access_key_id"`
	SecretAccessKey string `cfg:"secret_access_key"`
	SessionToken    string `cfg:"session_token"`
}

type ClientHttpSettings struct {
	Timeout time.Duration `cfg:"timeout" default:"0"`
}

type ClientSettings struct {
	Region     string             `cfg:"region" default:"eu-central-1"`
	Endpoint   string             `cfg:"endpoint" default:"http://localhost:4566"`
	HttpClient ClientHttpSettings `cfg:"http_client"`
	Backoff    exec.BackoffSettings
}

func (s *ClientSettings) SetBackoff(backoff exec.BackoffSettings) {
	s.Backoff = backoff
}

func UnmarshalClientSettings(config cfg.Config, settings ClientSettingsAware, service string, name string) {
	if name == "" {
		name = "default"
	}

	clientsKey := GetClientConfigKey(service, name)
	defaultClientKey := GetClientConfigKey(service, "default")

	config.UnmarshalKey(clientsKey, settings, []cfg.UnmarshalDefaults{
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.region", "region"),
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.endpoint", "endpoint"),
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.http_client", "http_client"),
		cfg.UnmarshalWithDefaultsFromKey(defaultClientKey, "."),
	}...)

	backoffSettings := exec.ReadBackoffSettings(config, clientsKey, "cloud.aws.defaults")
	settings.SetBackoff(backoffSettings)
}

func UnmarshalCredentials(config cfg.Config) *Credentials {
	if !config.IsSet("cloud.aws.credentials") {
		return nil
	}

	creds := &Credentials{}
	config.UnmarshalKey("cloud.aws.credentials", creds)

	return creds
}

func DefaultClientOptions(config cfg.Config, logger log.Logger, clientConfig ClientConfigAware) []func(options *awsCfg.LoadOptions) error {
	settings := clientConfig.GetSettings()

	options := []func(options *awsCfg.LoadOptions) error{
		awsCfg.WithRegion(settings.Region),
		awsCfg.WithEndpointResolverWithOptions(EndpointResolver(settings.Endpoint)),
		awsCfg.WithLogger(NewLogger(logger)),
		awsCfg.WithClientLogMode(aws.ClientLogMode(0)),
		awsCfg.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(DefaultClientRetryOptions(clientConfig)...)
		}),
	}

	if creds := UnmarshalCredentials(config); creds != nil {
		credentialsProvider := credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)
		options = append(options, awsCfg.WithCredentialsProvider(credentialsProvider))
	}

	options = append(options, clientConfig.GetLoadOptions()...)

	return options
}

func DefaultClientConfig(ctx context.Context, config cfg.Config, logger log.Logger, clientConfig ClientConfigAware) (aws.Config, error) {
	var err error
	var awsConfig aws.Config

	settings := clientConfig.GetSettings()
	options := DefaultClientOptions(config, logger, clientConfig)

	if awsConfig, err = awsCfg.LoadDefaultConfig(ctx, options...); err != nil {
		return awsConfig, fmt.Errorf("can not initialize config: %w", err)
	}

	awsConfig.APIOptions = append(awsConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Initialize.Add(AttemptLoggerInitMiddleware(logger, &settings.Backoff), middleware.After)
	})
	awsConfig.APIOptions = append(awsConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Finalize.Insert(AttemptLoggerRetryMiddleware(logger), "Retry", middleware.After)
	})

	if settings.HttpClient.Timeout > 0 {
		awsConfig.HTTPClient = awsHttp.NewBuildableClient().WithTimeout(settings.HttpClient.Timeout)
	}

	return awsConfig, nil
}

func WithEndpoint(url string) func(options *awsCfg.LoadOptions) error {
	return func(o *awsCfg.LoadOptions) error {
		o.EndpointResolverWithOptions = EndpointResolver(url)
		return nil
	}
}

type endpointResolver struct {
	url string
}

func (e *endpointResolver) ResolveEndpoint(service, region string, options ...interface{}) (aws.Endpoint, error) {
	if e.url == "" {
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	}

	return aws.Endpoint{
		PartitionID:   "aws",
		URL:           e.url,
		SigningRegion: region,
	}, nil
}

func EndpointResolver(url string) *endpointResolver {
	return &endpointResolver{
		url: url,
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

func GetClientConfigKey(service string, name string) string {
	return fmt.Sprintf("cloud.aws.%s.clients.%s", service, name)
}
