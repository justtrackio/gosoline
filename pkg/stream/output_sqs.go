package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/applike/gosoline/pkg/tracing"
)

const sqsOutputBatchSize = 10

type SqsOutputSettings struct {
	cfg.AppId
	QueueId string
}

type sqsOutput struct {
	logger   mon.Logger
	tracer   tracing.Tracer
	queue    sqs.Queue
	settings SqsOutputSettings
}

func NewSqsOutput(config cfg.Config, logger mon.Logger, s SqsOutputSettings) Output {
	s.PadFromConfig(config)

	queue := sqs.New(config, logger, sqs.Settings{
		AppId:   s.AppId,
		QueueId: s.QueueId,
	})

	tracer := tracing.NewAwsTracer(config)

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

	ctx, trans := o.tracer.StartSpanFromContext(ctx, spanName)
	defer trans.Finish()

	for _, msg := range batch {
		msg.Trace = trans.GetTrace()
	}

	return o.sendToQueue(ctx, batch)
}

func (o *sqsOutput) sendToQueue(ctx context.Context, batch []*Message) error {
	chunks, err := BuildChunks(batch, sqsOutputBatchSize)

	if err != nil {
		o.logger.Error(err, "could not batch all messages")
	}

	errors := make([]error, 0)

	for _, chunk := range chunks {
		strings := ByteChunkToStrings(chunk)
		err = o.queue.SendBatch(ctx, strings)

		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("there were %v errors on writing to the sqs stream", len(errors))
	}

	return nil
}
