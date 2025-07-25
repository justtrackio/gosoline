package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
)

// producerDaemonNoopAggregator creates a single message aggregate
type producerDaemonNoopAggregator struct {
	attributes map[string]string
}

func NewProducerDaemonNoopAggregator(attributeSets ...map[string]string) ProducerDaemonAggregator {
	aggregator := &producerDaemonNoopAggregator{
		attributes: map[string]string{
			AttributeEncoding: EncodingJson.String(),
		},
	}

	for _, attributes := range attributeSets {
		for k, v := range attributes {
			aggregator.attributes[k] = v
		}
	}

	return aggregator
}

func (a *producerDaemonNoopAggregator) Write(_ context.Context, msg *Message) ([]AggregateFlush, error) {
	encodedMessage, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message for aggregate: %w", err)
	}

	return []AggregateFlush{
		{
			Attributes:   a.attributes,
			MessageCount: 1,
			Body:         string(encodedMessage),
		},
	}, nil
}

func (a *producerDaemonNoopAggregator) Flush() ([]AggregateFlush, error) {
	// we already flush our single message aggregate on write
	return nil, nil
}
