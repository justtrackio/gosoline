package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/log"
)

type ConsumerAcknowledge struct {
	logger log.Logger
	input  Input
}

func NewConsumerAcknowledgeWithInterfaces(logger log.Logger, input Input) ConsumerAcknowledge {
	return ConsumerAcknowledge{
		logger: logger,
		input:  input,
	}
}

func (c *ConsumerAcknowledge) Acknowledge(ctx context.Context, cdata *consumerData) {
	var ok bool
	var ackInput AcknowledgeableInput

	if ackInput, ok = cdata.input.(AcknowledgeableInput); !ok {
		return
	}

	if err := ackInput.Ack(ctx, cdata.msg); err != nil {
		c.logger.WithContext(ctx).Error("could not acknowledge the message: %w", err)
	}
}

func (c *ConsumerAcknowledge) AcknowledgeBatch(ctx context.Context, cdata []*consumerData) {
	var ok bool
	var ackInput AcknowledgeableInput

	ackInputs := make(map[string]AcknowledgeableInput)
	ackMsgs := make(map[string][]*Message)

	for _, data := range cdata {
		if ackInput, ok = data.input.(AcknowledgeableInput); !ok {
			continue
		}

		if _, ok = ackInputs[data.src]; !ok {
			ackInputs[data.src] = ackInput
			ackMsgs[data.src] = make([]*Message, 0)
		}

		ackMsgs[data.src] = append(ackMsgs[data.src], data.msg)
	}

	for src, input := range ackInputs {
		if err := input.AckBatch(ctx, ackMsgs[src]); err != nil {
			c.logger.WithContext(ctx).Error("could not acknowledge the messages: %w", err)
		}
	}
}
