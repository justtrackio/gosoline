package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
)

const sqsOutputBatchSize = 10

type SqsOutputSettings struct {
	cfg.AppId
	QueueId           string
	VisibilityTimeout int
	Fifo              sqs.FifoSettings
	RedrivePolicy     sqs.RedrivePolicy
	Client            cloud.ClientSettings
	Backoff           cloud.BackoffSettings
}

type sqsOutput struct {
	logger   mon.Logger
	tracer   tracing.Tracer
	queue    sqs.Queue
	settings SqsOutputSettings
}

func NewSqsOutput(config cfg.Config, logger mon.Logger, s SqsOutputSettings) Output {
	s.PadFromConfig(config)

	queue := sqs.New(config, logger, &sqs.Settings{
		AppId:             s.AppId,
		QueueId:           s.QueueId,
		VisibilityTimeout: s.VisibilityTimeout,
		Fifo:              s.Fifo,
		RedrivePolicy:     s.RedrivePolicy,
		Client:            s.Client,
		Backoff:           s.Backoff,
	})

	tracer := tracing.ProviderTracer(config, logger)

	return NewSqsOutputWithInterfaces(logger, tracer, queue, s)
}

func NewSqsOutputWithInterfaces(logger mon.Logger, tracer tracing.Tracer, queue sqs.Queue, s SqsOutputSettings) Output {
	return &sqsOutput{
		logger:   logger,
		tracer:   tracer,
		queue:    queue,
		settings: s,
	}
}

func (o *sqsOutput) WriteOne(ctx context.Context, record *Message) error {
	return o.Write(ctx, []*Message{record})
}

func (o *sqsOutput) Write(ctx context.Context, batch []*Message) error {
	spanName := fmt.Sprintf("sqs-output-%v-%v-%v", o.settings.Family, o.settings.Application, o.settings.QueueId)

	ctx, trans := o.tracer.StartSubSpan(ctx, spanName)
	defer trans.Finish()

	return o.sendToQueue(ctx, batch)
}

func (o *sqsOutput) sendToQueue(ctx context.Context, batch []*Message) error {
	chunks, ok := funk.Chunk(batch, sqsOutputBatchSize).([][]*Message)

	if !ok {
		err := fmt.Errorf("can not chunk messages for sending to sqs")
		o.logger.Error(err, "can not chunk messages for sending to sqs")

		return err
	}

	var result error

	for _, chunk := range chunks {
		messages, err := o.buildSqsMessages(chunk)

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
		return errors.Wrap(result, "there were errors on writing to the sqs stream")
	}

	return nil
}

func (o *sqsOutput) buildSqsMessages(messages []*Message) ([]*sqs.Message, error) {
	var result error
	sqsMessages := make([]*sqs.Message, 0)

	for _, msg := range messages {
		sqsMessage, err := o.buildSqsMessage(msg)

		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

		sqsMessages = append(sqsMessages, sqsMessage)
	}

	return sqsMessages, result
}

func (o *sqsOutput) buildSqsMessage(msg *Message) (*sqs.Message, error) {
	var delay *int64
	var messageGroupId *string

	if d, ok := msg.Attributes[AttributeSqsDelaySeconds]; ok {
		if dInt64, ok := d.(int64); ok {
			delay = mdl.Int64(dInt64)
		} else {
			return nil, fmt.Errorf("the type of the %s attribute should be int64 but instead is %T", AttributeSqsDelaySeconds, d)
		}
	}

	if d, ok := msg.Attributes[AttributeSqsMessageGroupId]; ok {
		if groupIdString, ok := d.(string); ok {
			messageGroupId = mdl.String(groupIdString)
		} else {
			return nil, fmt.Errorf("the type of the %s attribute should be int64 but instead is %T", AttributeSqsDelaySeconds, d)
		}
	}

	body, err := msg.MarshalToString()

	if err != nil {
		return nil, err
	}

	sqsMessage := &sqs.Message{
		DelaySeconds:   delay,
		MessageGroupId: messageGroupId,
		Body:           mdl.String(body),
	}

	return sqsMessage, nil
}
