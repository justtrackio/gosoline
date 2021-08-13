package stream

import "context"

//go:generate mockery --name Input
type Input interface {
	Run(ctx context.Context) error
	Stop()
	Data() chan *Message
}

//go:generate mockery --name AcknowledgeableInput
type AcknowledgeableInput interface {
	Ack(ctx context.Context, msg *Message) error
	AckBatch(ctx context.Context, msgs []*Message) error
}
