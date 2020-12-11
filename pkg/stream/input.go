package stream

import "context"

//go:generate mockery -name Input
type Input interface {
	Run(ctx context.Context) error
	Stop()
	Data() chan *Message
}

//go:generate mockery -name AcknowledgeableInput
type AcknowledgeableInput interface {
	Input
	Ack(msg *Message) error
	AckBatch(msgs []*Message) error
}
