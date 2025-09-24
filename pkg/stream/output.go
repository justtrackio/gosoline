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

type OutputFactory func(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputSettings, error)

type OutputSettings struct {
	IsPartitionedOutput               bool
	ProvidesCompression               bool
	SupportsAggregation               bool
	MaxBatchSize                      *int
	MaxMessageSize                    *int
	IgnoreProducerDaemonBatchSettings bool
}

var DefaultOutputSettings = &OutputSettings{
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
