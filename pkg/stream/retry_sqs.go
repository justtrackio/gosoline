package stream

import (
	"context"
	"fmt"
	"strconv"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/log"
)

const attributeRetrySqs = "goso.retry.sqs"

func init() {
	retryHandlers["sqs"] = NewRetryHandlerSqs
}

type RetryHandlerSqsSettings struct {
	cfg.AppId
	RetryHandlerSettings
	ClientName          string `cfg:"client_name" default:"default"`
	MaxNumberOfMessages int32  `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32  `cfg:"wait_time" default:"10"`
	RunnerCount         int    `cfg:"runner_count" default:"1"`
	QueueId             string `cfg:"queue_id"`
}

type RetryHandlerSqs struct {
	AcknowledgeableInput
	output   Output
	settings *RetryHandlerSqsSettings
}

func NewRetryHandlerSqs(ctx context.Context, config cfg.Config, logger log.Logger, name string) (RetryHandler, error) {
	var err error
	var input AcknowledgeableInput
	var output Output

	key := ConfigurableConsumerRetryKey(name)
	settings := &RetryHandlerSqsSettings{}
	config.UnmarshalKey(key, settings)

	if settings.QueueId == "" {
		settings.QueueId = fmt.Sprintf("consumer-retry-%s", name)
	}

	inputSettings := &SqsInputSettings{
		AppId:               settings.AppId,
		QueueId:             settings.QueueId,
		MaxNumberOfMessages: settings.MaxNumberOfMessages,
		WaitTime:            settings.WaitTime,
		VisibilityTimeout:   int(settings.After.Seconds()),
		RunnerCount:         settings.RunnerCount,
		RedrivePolicy: sqs.RedrivePolicy{
			Enabled:         true,
			MaxReceiveCount: settings.MaxAttempts,
		},
		ClientName:   settings.ClientName,
		Unmarshaller: UnmarshallerMsg,
	}

	if input, err = NewSqsInput(ctx, config, logger, inputSettings); err != nil {
		return nil, fmt.Errorf("can not create input: %w", err)
	}

	outputSettings := &SqsOutputSettings{
		AppId:             inputSettings.AppId,
		QueueId:           inputSettings.QueueId,
		VisibilityTimeout: inputSettings.VisibilityTimeout,
		RedrivePolicy:     inputSettings.RedrivePolicy,
		ClientName:        inputSettings.ClientName,
	}

	if output, err = NewSqsOutput(ctx, config, logger, outputSettings); err != nil {
		return nil, fmt.Errorf("can not create input: %w", err)
	}

	return NewRetryHandlerSqsWithInterfaces(input, output, settings), nil
}

func NewRetryHandlerSqsWithInterfaces(input AcknowledgeableInput, output Output, settings *RetryHandlerSqsSettings) *RetryHandlerSqs {
	return &RetryHandlerSqs{
		AcknowledgeableInput: input,
		output:               output,
		settings:             settings,
	}
}

func (r *RetryHandlerSqs) Put(ctx context.Context, msg *Message) error {
	// do not put it back into retry if it's already in there
	// sqs will redeliver the message automatically
	if _, ok := msg.Attributes[attributeRetrySqs]; ok {
		return nil
	}

	msg.Attributes[attributeRetrySqs] = strconv.FormatBool(true)
	msg.Attributes[sqs.AttributeSqsDelaySeconds] = strconv.Itoa(int(r.settings.After.Seconds()))

	if err := r.output.WriteOne(ctx, msg); err != nil {
		return fmt.Errorf("can not write the message to the output: %w", err)
	}

	return nil
}
