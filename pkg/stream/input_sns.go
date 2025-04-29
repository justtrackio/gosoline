package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

var _ AcknowledgeableInput = &snsInput{}

type SnsInputSettings struct {
	cfg.AppId
	QueueId             string                     `cfg:"queue_id"`
	MaxNumberOfMessages int32                      `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32                      `cfg:"wait_time"`
	RedrivePolicy       sqs.RedrivePolicy          `cfg:"redrive_policy"`
	VisibilityTimeout   int                        `cfg:"visibility_timeout"`
	RunnerCount         int                        `cfg:"runner_count"`
	ClientName          string                     `cfg:"client_name"`
	Healthcheck         health.HealthCheckSettings `cfg:"healthcheck"`
}

func (s SnsInputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s SnsInputSettings) GetClientName() string {
	return s.ClientName
}

func (s SnsInputSettings) GetQueueId() string {
	return s.QueueId
}

func (s SnsInputSettings) IsFifoEnabled() bool {
	return false
}

type SnsInputTarget struct {
	cfg.AppId
	TopicId    string
	Attributes map[string]string
	ClientName string
}

func (t SnsInputTarget) GetAppId() cfg.AppId {
	return t.AppId
}

func (t SnsInputTarget) GetClientName() string {
	return t.ClientName
}

func (t SnsInputTarget) GetTopicId() string {
	return t.TopicId
}

type snsInput struct {
	*sqsInput
}

func NewSnsInput(ctx context.Context, config cfg.Config, logger log.Logger, settings *SnsInputSettings, targets []SnsInputTarget) (*snsInput, error) {
	var err error
	var input *sqsInput

	sqsInputSettings := &SqsInputSettings{
		AppId:               settings.AppId,
		QueueId:             settings.QueueId,
		MaxNumberOfMessages: settings.MaxNumberOfMessages,
		WaitTime:            settings.WaitTime,
		VisibilityTimeout:   settings.VisibilityTimeout,
		RunnerCount:         settings.RunnerCount,
		RedrivePolicy:       settings.RedrivePolicy,
		ClientName:          settings.ClientName,
		Healthcheck:         settings.Healthcheck,
		Unmarshaller:        UnmarshallerSns,
	}

	if input, err = NewSqsInput(ctx, config, logger, sqsInputSettings); err != nil {
		return nil, fmt.Errorf("can not create sqsInput: %w", err)
	}

	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManager(sqsInputSettings, targets)); err != nil {
		return nil, fmt.Errorf("can not add lifecycleer: %w", err)
	}

	return NewSnsInputWithInterfaces(input), nil
}

func NewSnsInputWithInterfaces(sqsInput *sqsInput) *snsInput {
	return &snsInput{
		sqsInput,
	}
}
