package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

type SnsInputSettings struct {
	Identity            cfg.Identity               `cfg:"identity"`
	QueueId             string                     `cfg:"queue_id"`
	MaxNumberOfMessages int32                      `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32                      `cfg:"wait_time"`
	RedrivePolicy       sqs.RedrivePolicy          `cfg:"redrive_policy"`
	VisibilityTimeout   int                        `cfg:"visibility_timeout"`
	GraceTime           time.Duration              `cfg:"grace_time" default:"10s"`
	RunnerCount         int                        `cfg:"runner_count"`
	AcknowledgementMode SqsAcknowledgementMode     `cfg:"acknowledgement_mode" default:"individual" validate:"oneof=individual batch"`
	ClientName          string                     `cfg:"client_name"`
	Healthcheck         health.HealthCheckSettings `cfg:"healthcheck"`
}

func (s SnsInputSettings) GetIdentity() cfg.Identity {
	return s.Identity
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
	Identity     cfg.Identity
	TopicId      string
	FilterPolicy map[string][]string
	ClientName   string
}

func (t SnsInputTarget) GetIdentity() cfg.Identity {
	return t.Identity
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

	sqsInputSettings := newSqsInputSettingsFromSns(settings)

	if input, err = NewSqsInput(ctx, config, logger, sqsInputSettings); err != nil {
		return nil, fmt.Errorf("can not create sqsInput: %w", err)
	}

	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManager(sqsInputSettings, targets)); err != nil {
		return nil, fmt.Errorf("can not add lifecycleer: %w", err)
	}

	return NewSnsInputWithInterfaces(input), nil
}

func newSqsInputSettingsFromSns(settings *SnsInputSettings) *SqsInputSettings {
	return &SqsInputSettings{
		Identity:            settings.Identity,
		QueueId:             settings.QueueId,
		MaxNumberOfMessages: settings.MaxNumberOfMessages,
		WaitTime:            settings.WaitTime,
		VisibilityTimeout:   settings.VisibilityTimeout,
		GraceTime:           settings.GraceTime,
		RunnerCount:         settings.RunnerCount,
		AcknowledgementMode: settings.AcknowledgementMode,
		RedrivePolicy:       settings.RedrivePolicy,
		ClientName:          settings.ClientName,
		Healthcheck:         settings.Healthcheck,
		Unmarshaller:        UnmarshallerSns,
	}
}

func NewSnsInputWithInterfaces(sqsInput *sqsInput) *snsInput {
	return &snsInput{
		sqsInput,
	}
}
