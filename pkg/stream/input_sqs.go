package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
)

type SqsInputSettings struct {
	cfg.AppId
	QueueId           string            `mapstructure:"queue_id"`
	Fifo              sqs.FifoSettings  `mapstructure:"fifo"`
	WaitTime          int64             `mapstructure:"wait_time"`
	RedrivePolicy     sqs.RedrivePolicy `mapstructure:"redrive_policy"`
	VisibilityTimeout int               `mapstructure:"visibility_timeout"`
}

type sqsInput struct {
	logger      mon.Logger
	queue       sqs.Queue
	settings    SqsInputSettings
	unmarshaler MessageUnmarshaler

	channel chan *Message
	stopped bool
}

func NewSqsInput(config cfg.Config, logger mon.Logger, s SqsInputSettings) *sqsInput {
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
	return &sqsInput{
		logger:      logger,
		queue:       queue,
		settings:    s,
		unmarshaler: BasicUnmarshaler,
		channel:     make(chan *Message),
	}
}

func (i *sqsInput) Data() chan *Message {
	return i.channel
}

func (i *sqsInput) Run() error {
	defer close(i.channel)
	defer i.logger.Info("leaving sqs input")

	for {
		if i.stopped {
			return nil
		}

		sqsMessages, err := i.queue.Receive(i.settings.WaitTime)

		if err != nil {
			i.logger.Error(err, "could not get messages from sqs")
			i.stopped = true
			return err
		}

		for _, sqsMessage := range sqsMessages {
			msg, err := i.unmarshaler(sqsMessage.Body)

			if err != nil {
				i.logger.Error(err, "could not unmarshal message")
				continue
			}

			msg.Attributes[AttributeSqsReceiptHandle] = *sqsMessage.ReceiptHandle

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

	return i.queue.DeleteMessage(receiptHandleString)
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
