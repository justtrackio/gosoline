package stream

import (
	"context"
	"fmt"
	"strconv"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/log"
)

type manualSqsRetryHandler struct {
	output Output
}

func NewManualSqsRetryHandler(logger log.Logger, queue sqs.Queue, settings *SqsOutputSettings) RetryHandler {
	return NewManualSqsRetryHandlerFromInterfaces(NewSqsOutputWithInterfaces(logger, queue, settings))
}

func NewManualSqsRetryHandlerFromInterfaces(output Output) RetryHandler {
	return manualSqsRetryHandler{
		output: output,
	}
}

func (r manualSqsRetryHandler) Put(ctx context.Context, msg *Message) error {
	// do not put it back into retry if it's already in there
	// sqs will redeliver the message automatically
	if _, ok := msg.Attributes[attributeRetrySqs]; ok {
		return nil
	}

	msg.Attributes[attributeRetrySqs] = strconv.FormatBool(true)

	if err := r.output.WriteOne(ctx, msg); err != nil {
		return fmt.Errorf("can not write the message to the output: %w", err)
	}

	return nil
}
