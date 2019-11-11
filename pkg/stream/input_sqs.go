package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/compression"
	"github.com/applike/gosoline/pkg/encoding/base64"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/hashicorp/go-multierror"
)

type SqsInputSettings struct {
	cfg.AppId
	QueueId           string            `cfg:"queue_id"`
	Fifo              sqs.FifoSettings  `cfg:"fifo"`
	WaitTime          int64             `cfg:"wait_time"`
	RedrivePolicy     sqs.RedrivePolicy `cfg:"redrive_policy"`
	VisibilityTimeout int               `cfg:"visibility_timeout"`
	RunnerCount       int               `cfg:"runner_count"`
}

type sqsInput struct {
	logger      mon.Logger
	queue       sqs.Queue
	settings    SqsInputSettings
	unmarshaler MessageUnmarshaler

	cfn     coffin.Coffin
	channel chan *Message
	stopped bool
}

func NewSqsInput(config cfg.Config, logger mon.Logger, s SqsInputSettings) *sqsInput {
	s.AppId.PadFromConfig(config)

	queue := sqs.New(config, logger, sqs.Settings{
		AppId:             s.AppId,
		QueueId:           s.QueueId,
		Fifo:              s.Fifo,
		RedrivePolicy:     s.RedrivePolicy,
		VisibilityTimeout: s.VisibilityTimeout,
	})

	return NewSqsInputWithInterfaces(logger, queue, s)
}

func NewSqsInputWithInterfaces(logger mon.Logger, queue sqs.Queue, s SqsInputSettings) *sqsInput {
	if s.RunnerCount <= 0 {
		s.RunnerCount = 1
	}

	return &sqsInput{
		logger:      logger,
		queue:       queue,
		settings:    s,
		unmarshaler: BasicUnmarshaler,
		cfn:         coffin.New(),
		channel:     make(chan *Message),
	}
}

func (i *sqsInput) Data() chan *Message {
	return i.channel
}

func (i *sqsInput) Run(ctx context.Context) error {
	defer close(i.channel)
	defer i.logger.Info("leaving sqs input")

	i.logger.Infof("starting sqs input with %d runners", i.settings.RunnerCount)

	for j := 0; j < i.settings.RunnerCount; j++ {
		i.cfn.Gof(func() error {
			return i.runLoop(ctx)
		}, "panic in sqs input runner")
	}

	<-i.cfn.Dying()
	i.Stop()

	return i.cfn.Wait()
}

func (i *sqsInput) runLoop(ctx context.Context) error {
	defer i.logger.Info("leaving sqs input runner")

	for {
		if i.stopped {
			return nil
		}

		sqsMessages, err := i.queue.Receive(ctx, i.settings.WaitTime)

		if err != nil {
			i.logger.Error(err, "could not get messages from sqs")
			continue
		}

		for _, sqsMessage := range sqsMessages {
			msg, err := i.unmarshaler(sqsMessage.Body)

			if err != nil {
				i.logger.Error(err, "could not unmarshal message")
				continue
			}

			if msg.Attributes == nil {
				msg.Attributes = make(map[string]interface{})
			}

			msg.Attributes[AttributeSqsReceiptHandle] = *sqsMessage.ReceiptHandle

			if msg.IsCompressed() {
				body, err := base64.DecodeString(msg.Body)
				if err != nil {
					return err
				}

				decompressedBody, err := compression.GunzipToString(body)
				if err != nil {
					return err
				}

				msg.Body = decompressedBody
			}

			i.channel <- msg
		}
	}
}

func (i *sqsInput) Stop() {
	i.stopped = true
}

func (i *sqsInput) Ack(msg *Message) error {
	var ok bool
	var receiptHandleInterface interface{}
	var receiptHandleString string

	if receiptHandleInterface, ok = msg.Attributes[AttributeSqsReceiptHandle]; !ok {
		return fmt.Errorf("the message has no attribute %s", AttributeSqsReceiptHandle)
	}

	if receiptHandleString, ok = receiptHandleInterface.(string); !ok {
		return fmt.Errorf("the attribute %s of the message should be string but instead is %T", AttributeSqsReceiptHandle, receiptHandleInterface)
	}

	if receiptHandleString == "" {
		return fmt.Errorf("the attribute %s of the message should not be empty", AttributeSqsReceiptHandle)
	}

	return i.queue.DeleteMessage(receiptHandleString)
}

func (i *sqsInput) AckBatch(msgs []*Message) error {
	receiptHandles := make([]string, 0, len(msgs))
	multiError := new(multierror.Error)

	for _, msg := range msgs {
		receiptHandleInterface, ok := msg.Attributes[AttributeSqsReceiptHandle]

		if !ok {
			multiError = multierror.Append(multiError, fmt.Errorf("the message has no attribute %s", AttributeSqsReceiptHandle))

			continue
		}

		receiptHandleString, ok := receiptHandleInterface.(string)

		if !ok {
			multiError = multierror.Append(multiError, fmt.Errorf("the attribute %s of the message should be string but instead is %T", AttributeSqsReceiptHandle, receiptHandleInterface))

			continue
		}

		if receiptHandleString == "" {
			multiError = multierror.Append(multiError, fmt.Errorf("the attribute %s of the message must not be empty", AttributeSqsReceiptHandle))

			continue
		}

		receiptHandles = append(receiptHandles, receiptHandleString)
	}

	if len(receiptHandles) == 0 {
		return multiError.ErrorOrNil()
	}

	if err := i.queue.DeleteMessageBatch(receiptHandles); err != nil {
		multiError = multierror.Append(multiError)
	}

	return multiError.ErrorOrNil()
}

func (i *sqsInput) SetUnmarshaler(unmarshaler MessageUnmarshaler) {
	i.unmarshaler = unmarshaler
}

func (i *sqsInput) GetQueueUrl() string {
	return i.queue.GetUrl()
}

func (i *sqsInput) GetQueueArn() string {
	return i.queue.GetArn()
}
