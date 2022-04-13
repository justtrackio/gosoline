package stream

import "context"

// An Input provides you with a steady stream of messages until you Stop it.
//go:generate mockery --name Input
type Input interface {
	// Run provides a steady stream of messages, returned via Data. Run does not return until Stop is called and thus
	// should be called in its own go routine. The only exception to this is if we either fail to produce messages and
	// return an error or if the input is depleted (like an InMemoryInput).
	//
	// Run should only be called once, not all inputs can be resumed.
	Run(ctx context.Context) error
	// Stop causes Run to return as fast as possible. Calling Stop is preferable to canceling the context passed to Run
	// as it allows Run to shut down cleaner (and might take a bit longer, e.g., to finish processing the current batch
	// of messages).
	Stop()
	// Data returns a channel containing the messages produced by this input.
	Data() <-chan *Message
}

// An AcknowledgeableInput is an Input with the additional ability to mark messages as successfully consumed. For example,
// an SQS queue would provide a message after its visibility timeout a second time if we didn't acknowledge it.
//go:generate mockery --name AcknowledgeableInput
type AcknowledgeableInput interface {
	Input
	// Ack acknowledges a single message. If possible, prefer calling AckBatch as it is more efficient.
	Ack(ctx context.Context, msg *Message, ack bool) error
	// AckBatch does the same as calling Ack for every single message would, but it might use fewer calls to an external
	// service.
	AckBatch(ctx context.Context, msgs []*Message, acks []bool) error
}

type RetryableInput interface {
	HasRetry() bool
}
