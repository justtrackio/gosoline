package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/cloud/aws/kinesis"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/sqs"
	"time"
)

const (
	InputTypeFile     = "file"
	InputTypeInMemory = "inMemory"
	InputTypeKinesis  = "kinesis"
	InputTypeRedis    = "redis"
	InputTypeSns      = "sns"
	InputTypeSqs      = "sqs"
)

type InputFactory func(config cfg.Config, logger log.Logger, name string) (Input, error)

var inputFactories = map[string]InputFactory{
	InputTypeFile:     newFileInputFromConfig,
	InputTypeInMemory: newInMemoryInputFromConfig,
	InputTypeKinesis:  newKinesisInputFromConfig,
	InputTypeRedis:    newRedisInputFromConfig,
	InputTypeSns:      newSnsInputFromConfig,
	InputTypeSqs:      newSqsInputFromConfig,
}

func SetInputFactory(typ string, factory InputFactory) {
	inputFactories[typ] = factory
}

var inputs = map[string]Input{}

func ProvideConfigurableInput(config cfg.Config, logger log.Logger, name string) (Input, error) {
	var ok bool
	var err error
	var input Input

	if input, ok = inputs[name]; ok {
		return input, nil
	}

	if inputs[name], err = NewConfigurableInput(config, logger, name); err != nil {
		return nil, err
	}

	return inputs[name], nil
}

func NewConfigurableInput(config cfg.Config, logger log.Logger, name string) (Input, error) {
	t := readInputType(config, name)

	factory, ok := inputFactories[t]

	if !ok {
		return nil, fmt.Errorf("invalid input %s of type %s", name, t)
	}

	input, err := factory(config, logger, name)

	if err != nil {
		return nil, fmt.Errorf("failed to create input: %w", err)
	}

	return input, nil
}

func newFileInputFromConfig(config cfg.Config, logger log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)
	settings := FileSettings{}
	config.UnmarshalKey(key, &settings)

	return NewFileInput(config, logger, settings), nil
}

func newInMemoryInputFromConfig(config cfg.Config, _ log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)
	settings := &InMemorySettings{}
	config.UnmarshalKey(key, settings)

	return ProvideInMemoryInput(name, settings), nil
}

type kinesisInputConfiguration struct {
	StreamName      string `cfg:"stream_name" validate:"required"`
	ApplicationName string `cfg:"application_name" validate:"required"`
}

func newKinesisInputFromConfig(config cfg.Config, logger log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)

	settings := kinesisInputConfiguration{}
	config.UnmarshalKey(key, &settings)

	readerSettings := kinesis.KinsumerSettings{
		StreamName:      settings.StreamName,
		ApplicationName: settings.ApplicationName,
	}

	return NewKinesisInput(config, logger, kinesis.NewKinsumer, readerSettings)
}

type redisInputConfiguration struct {
	Project     string        `cfg:"project"`
	Family      string        `cfg:"family"`
	Application string        `cfg:"application"`
	ServerName  string        `cfg:"server_name" default:"default" validate:"min=1"`
	Key         string        `cfg:"key" validate:"required,min=1"`
	WaitTime    time.Duration `cfg:"wait_time" default:"3s"`
}

func newRedisInputFromConfig(config cfg.Config, logger log.Logger, name string) (Input, error) {
	key := ConfigurableInputKey(name)

	configuration := redisInputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	settings := &RedisListInputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		ServerName: configuration.ServerName,
		Key:        configuration.Key,
		WaitTime:   configuration.WaitTime,
	}

	return NewRedisListInput(config, logger, settings)
}

type SnsInputTargetConfiguration struct {
	Family      string                 `cfg:"family"`
	Application string                 `cfg:"application" validate:"required"`
	TopicId     string                 `cfg:"topic_id" validate:"required"`
	Attributes  map[string]interface{} `cfg:"attributes"`
}

type SnsInputConfiguration struct {
	Type                string                        `cfg:"type" default:"sns"`
	ConsumerId          string                        `cfg:"id" validate:"required"`
	Family              string                        `cfg:"family" default:""`
	Application         string                        `cfg:"application" default:""`
	Targets             []SnsInputTargetConfiguration `cfg:"targets" validate:"min=1"`
	MaxNumberOfMessages int64                         `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int64                         `cfg:"wait_time" default:"3" validate:"min=1"`
	VisibilityTimeout   int                           `cfg:"visibility_timeout" default:"30" validate:"min=1"`
	RunnerCount         int                           `cfg:"runner_count" default:"1" validate:"min=1"`
	RedrivePolicy       sqs.RedrivePolicy             `cfg:"redrive_policy"`
	Client              cloud.ClientSettings          `cfg:"client"`
	Backoff             exec.BackoffSettings          `cfg:"backoff"`
}

func readSnsInputSettings(config cfg.Config, name string) (SnsInputSettings, []SnsInputTarget) {
	key := ConfigurableInputKey(name)

	configuration := &SnsInputConfiguration{}
	config.UnmarshalKey(key, configuration, cfg.UnmarshalWithDefaultsFromKey(ConfigKeyStreamBackoff, "backoff"))

	settings := SnsInputSettings{
		AppId: cfg.AppId{
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		QueueId:             configuration.ConsumerId,
		MaxNumberOfMessages: configuration.MaxNumberOfMessages,
		WaitTime:            configuration.WaitTime,
		VisibilityTimeout:   configuration.VisibilityTimeout,
		RunnerCount:         configuration.RunnerCount,
		RedrivePolicy:       configuration.RedrivePolicy,
		Client:              configuration.Client,
		Backoff:             configuration.Backoff,
	}

	settings.PadFromConfig(config)

	targets := make([]SnsInputTarget, len(configuration.Targets))
	for i, t := range configuration.Targets {
		targets[i] = SnsInputTarget{
			AppId: cfg.AppId{
				Family:      t.Family,
				Application: t.Application,
			},
			TopicId:    t.TopicId,
			Attributes: t.Attributes,
		}
	}

	return settings, targets
}

func newSnsInputFromConfig(config cfg.Config, logger log.Logger, name string) (Input, error) {
	settings, targets := readSnsInputSettings(config, name)

	return NewSnsInput(config, logger, settings, targets)
}

type sqsInputConfiguration struct {
	Family              string               `cfg:"target_family"`
	Application         string               `cfg:"target_application"`
	QueueId             string               `cfg:"target_queue_id" validate:"min=1"`
	MaxNumberOfMessages int64                `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int64                `cfg:"wait_time" default:"3" validate:"min=1"`
	VisibilityTimeout   int                  `cfg:"visibility_timeout" default:"30" validate:"min=1"`
	RunnerCount         int                  `cfg:"runner_count" default:"1" validate:"min=1"`
	Fifo                sqs.FifoSettings     `cfg:"fifo"`
	RedrivePolicy       sqs.RedrivePolicy    `cfg:"redrive_policy"`
	Client              cloud.ClientSettings `cfg:"client"`
	Backoff             exec.BackoffSettings `cfg:"backoff"`
	Unmarshaller        string               `cfg:"unmarshaller" default:"msg"`
}

func readSqsInputSettings(config cfg.Config, name string) SqsInputSettings {
	key := ConfigurableInputKey(name)

	configuration := sqsInputConfiguration{}
	config.UnmarshalKey(key, &configuration, cfg.UnmarshalWithDefaultsFromKey(ConfigKeyStreamBackoff, "backoff"))

	settings := SqsInputSettings{
		AppId: cfg.AppId{
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		QueueId:             configuration.QueueId,
		MaxNumberOfMessages: configuration.MaxNumberOfMessages,
		WaitTime:            configuration.WaitTime,
		VisibilityTimeout:   configuration.VisibilityTimeout,
		RunnerCount:         configuration.RunnerCount,
		Fifo:                configuration.Fifo,
		RedrivePolicy:       configuration.RedrivePolicy,
		Client:              configuration.Client,
		Backoff:             configuration.Backoff,
		Unmarshaller:        configuration.Unmarshaller,
	}

	settings.PadFromConfig(config)

	return settings
}

func newSqsInputFromConfig(config cfg.Config, logger log.Logger, name string) (Input, error) {
	settings := readSqsInputSettings(config, name)

	return NewSqsInput(config, logger, settings)
}

func ConfigurableInputKey(name string) string {
	return fmt.Sprintf("stream.input.%s", name)
}

func readInputType(config cfg.Config, name string) string {
	key := fmt.Sprintf("%s.type", ConfigurableInputKey(name))
	t := config.GetString(key)

	return t
}

func readAllInputTypes(config cfg.Config) map[string]string {
	inputMap := config.GetStringMap("stream.input", map[string]interface{}{})
	inputsTypes := make(map[string]string, len(inputMap))

	for name := range inputMap {
		inputsTypes[name] = readInputType(config, name)
	}

	return inputsTypes
}
