package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type WritableMessage interface {
	MarshalToBytes() ([]byte, error)
	MarshalToString() (string, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name Output
type Output interface {
	WriteOne(ctx context.Context, msg WritableMessage) error
	Write(ctx context.Context, batch []WritableMessage) error
}

//go:generate go run github.com/vektra/mockery/v2 --name PartitionedOutput
type PartitionedOutput interface {
	Output
	// IsPartitionedOutput returns true if the output is writing to more than one shard/partition/bucket, and we need to
	// take care about writing messages to the correct partition.
	IsPartitionedOutput() bool
}

//go:generate go run github.com/vektra/mockery/v2 --name SizeRestrictedOutput
type SizeRestrictedOutput interface {
	Output
	// GetMaxMessageSize returns the maximum size of a message for this output (or nil if there is no limit on message size).
	GetMaxMessageSize() *int
	// GetMaxBatchSize returns the maximum number of messages we can write at once to the output (or nil if there is no limit).
	GetMaxBatchSize() *int
}

type OutputFactory func(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, error)

func MessagesToWritableMessages(batch []*Message) []WritableMessage {
	writableBatch := make([]WritableMessage, len(batch))

	for i, record := range batch {
		writableBatch[i] = record
	}

	return writableBatch
}

type hasAttributes interface {
	GetAttributes() map[string]string
}

// ensure all the types we actually write to SQS/SNS implement hasAttributes
var (
	_ hasAttributes = &Message{}
	_ hasAttributes = rawJsonMessage{}
)

func getAttributes(msg WritableMessage) map[string]string {
	if withAttributes, ok := msg.(hasAttributes); ok {
		return withAttributes.GetAttributes()
	}

	return map[string]string{}
}
