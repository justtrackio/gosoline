package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/funk"
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

func (c *ConsumerAcknowledge) Acknowledge(ctx context.Context, cdata *consumerData, ack bool) {
	var ok bool
	var ackInput AcknowledgeableInput

	if ackInput, ok = cdata.input.(AcknowledgeableInput); !ok {
		return
	}

	msg := &cdata.msg
	if cdata.originalMessage != nil {
		msg = &cdata.originalMessage.Message
	}

	if err := ackInput.Ack(ctx, msg, ack); err != nil {
		c.logger.WithContext(ctx).Error("could not acknowledge the message: %w", err)
	}
}

func (c *ConsumerAcknowledge) AcknowledgeBatch(ctx context.Context, cdata []*consumerData, acks []bool) {
	var ok bool
	var ackInput AcknowledgeableInput

	inputs := make(map[string]AcknowledgeableInput)
	inputMsgs := make(map[string][]*Message)
	inputAcks := make(map[string][]bool)
	seenMessageIds := funk.NewSet[string]()

	for i := range cdata {
		data := cdata[i]
		ack := acks[i]

		if ackInput, ok = data.input.(AcknowledgeableInput); !ok {
			continue
		}

		if _, ok = inputs[data.src]; !ok {
			inputs[data.src] = ackInput
			inputMsgs[data.src] = make([]*Message, 0)
			inputAcks[data.src] = make([]bool, 0)
		}

		if data.originalMessage == nil {
			inputMsgs[data.src] = append(inputMsgs[data.src], &data.msg)
			inputAcks[data.src] = append(inputAcks[data.src], ack)
		} else {
			if seenMessageIds.Contains(data.originalMessage.id) {
				continue
			}

			seenMessageIds.Set(data.originalMessage.id)
			inputMsgs[data.src] = append(inputMsgs[data.src], &data.originalMessage.Message)
			inputAcks[data.src] = append(inputAcks[data.src], ack)
		}

	}

	for src, input := range inputs {
		if err := input.AckBatch(ctx, inputMsgs[src], inputAcks[src]); err != nil {
			c.logger.WithContext(ctx).Error("could not acknowledge the messages: %w", err)
		}
	}
}

// TODO: tests
