package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

const (
	InputTypeFile     = "file"
	InputTypeInMemory = "inMemory"
	InputTypeKinesis  = "kinesis"
	InputTypeRedis    = "redis"
	InputTypeSns      = "sns"
	InputTypeSqs      = "sqs"
	InputTypeKafka    = "kafka"
)

type InputFactory func(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error)

var inputFactories = map[string]InputFactory{
	InputTypeFile:     newFileInputFromConfig,
	InputTypeInMemory: newInMemoryInputFromConfig,
	InputTypeKinesis:  newKinesisInputFromConfig,
	InputTypeRedis:    newRedisInputFromConfig,
	InputTypeSns:      newSnsInputFromConfig,
	InputTypeSqs:      newSqsInputFromConfig,
	InputTypeKafka:    newKafkaInputFromConfig,
}

func SetInputFactory(typ string, factory InputFactory) {
	inputFactories[typ] = factory
}

var inputs = map[string]Input{}

func ProvideConfigurableInput(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
	var ok bool
	var err error
	var input Input

	if input, ok = inputs[name]; ok {
		return input, nil
	}

	if inputs[name], err = NewConfigurableInput(ctx, config, logger, name); err != nil {
		return nil, err
	}

	return inputs[name], nil
}

func NewConfigurableInput(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
	t, err := readInputType(config, name)
	if err != nil {
		return nil, fmt.Errorf("could not read input type: %w", err)
	}

	factory, ok := inputFactories[t]

	if !ok {
		return nil, fmt.Errorf("invalid input %s of type %s", name, t)
	}

	input, err := factory(ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create input: %w", err)
	}

	return input, nil
}

func newFileInputFromConfig(_ context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)
	settings := FileSettings{}
	if err := config.UnmarshalKey(key, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file input settings: %w", err)
	}

	return NewFileInput(config, logger, settings), nil
}

func newInMemoryInputFromConfig(_ context.Context, config cfg.Config, _ log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)
	settings := &InMemorySettings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal in-memory input settings: %w", err)
	}

	return ProvideInMemoryInput(name, settings), nil
}

func newKafkaInputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)

	return NewKafkaInput(ctx, config, logger, key)
}

type KinesisInputConfiguration struct {
	kinesis.Settings
	Type string `cfg:"type" default:"kinesis"`
}

func newKinesisInputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)

	settings := KinesisInputConfiguration{}
	if err := config.UnmarshalKey(key, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kinesis input settings: %w", err)
	}
	settings.Name = name

	return NewKinesisInput(ctx, config, logger, settings.Settings)
}

type redisInputConfiguration struct {
	Project     string                     `cfg:"project"`
	Family      string                     `cfg:"family"`
	Group       string                     `cfg:"group"`
	Application string                     `cfg:"application"`
	ServerName  string                     `cfg:"server_name" default:"default" validate:"min=1"`
	Key         string                     `cfg:"key" validate:"required,min=1"`
	WaitTime    time.Duration              `cfg:"wait_time" default:"3s"`
	Healthcheck health.HealthCheckSettings `cfg:"healthcheck"`
}

func newRedisInputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)

	configuration := redisInputConfiguration{}
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal redis input settings: %w", err)
	}

	settings := &RedisListInputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Group:       configuration.Group,
			Application: configuration.Application,
		},
		ServerName:         configuration.ServerName,
		Key:                configuration.Key,
		WaitTime:           configuration.WaitTime,
		HealthcheckTimeout: configuration.Healthcheck.Timeout,
	}

	return NewRedisListInput(ctx, config, logger, settings)
}

type SnsInputTargetConfiguration struct {
	Family      string            `cfg:"family"`
	Group       string            `cfg:"group" validate:"required"`
	Application string            `cfg:"application" validate:"required"`
	TopicId     string            `cfg:"topic_id" validate:"required"`
	Attributes  map[string]string `cfg:"attributes"`
	ClientName  string            `cfg:"client_name" default:"default"`
}

type SnsInputConfiguration struct {
	Type                string                        `cfg:"type" default:"sns"`
	ConsumerId          string                        `cfg:"id" validate:"required"`
	Family              string                        `cfg:"family" default:""`
	Group               string                        `cfg:"group" default:""`
	Application         string                        `cfg:"application" default:""`
	Targets             []SnsInputTargetConfiguration `cfg:"targets" validate:"min=1"`
	MaxNumberOfMessages int32                         `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32                         `cfg:"wait_time" default:"3" validate:"min=1"`
	VisibilityTimeout   int                           `cfg:"visibility_timeout" default:"30" validate:"min=1"`
	RunnerCount         int                           `cfg:"runner_count" default:"1" validate:"min=1"`
	RedrivePolicy       sqs.RedrivePolicy             `cfg:"redrive_policy"`
	ClientName          string                        `cfg:"client_name" default:"default"`
	Healthcheck         health.HealthCheckSettings    `cfg:"healthcheck"`
}

func readSnsInputSettings(config cfg.Config, name string) (*SnsInputSettings, []SnsInputTarget, error) {
	key := ConfigurableInputKey(name)

	configuration := &SnsInputConfiguration{}
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal sns input settings for key %q in readSnsInputSettings: %w", key, err)
	}

	settings := &SnsInputSettings{
		AppId: cfg.AppId{
			Family:      configuration.Family,
			Group:       configuration.Group,
			Application: configuration.Application,
		},
		QueueId:             configuration.ConsumerId,
		MaxNumberOfMessages: configuration.MaxNumberOfMessages,
		WaitTime:            configuration.WaitTime,
		VisibilityTimeout:   configuration.VisibilityTimeout,
		RunnerCount:         configuration.RunnerCount,
		RedrivePolicy:       configuration.RedrivePolicy,
		ClientName:          configuration.ClientName,
		Healthcheck:         configuration.Healthcheck,
	}

	if err := settings.PadFromConfig(config); err != nil {
		return nil, nil, fmt.Errorf("failed to pad sns input settings from config: %w", err)
	}

	targets := make([]SnsInputTarget, len(configuration.Targets))
	for i, t := range configuration.Targets {
		targetAppId := cfg.AppId{
			Family:      t.Family,
			Group:       t.Group,
			Application: t.Application,
		}

		if err := targetAppId.PadFromConfig(config); err != nil {
			return nil, nil, fmt.Errorf("failed to pad target app id from config: %w", err)
		}

		clientName := t.ClientName
		if clientName == "" {
			clientName = "default"
		}

		targets[i] = SnsInputTarget{
			AppId:      targetAppId,
			TopicId:    t.TopicId,
			Attributes: t.Attributes,
			ClientName: clientName,
		}
	}

	return settings, targets, nil
}

func newSnsInputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
	settings, targets, err := readSnsInputSettings(config, name)
	if err != nil {
		return nil, fmt.Errorf("failed to read sns input settings in newSnsInputFromConfig: %w", err)
	}

	return NewSnsInput(ctx, config, logger, settings, targets)
}

type sqsInputConfiguration struct {
	Family              string                     `cfg:"target_family"`
	Group               string                     `cfg:"target_group"`
	Application         string                     `cfg:"target_application"`
	QueueId             string                     `cfg:"target_queue_id" validate:"min=1"`
	MaxNumberOfMessages int32                      `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32                      `cfg:"wait_time" default:"3" validate:"min=1"`
	VisibilityTimeout   int                        `cfg:"visibility_timeout" default:"30" validate:"min=1"`
	RunnerCount         int                        `cfg:"runner_count" default:"1" validate:"min=1"`
	Fifo                sqs.FifoSettings           `cfg:"fifo"`
	RedrivePolicy       sqs.RedrivePolicy          `cfg:"redrive_policy"`
	ClientName          string                     `cfg:"client_name" default:"default"`
	Healthcheck         health.HealthCheckSettings `cfg:"healthcheck"`
	Unmarshaller        string                     `cfg:"unmarshaller" default:"msg"`
}

func readSqsInputSettings(config cfg.Config, name string) (*SqsInputSettings, error) {
	key := ConfigurableInputKey(name)

	configuration := sqsInputConfiguration{}
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sqs input settings for key %q in readSqsInputSettings: %w", key, err)
	}

	settings := &SqsInputSettings{
		AppId: cfg.AppId{
			Family:      configuration.Family,
			Group:       configuration.Group,
			Application: configuration.Application,
		},
		QueueId:             configuration.QueueId,
		MaxNumberOfMessages: configuration.MaxNumberOfMessages,
		WaitTime:            configuration.WaitTime,
		VisibilityTimeout:   configuration.VisibilityTimeout,
		RunnerCount:         configuration.RunnerCount,
		Fifo:                configuration.Fifo,
		RedrivePolicy:       configuration.RedrivePolicy,
		ClientName:          configuration.ClientName,
		Healthcheck:         configuration.Healthcheck,
		Unmarshaller:        configuration.Unmarshaller,
	}

	if err := settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("failed to pad sqs input settings from config: %w", err)
	}

	return settings, nil
}

func newSqsInputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
	settings, err := readSqsInputSettings(config, name)
	if err != nil {
		return nil, fmt.Errorf("failed to read sqs input settings in newSqsInputFromConfig: %w", err)
	}

	return NewSqsInput(ctx, config, logger, settings)
}

func ConfigurableInputKey(name string) string {
	return fmt.Sprintf("stream.input.%s", name)
}

func readInputType(config cfg.Config, name string) (string, error) {
	key := fmt.Sprintf("%s.type", ConfigurableInputKey(name))
	t, err := config.GetString(key)
	if err != nil {
		return "", fmt.Errorf("could not get string for key %s: %w", key, err)
	}

	return t, nil
}

func readAllInputTypes(config cfg.Config) (map[string]string, error) {
	inputMap, err := config.GetStringMap("stream.input", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("could not get string map for key stream.input: %w", err)
	}

	inputsTypes := make(map[string]string, len(inputMap))

	for name := range inputMap {
		var err error
		inputsTypes[name], err = readInputType(config, name)
		if err != nil {
			return nil, fmt.Errorf("could not read input type for %s: %w", name, err)
		}
	}

	return inputsTypes, nil
}
