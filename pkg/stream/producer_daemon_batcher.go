package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/json"
)

type producerDaemonBatcher struct {
	maxMessages int
	maxBytes    int

	messages []WritableMessage
	size     int
}

//go:generate mockery -name ProducerDaemonBatcher
type ProducerDaemonBatcher interface {
	Append(msg *Message) ([]WritableMessage, error)
	Flush() []WritableMessage
}

type rawJsonMessage struct {
	attributes map[string]interface{}
	body       []byte
}

var _ json.Marshaler = rawJsonMessage{}

func (r rawJsonMessage) MarshalToBytes() ([]byte, error) {
	return r.body, nil
}

func (r rawJsonMessage) MarshalToString() (string, error) {
	return string(r.body), nil
}

func (r rawJsonMessage) MarshalJSON() ([]byte, error) {
	return r.body, nil
}

func (r rawJsonMessage) GetAttributes() map[string]interface{} {
	return r.attributes
}

func NewProducerDaemonBatcher(settings ProducerDaemonSettings) ProducerDaemonBatcher {
	return &producerDaemonBatcher{
		maxMessages: settings.BatchSize,
		maxBytes:    settings.BatchMaxSize,
		messages:    make([]WritableMessage, 0, settings.BatchSize),
		size:        0,
	}
}

func (b *producerDaemonBatcher) Append(msg *Message) ([]WritableMessage, error) {
	encodedMessage, err := json.Marshal(msg)

	if err != nil {
		return nil, fmt.Errorf("failed to encode message for batch: %w", err)
	}

	var result []WritableMessage = nil

	// if we can't fit this message in the batch, create a new one
	// subtract 1 so if we can fit it exactly so, we do that and flush after adding it
	if b.needsFlush(len(encodedMessage) - 1) {
		result = b.Flush()
	}

	b.messages = append(b.messages, rawJsonMessage{
		attributes: msg.Attributes,
		body:       encodedMessage,
	})
	b.size += len(encodedMessage)

	// if this batch is already full (we added a message exactly the max batch size), flush that too
	if b.needsFlush(0) {
		result = append(result, b.Flush()...)
	}

	return result, nil
}

func (b *producerDaemonBatcher) needsFlush(nextSize int) bool {
	newSize := b.size + nextSize

	return len(b.messages) >= b.maxMessages || (b.maxBytes != 0 && newSize >= b.maxBytes)
}

func (b *producerDaemonBatcher) Flush() []WritableMessage {
	result := b.messages
	b.messages = make([]WritableMessage, 0, b.maxMessages)
	b.size = 0

	return result
}
