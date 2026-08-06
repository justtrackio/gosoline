package stream

import (
	"fmt"
	"sync/atomic"

	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	AttributeKafkaKey            = "KafkaKey"
	MetaDataKafkaOriginalMessage = "KafkaOriginal"

	// metaDataKafkaBatchCompletion carries the kafkaBatchCompletion of the batch a message was part of. It is not
	// exported as it is purely internal plumbing between the kafka input and the consumer.
	metaDataKafkaBatchCompletion = "KafkaBatchCompletion"
)

type KafkaSourceMessage struct {
	kgo.Record
}

func NewKafkaMessageAttrs(key string) map[string]any {
	return map[string]any{AttributeKafkaKey: key}
}

func KafkaHeadersToGosoAttributes(kafkaRecordHeaders []kgo.RecordHeader) map[string]string {
	attributes := make(map[string]string)

	for _, v := range kafkaRecordHeaders {
		attributes[v.Key] = string(v.Value)
	}

	return attributes
}

func KafkaToGosoMessage(kafkaRecord kgo.Record) *Message {
	attributes := KafkaHeadersToGosoAttributes(kafkaRecord.Headers)
	metaData := map[string]any{
		MetaDataKafkaOriginalMessage: KafkaSourceMessage{Record: kafkaRecord},
	}

	return &Message{Body: string(kafkaRecord.Value), Attributes: attributes, metaData: metaData}
}

func NewKafkaMessage(message WritableMessage) (*kgo.Record, error) {
	kafkaRecord := &kgo.Record{}
	var body []byte
	var attributes map[string]string

	// if the message comes from the producer daemon it's a rawJsonMessage that only holds the encoded model in the body
	// otherwise, it's a *Message
	switch m := message.(type) {
	case *Message:
		body = []byte(m.Body)
		attributes = m.Attributes
	case rawJsonMessage:
		body = m.body
		attributes = m.attributes
	default:
		return nil, fmt.Errorf("unexpected message type: %T", m)
	}

	kafkaRecord.Value = body

	key, ok := attributes[AttributeKafkaKey]
	if ok {
		kafkaRecord.Key = []byte(key)
	}

	for k, v := range attributes {
		if k == AttributeKafkaKey {
			continue
		}

		kafkaRecord.Headers = append(
			kafkaRecord.Headers,
			kgo.RecordHeader{Key: k, Value: []byte(v)},
		)
	}

	return kafkaRecord, nil
}

func NewKafkaMessages(messages []WritableMessage) ([]*kgo.Record, error) {
	var err error
	out := make([]*kgo.Record, len(messages))

	for i, message := range messages {
		if out[i], err = NewKafkaMessage(message); err != nil {
			return nil, fmt.Errorf("can not build kafka message: %w", err)
		}
	}

	return out, nil
}

type kafkaMessageHandler struct {
	data chan *Message
}

func NewKafkaMessageHandler(data chan *Message) kafkaConsumer.KafkaMessageHandler {
	return &kafkaMessageHandler{
		data: data,
	}
}

func (h *kafkaMessageHandler) Handle(kafkaRecords []*kgo.Record) kafkaConsumer.BatchCompletion {
	messages := make([]*Message, 0, len(kafkaRecords))

	for _, record := range kafkaRecords {
		if record == nil {
			continue
		}

		messages = append(messages, KafkaToGosoMessage(*record))
	}

	// the completion has to know the final number of records before we hand out the first message, otherwise it could
	// already report the batch as done while we are still pushing messages into the channel
	completion := newKafkaBatchCompletion(len(messages))

	for _, msg := range messages {
		msg.metaData[metaDataKafkaBatchCompletion] = completion
		h.data <- msg
	}

	return completion
}

func (h *kafkaMessageHandler) Stop() {
	close(h.data)
}

// kafkaBatchCompletion counts down the records of a single batch until all of them have been processed.
type kafkaBatchCompletion struct {
	pending atomic.Int64
	failed  atomic.Int64
	done    chan struct{}
}

func newKafkaBatchCompletion(count int) *kafkaBatchCompletion {
	completion := &kafkaBatchCompletion{
		done: make(chan struct{}),
	}
	completion.pending.Store(int64(count))

	if count == 0 {
		close(completion.done)
	}

	return completion
}

// Done implements kafkaConsumer.BatchCompletion.
func (c *kafkaBatchCompletion) Done() <-chan struct{} {
	return c.done
}

// FailedCount implements kafkaConsumer.BatchCompletion.
func (c *kafkaBatchCompletion) FailedCount() int {
	return int(c.failed.Load())
}

// complete marks a single record of the batch as processed. Should it ever be called more often than there are records
// in the batch, the counter simply drops below zero and we never close the channel a second time.
func (c *kafkaBatchCompletion) complete(success bool) {
	if !success {
		c.failed.Add(1)
	}

	if c.pending.Add(-1) == 0 {
		close(c.done)
	}
}
