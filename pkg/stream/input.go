package stream

import (
	"context"
)

type InputProcess func(ctx context.Context, msg *Message) (ack bool)

// An Input provides you with a steady stream of messages until you Stop it.
//
//go:generate go run github.com/vektra/mockery/v2 --name Input
type Input interface {
	// Run provides a steady stream of messages, returned via Data. Run does not return until Stop is called and thus
	// should be called in its own go routine. The only exception to this is if we either fail to produce messages and
	// return an error or if the input is depleted (like an InMemoryInput).
	//
	// Run should only be called once, not all inputs can be resumed.
	Run(ctx context.Context, process InputProcess) error
	// Stop prevents subsequent fetches and causes Run to return as fast as possible. Messages already fetched when Stop
	// is called may still be passed to process, allowing the consumer to finish them during its shutdown grace period.
	// Calling Stop is preferable to canceling the context passed to Run as it allows Run to shut down cleaner (and might
	// take a bit longer, e.g., to finish processing the current batch of messages).
	Stop(ctx context.Context)
	// IsHealthy checks if the input is still able to produce data. An Input is healthy if it produces zero or more
	// messages repeatedly. Producing zero messages would for example happen if the input requested data from an
	// external queue, but the queue was empty. An Input is unhealthy if it is no longer able to produce any messages.
	//
	// If an input exhausts its source (file, finished stream, fixed list, ...), it is still considered as healthy.
	IsHealthy() bool
}

//go:generate go run github.com/vektra/mockery/v2 --name SchemaRegistryAwareInput
type SchemaRegistryAwareInput interface {
	Input
	// InitSchemaRegistry initializes the schema registry and returns the encoder/decoder corresponding to the schema
	InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error)
}

type RetryingInput interface {
	GetRetryHandler() (Input, RetryHandler)
}
