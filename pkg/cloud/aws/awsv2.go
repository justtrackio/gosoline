package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
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
	Region      string             `cfg:"region" default:"eu-central-1"`
	Endpoint    string             `cfg:"endpoint" default:"http://localhost:4566"`
	AssumeRole  string             `cfg:"assume_role"`
	Profile     string             `cfg:"profile"`
	Credentials Credentials        `cfg:"credentials"`
	HttpClient  ClientHttpSettings `cfg:"http_client"`
	Backoff     exec.BackoffSettings
}

func (s *ClientSettings) SetBackoff(backoff exec.BackoffSettings) {
	s.Backoff = backoff
}

func (s *ClientSettings) LogFields() log.Fields {
	return log.Fields{
		"settings_region":                   s.Region,
		"settings_endpoint":                 s.Endpoint,
		"settings_assume_role":              s.AssumeRole,
		"settings_http_client_timeout":      s.HttpClient.Timeout,
		"settings_backoff_max_attempts":     s.Backoff.MaxAttempts,
		"settings_backoff_max_interval":     s.Backoff.MaxInterval,
		"settings_backoff_initial_interval": s.Backoff.InitialInterval,
		"settings_backoff_cancel_delay":     s.Backoff.CancelDelay,
		"settings_backoff_max_elapsed_time": s.Backoff.MaxElapsedTime,
	}
}

func LogNewClientCreated(ctx context.Context, logger log.Logger, service string, clientName string, settings ClientSettings) {
	logger.WithContext(ctx).WithFields(settings.LogFields()).WithFields(log.Fields{
		"aws_service":     service,
		"aws_client_name": clientName,
	}).Info("created new %s client %s", service, clientName)
}

func UnmarshalClientSettings(config cfg.Config, settings ClientSettingsAware, service string, name string) error {
	if name == "" {
		name = "default"
	}

	clientsKey := GetClientConfigKey(service, name)
	defaultClientKey := GetClientConfigKey(service, "default")
	clientDefaultsKey := GetDefaultsKey(name)
	defaultsKey := GetDefaultsKey("default")

	defaults := []cfg.UnmarshalDefaults{
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.credentials", "credentials"),
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.region", "region"),
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.endpoint", "endpoint"),
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.http_client", "http_client"),
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.assume_role", "assume_role"),
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults.profile", "profile"),
		cfg.UnmarshalWithDefaultsFromKey(defaultsKey, "."),
		cfg.UnmarshalWithDefaultsFromKey(clientDefaultsKey, "."),
		cfg.UnmarshalWithDefaultsFromKey(defaultClientKey, "."),
	}

	if err := config.UnmarshalKey(clientsKey, settings, defaults...); err != nil {
		panic(fmt.Errorf("failed to unmarshal client settings for %s: %w", clientsKey, err))
	}

	backoffSettings, err := exec.ReadBackoffSettings(config, clientsKey, clientDefaultsKey, defaultClientKey, defaultsKey, "cloud.aws.defaults")
	if err != nil {
		return fmt.Errorf("failed to unmarshal client settings for %s: %w", clientsKey, err)
	}

	settings.SetBackoff(backoffSettings)

	return nil
}

func DefaultClientOptions(ctx context.Context, _ cfg.Config, logger log.Logger, clientConfig ClientConfigAware) ([]func(options *awsCfg.LoadOptions) error, error) {
	settings := clientConfig.GetSettings()

	options := []func(options *awsCfg.LoadOptions) error{
		awsCfg.WithRegion(settings.Region),
		awsCfg.WithLogger(NewLogger(logger)),
		awsCfg.WithClientLogMode(aws.ClientLogMode(0)),
		awsCfg.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(DefaultClientRetryOptions(clientConfig)...)
		}),
	}

	var err error
	var credentialsOption func(options *awsCfg.LoadOptions) error

	if credentialsOption, err = GetCredentialsOption(ctx, settings); err != nil {
		return nil, fmt.Errorf("can not get credentials option: %w", err)
	}

	if credentialsOption != nil {
		options = append(options, credentialsOption)
	}

	options = append(options, clientConfig.GetLoadOptions()...)

	return options, nil
}

func DefaultClientConfig(ctx context.Context, config cfg.Config, logger log.Logger, clientConfig ClientConfigAware) (aws.Config, error) {
	var err error
	var options []func(options *awsCfg.LoadOptions) error
	var awsConfig aws.Config

	settings := clientConfig.GetSettings()

	if options, err = DefaultClientOptions(ctx, config, logger, clientConfig); err != nil {
		return awsConfig, fmt.Errorf("can not get default client options: %w", err)
	}

	if awsConfig, err = awsCfg.LoadDefaultConfig(ctx, options...); err != nil {
		return awsConfig, fmt.Errorf("can not initialize config: %w", err)
	}

	awsConfig.BaseEndpoint = NilIfEmpty(settings.Endpoint)

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

type Logger struct {
	base log.Logger
}

func NewLogger(base log.Logger) *Logger {
	return &Logger{
		base: base,
	}
}

func (l Logger) Logf(classification logging.Classification, format string, v ...any) {
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

func GetDefaultsKey(name string) string {
	return GetClientConfigKey("defaults", name)
}

func NilIfEmpty[T comparable](val T) *T {
	if *new(T) == val {
		return nil
	}

	return &val
}
