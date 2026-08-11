package stream

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

var (
	_ Input         = &sqsInput{}
	_ RetryingInput = &sqsInput{}
)

type SqsAcknowledgementMode string

const (
	SqsAcknowledgementModeIndividual SqsAcknowledgementMode = "individual"
	SqsAcknowledgementModeBatch      SqsAcknowledgementMode = "batch"
)

// SqsInputSettings configures an SQS input. Messages are delivered at least once: acknowledgements use the
// configured AWS client retries, and a message can be delivered again if deletion still fails. The input continues
// consuming after such a failure because stopping it cannot repair the acknowledgement or prevent redelivery.
type SqsInputSettings struct {
	Identity            cfg.Identity `cfg:"identity"`
	QueueId             string       `cfg:"queue_id"`
	MaxNumberOfMessages int32        `cfg:"max_number_of_messages" default:"10" validate:"min=1,max=10"`
	WaitTime            int32        `cfg:"wait_time"`
	VisibilityTimeout   int          `cfg:"visibility_timeout"`
	// GraceTime bounds acknowledging already processed messages after the input context was canceled. It does not
	// bound processing itself, which is owned by stream.consumer.<name>.grace_time.
	GraceTime           time.Duration              `cfg:"grace_time" default:"10s"`
	RunnerCount         int                        `cfg:"runner_count"`
	AcknowledgementMode SqsAcknowledgementMode     `cfg:"acknowledgement_mode" default:"individual" validate:"oneof=individual batch"`
	Fifo                sqs.FifoSettings           `cfg:"fifo"`
	RedrivePolicy       sqs.RedrivePolicy          `cfg:"redrive_policy"`
	ClientName          string                     `cfg:"client_name"`
	Unmarshaller        string                     `cfg:"unmarshaller" default:"msg"`
	Healthcheck         health.HealthCheckSettings `cfg:"healthcheck"`
}

func (s SqsInputSettings) GetIdentity() cfg.Identity {
	return s.Identity
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
	stopped int32
	started int32
}

func NewSqsInput(ctx context.Context, config cfg.Config, logger log.Logger, settings *SqsInputSettings) (*sqsInput, error) {
	var err error
	var ok bool
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
	return &sqsInput{
		logger:           logger,
		queue:            queue,
		settings:         settings,
		unmarshaler:      unmarshaller,
		healthCheckTimer: healthCheckTimer,
		cfn:              coffin.New(),
	}
}

func (i *sqsInput) Run(ctx context.Context, process InputProcess) error {
	alreadyStarted := atomic.SwapInt32(&i.started, 1)
	if alreadyStarted == 1 {
		return fmt.Errorf("can not run an sqs input a second time")
	}

	defer i.logger.Info(ctx, "leaving sqs input")

	i.logger.Info(ctx, "starting sqs input with %d runners", i.settings.RunnerCount)

	i.cfn.Go(func() error {
		for j := 0; j < i.settings.RunnerCount; j++ {
			i.cfn.Gof(func() error {
				return i.runLoop(ctx, process)
			}, "panic in sqs input runner")
		}

		return nil
	})

	<-i.cfn.Dying()
	i.Stop(ctx)

	return i.cfn.Wait()
}

func (i *sqsInput) runLoop(ctx context.Context, process InputProcess) error {
	defer i.logger.Info(ctx, "leaving sqs input runner")

	for {
		if atomic.LoadInt32(&i.stopped) != 0 {
			return nil
		}

		// we are about to request some messages, so mark us as making progress (so far)
		i.healthCheckTimer.MarkHealthy()

		sqsMessages, err := i.queue.Receive(ctx, i.settings.MaxNumberOfMessages, i.settings.WaitTime)
		if exec.IsRequestCanceled(err) || ctx.Err() != nil {
			return nil
		}

		if err != nil {
			i.logger.Error(ctx, "could not get messages from sqs: %w", err)

			continue
		}

		i.processReceivedMessages(ctx, sqsMessages, process)
	}
}

func (i *sqsInput) processReceivedMessages(ctx context.Context, sqsMessages []types.Message, process InputProcess) {
	if i.settings.AcknowledgementMode == SqsAcknowledgementModeBatch {
		i.processMessageBatch(ctx, sqsMessages, process)

		return
	}

	for _, sqsMessage := range sqsMessages {
		if receiptHandle, ok := i.processMessage(ctx, sqsMessage, process); ok {
			if err := i.ack(ctx, receiptHandle); err != nil {
				// The AWS client has already exhausted its configured retries. Stopping the input would not repair the
				// acknowledgement or prevent redelivery, so preserve throughput and rely on at-least-once semantics.
				i.logger.Error(ctx, "could not acknowledge sqs message, message may be delivered again: %w", err)
			}
		}
	}
}

func (i *sqsInput) processMessageBatch(ctx context.Context, sqsMessages []types.Message, process InputProcess) {
	receiptHandles := make([]string, 0, len(sqsMessages))

	for _, sqsMessage := range sqsMessages {
		if receiptHandle, ok := i.processMessage(ctx, sqsMessage, process); ok {
			receiptHandles = append(receiptHandles, receiptHandle)
		}
	}

	if len(receiptHandles) == 0 {
		return
	}

	if err := i.ackBatch(ctx, receiptHandles); err != nil {
		i.logger.Error(ctx, "could not acknowledge sqs message batch, messages may be delivered again: %w", err)
	}
}

func (i *sqsInput) processMessage(ctx context.Context, sqsMessage types.Message, process InputProcess) (string, bool) {
	if sqsMessage.MessageId == nil || *sqsMessage.MessageId == "" {
		i.logger.Error(ctx, "could not process sqs message: message id is missing")

		return "", false
	}

	if sqsMessage.ReceiptHandle == nil || *sqsMessage.ReceiptHandle == "" {
		i.logger.Error(ctx, "could not process sqs message: receipt handle is missing")

		return "", false
	}

	msg, err := i.unmarshaler(sqsMessage.Body)
	if err != nil {
		i.logger.Error(ctx, "could not unmarshal message: %w", err)

		return "", false
	}

	if msg.Attributes == nil {
		msg.Attributes = make(map[string]string)
	}

	msg.Attributes[AttributeSqsMessageId] = *sqsMessage.MessageId
	msg.Attributes[AttributeSqsReceiptHandle] = *sqsMessage.ReceiptHandle

	if approximateReceiveCount, ok := sqsMessage.Attributes["ApproximateReceiveCount"]; ok {
		msg.Attributes[AttributeSqsApproximateReceiveCount] = approximateReceiveCount
	}

	ack := process(ctx, msg)

	// after every message we pushed to the channel, mark us as healthy as we made some progress
	// (even though the other side might be slow)
	i.healthCheckTimer.MarkHealthy()

	if !ack {
		return "", false
	}

	return *sqsMessage.ReceiptHandle, true
}

func (i *sqsInput) Stop(ctx context.Context) {
	atomic.StoreInt32(&i.stopped, 1)
}

func (i *sqsInput) IsHealthy() bool {
	return i.healthCheckTimer.IsHealthy()
}

func (i *sqsInput) ack(ctx context.Context, receiptHandle string) error {
	return i.acknowledge(ctx, func(ctx context.Context) error {
		return i.queue.DeleteMessage(ctx, receiptHandle)
	})
}

func (i *sqsInput) ackBatch(ctx context.Context, receiptHandles []string) error {
	return i.acknowledge(ctx, func(ctx context.Context) error {
		return i.queue.DeleteMessageBatch(ctx, receiptHandles)
	})
}

func (i *sqsInput) acknowledge(ctx context.Context, acknowledge func(context.Context) error) error {
	delayedCtx, stop := exec.WithDelayedCancelContext(ctx, i.settings.GraceTime)
	defer stop()

	err := acknowledge(delayedCtx)
	if err == nil {
		return nil
	}

	if exec.IsRequestCanceled(err) || ctx.Err() != nil {
		i.logger.Warn(ctx, "could not acknowledge the message during shutdown: %w", err)

		return nil
	}

	return err
}

func (i *sqsInput) GetRetryHandler() (Input, RetryHandler) {
	retryHandler := NewManualSqsRetryHandler(i.logger, i.queue, &SqsOutputSettings{
		Identity:          i.settings.Identity,
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
