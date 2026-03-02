package stream

import (
	"context"
	"fmt"
	"strconv"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

const (
	attributeRetrySqs         = "goso.retry.sqs"
	MessageBodyBase64Encoding = "base64"
)

func init() {
	retryHandlers["sqs"] = NewRetryHandlerSqs
}

type RetryHandlerSqsSettings struct {
	cfg.ResourceIdentifier
	RetryHandlerSettings
	ClientName          string                     `cfg:"client_name" default:"default"`
	MaxNumberOfMessages int32                      `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32                      `cfg:"wait_time" default:"10"`
	RunnerCount         int                        `cfg:"runner_count" default:"1"`
	QueueId             string                     `cfg:"queue_id"`
	Healthcheck         health.HealthCheckSettings `cfg:"healthcheck"`
	MsgBodyEncoding     string                     `cfg:"msg_body_encoding" default:"raw"`
	Unmarshaller        string                     `cfg:"unmarshaller" default:"msg"`
}

type RetryHandlerSqs struct {
	output   Output
	settings *RetryHandlerSqsSettings
}

func NewRetryHandlerSqs(ctx context.Context, config cfg.Config, logger log.Logger, md RetryMetadata) (Input, RetryHandler, error) {
	var (
		err      error
		input    AcknowledgeableInput
		output   Output
		settings = &RetryHandlerSqsSettings{}
	)

	err = config.UnmarshalKey(md.retryConfigKey, settings)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal retry handler settings for %q: %w", md.name, err)
	}

	if err := settings.PadFromConfig(config); err != nil {
		return nil, nil, fmt.Errorf("failed to pad resource identifier for retry handler sqs %s: %w", md.name, err)
	}

	if settings.QueueId == "" {
		settings.QueueId = fmt.Sprintf("consumer-retry-%s", md.name)
	}

	identity := settings.ToIdentity()

	inputSettings := &SqsInputSettings{
		Identity:            identity,
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
		Healthcheck:  settings.Healthcheck,
		Unmarshaller: settings.Unmarshaller,
	}

	if input, err = NewSqsInput(ctx, config, logger, inputSettings); err != nil {
		return nil, nil, fmt.Errorf("can not create input: %w", err)
	}

	outputSettings := &SqsOutputSettings{
		Identity:          inputSettings.Identity,
		QueueId:           inputSettings.QueueId,
		VisibilityTimeout: inputSettings.VisibilityTimeout,
		RedrivePolicy:     inputSettings.RedrivePolicy,
		ClientName:        inputSettings.ClientName,
	}

	if output, err = NewSqsOutput(ctx, config, logger, outputSettings); err != nil {
		return nil, nil, fmt.Errorf("can not create input: %w", err)
	}

	return input, NewRetryHandlerSqsWithInterfaces(output, settings), nil
}

func NewRetryHandlerSqsWithInterfaces(output Output, settings *RetryHandlerSqsSettings) *RetryHandlerSqs {
	return &RetryHandlerSqs{
		output:   output,
		settings: settings,
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

	err := r.EncodeMessageBody(msg)
	if err != nil {
		return fmt.Errorf("failed to encode retry message body: %w", err)
	}

	if err := r.output.WriteOne(ctx, msg); err != nil {
		return fmt.Errorf("can not write the message to the output: %w", err)
	}

	return nil
}

func (r *RetryHandlerSqs) EncodeMessageBody(msg *Message) error {
	switch r.settings.MsgBodyEncoding {
	case MessageBodyBase64Encoding:
		return r.encodeMessageBodyBase64(msg)
	default:
		return nil
	}
}

func (r *RetryHandlerSqs) encodeMessageBodyBase64(msg *Message) error {
	msg.Body = base64.EncodeToString([]byte(msg.Body))

	return nil
}
