package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/log"
)

type consumerAcknowledge struct {
	logger log.Logger
	input  Input
}

func newConsumerAcknowledgeWithInterfaces(logger log.Logger, input Input) consumerAcknowledge {
	return consumerAcknowledge{
		logger: logger,
		input:  input,
	}
}

func (c *consumerAcknowledge) Acknowledge(ctx context.Context, cdata *consumerData, ack bool) {
	var ok bool
	var ackInput AcknowledgeableInput

	if ackInput, ok = cdata.input.(AcknowledgeableInput); !ok {
		return
	}

	var messageId string
	messageId, ok = cdata.msg.Attributes[AttributeSqsMessageId]
	if ok {
		c.logger.WithFields(log.Fields{
			"sqs_message_id": messageId,
			"ack":            ack,
		}).Debug(ctx, "acknowledging sqs message")
	}

	if err := ackInput.Ack(ctx, cdata.msg, ack); err != nil {
		c.logger.Error(ctx, "could not acknowledge the message: %w", err)
	}
}

func (c *consumerAcknowledge) AcknowledgeBatch(ctx context.Context, cdata []*consumerData, acks []bool) {
	var ok bool
	var ackInput AcknowledgeableInput

	var (
		inputs    = make(map[string]AcknowledgeableInput)
		inputMsgs = make(map[string][]*Message)
		inputAcks = make(map[string][]bool)
	)

	for i := range cdata {
		var (
			data = cdata[i]
			ack  = acks[i]
		)

		if ackInput, ok = data.input.(AcknowledgeableInput); !ok {
			continue
		}

		if _, ok = inputs[data.src]; !ok {
			inputs[data.src] = ackInput
			inputMsgs[data.src] = make([]*Message, 0)
			inputAcks[data.src] = make([]bool, 0)
		}

		inputMsgs[data.src] = append(inputMsgs[data.src], data.msg)
		inputAcks[data.src] = append(inputAcks[data.src], ack)
	}

	for src, input := range inputs {
		if err := input.AckBatch(ctx, inputMsgs[src], inputAcks[src]); err != nil {
			c.logger.Error(ctx, "could not acknowledge the messages: %w", err)
		}
	}
}
