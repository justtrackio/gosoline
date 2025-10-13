package stream

import (
	"context"
)

// producerDaemonNoopAggregator creates a single message aggregate
type producerDaemonNoopAggregator struct{}

func NewProducerDaemonNoopAggregator() ProducerDaemonAggregator {
	return &producerDaemonNoopAggregator{}
}

func (a *producerDaemonNoopAggregator) Write(_ context.Context, msg *Message) ([]AggregateFlush, error) {
	// we do not json marshal the whole message here like we do in the non-noop aggregator.
	// instead we just forward the raw body and attributes of the message
	// because the body might have already been encoded by some external encoder and json marshaling it would then possibly break it
	return []AggregateFlush{
		{
			Attributes:   msg.Attributes,
			MessageCount: 1,
			Body:         msg.Body,
		},
	}, nil
}

func (a *producerDaemonNoopAggregator) Flush() ([]AggregateFlush, error) {
	// we already flush our single message aggregate on write
	return nil, nil
}
