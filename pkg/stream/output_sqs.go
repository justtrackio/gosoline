package stream

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/spf13/cast"
)

const SqsOutputBatchSize = 10

type SqsOutputSettings struct {
	cfg.AppId
	ClientName        string
	Fifo              sqs.FifoSettings
	QueueId           string
	QueueNamePattern  string
	RedrivePolicy     sqs.RedrivePolicy
	VisibilityTimeout int
}

func (s SqsOutputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s SqsOutputSettings) GetClientName() string {
	return s.ClientName
}

func (s SqsOutputSettings) IsFifoEnabled() bool {
	return s.Fifo.Enabled
}

func (s SqsOutputSettings) GetQueueId() string {
	return s.QueueId
}

type sqsOutput struct {
	logger   log.Logger
	queue    sqs.Queue
	settings *SqsOutputSettings
}

func NewSqsOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *SqsOutputSettings) (Output, error) {
	settings.PadFromConfig(config)

	var err error
	var queueName string
	var queue sqs.Queue

	if queueName, err = sqs.GetQueueName(config, settings); err != nil {
		return nil, fmt.Errorf("can not get sqs queue name: %w", err)
	}

	queueSettings := &sqs.Settings{
		QueueName:         queueName,
		VisibilityTimeout: settings.VisibilityTimeout,
		Fifo:              settings.Fifo,
		RedrivePolicy:     settings.RedrivePolicy,
		ClientName:        settings.ClientName,
	}

	if queue, err = sqs.ProvideQueue(ctx, config, logger, queueSettings); err != nil {
		return nil, fmt.Errorf("can not create queue: %w", err)
	}

	return NewSqsOutputWithInterfaces(logger, queue, settings), nil
}

func NewSqsOutputWithInterfaces(logger log.Logger, queue sqs.Queue, settings *SqsOutputSettings) Output {
	return &sqsOutput{
		logger:   logger,
		queue:    queue,
		settings: settings,
	}
}

func (o *sqsOutput) WriteOne(ctx context.Context, record WritableMessage) error {
	sqsMessage, err := o.buildSqsMessage(ctx, record)
	if err != nil {
		return fmt.Errorf("could not build sqs message: %w", err)
	}

	err = o.queue.Send(ctx, sqsMessage)
	if err != nil {
		return fmt.Errorf("could not send sqs message: %w", err)
	}

	return nil
}

func (o *sqsOutput) Write(ctx context.Context, batch []WritableMessage) error {
	chunks := funk.Chunk(batch, SqsOutputBatchSize)

	var result error

	for _, chunk := range chunks {
		messages, err := o.buildSqsMessages(ctx, chunk)
		if err != nil {
			result = multierror.Append(result, err)
		}

		if len(messages) == 0 {
			continue
		}

		err = o.queue.SendBatch(ctx, messages)

		if err != nil {
			result = multierror.Append(result, err)
		}
	}

	if result != nil {
		return fmt.Errorf("there were errors on writing to the sqs stream: %w", result)
	}

	return nil
}

func (o *sqsOutput) GetMaxMessageSize() *int {
	return mdl.Box(256 * 1024)
}

func (o *sqsOutput) GetMaxBatchSize() *int {
	return mdl.Box(10)
}

func (o *sqsOutput) buildSqsMessages(ctx context.Context, messages []WritableMessage) ([]*sqs.Message, error) {
	var result error
	sqsMessages := make([]*sqs.Message, 0)

	for _, msg := range messages {
		sqsMessage, err := o.buildSqsMessage(ctx, msg)
		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

		sqsMessages = append(sqsMessages, sqsMessage)
	}

	return sqsMessages, result
}

func (o *sqsOutput) buildSqsMessage(ctx context.Context, msg WritableMessage) (*sqs.Message, error) {
	var err error
	var delay int32
	var messageGroupId string
	var messageDeduplicationId string

	attributes := getAttributes(msg)

	if d, ok := attributes[sqs.AttributeSqsDelaySeconds]; ok {
		if delay, err = cast.ToInt32E(d); err != nil {
			return nil, fmt.Errorf("the type of the %s attribute with value %v should be castable to int32: %w", sqs.AttributeSqsDelaySeconds, attributes[sqs.AttributeSqsDelaySeconds], err)
		}
	}

	if d, ok := attributes[sqs.AttributeSqsMessageGroupId]; ok {
		if messageGroupId, err = cast.ToStringE(d); err != nil {
			return nil, fmt.Errorf("the type of the %s attribute with value %v should be castable to string: %w", sqs.AttributeSqsMessageGroupId, attributes[sqs.AttributeSqsMessageGroupId], err)
		}
	}

	if d, ok := attributes[sqs.AttributeSqsMessageDeduplicationId]; ok {
		if messageDeduplicationId, err = cast.ToStringE(d); err != nil {
			return nil, fmt.Errorf("the type of the %s attribute with value %v should be castable to string: %w", sqs.AttributeSqsMessageDeduplicationId, attributes[sqs.AttributeSqsMessageDeduplicationId], err)
		}
	}

	if o.settings.Fifo.ContentBasedDeduplication && messageDeduplicationId == "" {
		o.logger.WithContext(ctx).WithFields(log.Fields{
			"stacktrace": log.GetStackTrace(0),
		}).Warn("writing message to queue %s (which is configured to use content based deduplication) without message deduplication id", o.queue.GetName())
	}

	body, err := msg.MarshalToString()
	if err != nil {
		return nil, err
	}

	sqsMessage := &sqs.Message{
		DelaySeconds: delay,
		Body:         mdl.Box(body),
	}

	if messageGroupId != "" {
		sqsMessage.MessageGroupId = mdl.Box(messageGroupId)
	}

	if messageDeduplicationId != "" {
		sqsMessage.MessageDeduplicationId = mdl.Box(messageDeduplicationId)
	}

	return sqsMessage, nil
}
