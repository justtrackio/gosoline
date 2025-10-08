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

//go:generate go run github.com/vektra/mockery/v2 --name SchemaRegistryAwareOutput
type SchemaRegistryAwareOutput interface {
	Output
	// InitSchemaRegistry initializes the schema registry and returns the encoder/decoder corresponding to the schema
	InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error)
}

type OutputFactory func(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputCapabilities, error)

type OutputCapabilities struct {
	// IsPartitionedOutput should be true if the output is writing to more than one shard/partition/bucket, and we need to
	// take care about writing messages to the correct partition.
	IsPartitionedOutput bool
	// ProvidesCompression should be true if the Output natively handles compression.
	ProvidesCompression bool
	// SupportsAggregation should be false if the Output can not handle aggregated messages, e.g. when using a schema registry.
	SupportsAggregation bool
	// MaxBatchSize is the maximum number of messages we can write at once to the output (or nil if there is no limit).
	MaxBatchSize *int
	// MaxMessageSize is the maximum size of a message for this output (or nil if there is no limit on message size).
	MaxMessageSize *int
	// IgnoreProducerDaemonBatchSettings should be true if only the size restrictions specified on the output capabilities should be used.
	// Otherwise, they are only used if lower than the restrictions specified on the producer daemon batch settings.
	IgnoreProducerDaemonBatchSettings bool
}

var DefaultOutputCapabilities = &OutputCapabilities{
	IsPartitionedOutput:               false,
	ProvidesCompression:               false,
	SupportsAggregation:               true,
	MaxBatchSize:                      nil,
	MaxMessageSize:                    nil,
	IgnoreProducerDaemonBatchSettings: false,
}

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
