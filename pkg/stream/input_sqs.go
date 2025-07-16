package stream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

var (
	_ AcknowledgeableInput = &sqsInput{}
	_ RetryingInput        = &sqsInput{}
)

type SqsInputSettings struct {
	cfg.AppId
	QueueId             string                     `cfg:"queue_id"`
	MaxNumberOfMessages int32                      `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32                      `cfg:"wait_time"`
	VisibilityTimeout   int                        `cfg:"visibility_timeout"`
	RunnerCount         int                        `cfg:"runner_count"`
	Fifo                sqs.FifoSettings           `cfg:"fifo"`
	RedrivePolicy       sqs.RedrivePolicy          `cfg:"redrive_policy"`
	ClientName          string                     `cfg:"client_name"`
	Unmarshaller        string                     `cfg:"unmarshaller" default:"msg"`
	Healthcheck         health.HealthCheckSettings `cfg:"healthcheck"`
}

func (s SqsInputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s SqsInputSettings) GetClientName() string {
	return s.ClientName
}

func (s SqsInputSettings) GetQueueId() string {
	return s.QueueId
}

func (s SqsInputSettings) IsFifoEnabled() bool {
	return s.Fifo.Enabled
}

type sqsInput struct {
	logger           log.Logger
	queue            sqs.Queue
	settings         *SqsInputSettings
	unmarshaler      UnmarshallerFunc
	healthCheckTimer clock.HealthCheckTimer

	cfn     coffin.Coffin
	channel chan *Message
	stopped int32
	started int32
}

func NewSqsInput(ctx context.Context, config cfg.Config, logger log.Logger, settings *SqsInputSettings) (*sqsInput, error) {
	settings.PadFromConfig(config)

	var ok bool
	var err error
	var queue sqs.Queue
	var queueName string
	var unmarshaller UnmarshallerFunc

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

	if unmarshaller, ok = unmarshallers[settings.Unmarshaller]; !ok {
		return nil, fmt.Errorf("unknown unmarshaller %s", settings.Unmarshaller)
	}

	healthCheckTimer, err := clock.NewHealthCheckTimer(settings.Healthcheck.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
	}

	return NewSqsInputWithInterfaces(logger, queue, unmarshaller, healthCheckTimer, settings), nil
}

func NewSqsInputWithInterfaces(
	logger log.Logger,
	queue sqs.Queue,
	unmarshaller UnmarshallerFunc,
	healthCheckTimer clock.HealthCheckTimer,
	settings *SqsInputSettings,
) *sqsInput {
	if settings.RunnerCount <= 0 {
		settings.RunnerCount = 1
	}

	return &sqsInput{
		logger:           logger,
		queue:            queue,
		settings:         settings,
		unmarshaler:      unmarshaller,
		healthCheckTimer: healthCheckTimer,
		cfn:              coffin.New(context.Background()),
		channel:          make(chan *Message),
	}
}

func (i *sqsInput) Data() <-chan *Message {
	return i.channel
}

func (i *sqsInput) Run(ctx context.Context) error {
	alreadyStarted := atomic.SwapInt32(&i.started, 1)
	if alreadyStarted == 1 {
		return fmt.Errorf("can not run an sqs input a second time")
	}

	defer close(i.channel)
	defer i.logger.Info("leaving sqs input")

	i.logger.Info("starting sqs input with %d runners", i.settings.RunnerCount)

	cfn := i.cfn.Entomb()
	for j := 0; j < i.settings.RunnerCount; j++ {
		i.cfn.Go(fmt.Sprintf("sqsInput/runLoop%03d", j), func() error {
			return i.runLoop(ctx)
		}, coffin.WithErrorWrapper("panic in sqs input runner"))
	}

	<-cfn.Dying()
	i.Stop()

	return i.cfn.Wait()
}

func (i *sqsInput) runLoop(ctx context.Context) error {
	defer i.logger.Info("leaving sqs input runner")

	for {
		if atomic.LoadInt32(&i.stopped) != 0 {
			return nil
		}

		// we are about to request some messages, so mark us as making progress (so far)
		i.healthCheckTimer.MarkHealthy()

		sqsMessages, err := i.queue.Receive(ctx, i.settings.MaxNumberOfMessages, i.settings.WaitTime)
		if err != nil {
			i.logger.Error("could not get messages from sqs: %w", err)

			continue
		}

		for _, sqsMessage := range sqsMessages {
			msg, err := i.unmarshaler(sqsMessage.Body)
			if err != nil {
				i.logger.Error("could not unmarshal message: %w", err)

				continue
			}

			if msg.Attributes == nil {
				msg.Attributes = make(map[string]string)
			}

			msg.Attributes[AttributeSqsMessageId] = *sqsMessage.MessageId
			msg.Attributes[AttributeSqsReceiptHandle] = *sqsMessage.ReceiptHandle

			if approximateReceiveCount, ok := sqsMessage.Attributes["ApproximateReceiveCount"]; ok {
				msg.Attributes[AttributeSqsApproximateReceiveCount] = approximateReceiveCount
			}

			i.channel <- msg

			// after every message we pushed to the channel, mark us as healthy as we made some progress
			// (even though the other side might be slow)
			i.healthCheckTimer.MarkHealthy()
		}
	}
}

func (i *sqsInput) Stop() {
	atomic.StoreInt32(&i.stopped, 1)
}

func (i *sqsInput) IsHealthy() bool {
	return i.healthCheckTimer.IsHealthy()
}

func (i *sqsInput) Ack(ctx context.Context, msg *Message, ack bool) error {
	if !ack {
		return nil
	}

	var ok bool
	var receiptHandleString string

	if receiptHandleString, ok = msg.Attributes[AttributeSqsReceiptHandle]; !ok {
		return fmt.Errorf("the message has no attribute %s", AttributeSqsReceiptHandle)
	}

	if receiptHandleString == "" {
		return fmt.Errorf("the attribute %s of the message should not be empty", AttributeSqsReceiptHandle)
	}

	return i.queue.DeleteMessage(ctx, receiptHandleString)
}

func (i *sqsInput) AckBatch(ctx context.Context, msgs []*Message, acks []bool) error {
	receiptHandles := make([]string, 0, len(msgs))
	multiError := new(multierror.Error)

	for i := range msgs {
		var (
			msg = msgs[i]
			ack = acks[i]
		)
		if !ack {
			continue
		}

		receiptHandleString, ok := msg.Attributes[AttributeSqsReceiptHandle]
		if !ok {
			multiError = multierror.Append(multiError, fmt.Errorf("the message has no attribute %s", AttributeSqsReceiptHandle))

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

	if err := i.queue.DeleteMessageBatch(ctx, receiptHandles); err != nil {
		multiError = multierror.Append(multiError, err)
	}

	return multiError.ErrorOrNil()
}

func (i *sqsInput) GetRetryHandler() (Input, RetryHandler) {
	retryHandler := NewManualSqsRetryHandler(i.logger, i.queue, &SqsOutputSettings{
		AppId:             i.settings.AppId,
		ClientName:        i.settings.ClientName,
		Fifo:              i.settings.Fifo,
		QueueId:           i.settings.QueueId,
		RedrivePolicy:     i.settings.RedrivePolicy,
		VisibilityTimeout: i.settings.VisibilityTimeout,
	})

	return NewNoopInput(), retryHandler
}

func (i *sqsInput) SetUnmarshaler(unmarshaler UnmarshallerFunc) {
	i.unmarshaler = unmarshaler
}

func (i *sqsInput) GetQueueUrl() string {
	return i.queue.GetUrl()
}

func (i *sqsInput) GetQueueArn() string {
	return i.queue.GetArn()
}
