package log

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/spf13/cast"
)

const MessageAttributeLoggerContext = "logger:context"

// MessageWithLoggingFieldsEncoder encodes context fields into message attributes during message production
// and decodes them back into the context during message consumption, enabling context propagation across boundaries.
type MessageWithLoggingFieldsEncoder struct {
	logger Logger
}

// NewMessageWithLoggingFieldsEncoder creates a new MessageWithLoggingFieldsEncoder.
// It uses a SamplingLogger to prevent flooding logs with warnings if context fields cannot be encoded.
func NewMessageWithLoggingFieldsEncoder(_ cfg.Config, logger Logger) *MessageWithLoggingFieldsEncoder {
	logger = NewSamplingLogger(logger, time.Minute)

	return NewMessageWithLoggingFieldsEncoderWithInterfaces(logger)
}

// NewMessageWithLoggingFieldsEncoderWithInterfaces creates a new MessageWithLoggingFieldsEncoder with a specific logger.
func NewMessageWithLoggingFieldsEncoderWithInterfaces(logger Logger) *MessageWithLoggingFieldsEncoder {
	return &MessageWithLoggingFieldsEncoder{
		logger: logger,
	}
}

// Encode extracts global context fields, serializes them to JSON, and adds them to message attributes.
// This allows fields to travel with the message.
func (m MessageWithLoggingFieldsEncoder) Encode(ctx context.Context, _ any, attributes map[string]string) (context.Context, map[string]string, error) {
	fields := GlobalContextFieldsResolver(ctx)
	if len(fields) == 0 {
		return ctx, attributes, nil
	}

	stringAble := make(map[string]any, len(fields))
	for k, v := range fields {
		if _, err := cast.ToStringE(v); err != nil {
			m.logger.Warn(ctx, "omitting logger context field %s of type %T during message encoding", k, v)

			continue
		}
		stringAble[k] = v
	}

	encodedFields, err := json.Marshal(stringAble)
	if err != nil {
		m.logger.Warn(ctx, "can not json marshal logger context fields during message encoding")

		return ctx, attributes, nil
	}

	attributes[MessageAttributeLoggerContext] = string(encodedFields)

	return ctx, attributes, nil
}

// Decode retrieves context fields from message attributes, deserializes them, and appends them to the context.
func (m MessageWithLoggingFieldsEncoder) Decode(ctx context.Context, _ any, attributes map[string]string) (context.Context, map[string]string, error) {
	var ok bool
	if _, ok = attributes[MessageAttributeLoggerContext]; !ok {
		return ctx, attributes, nil
	}

	fields := make(map[string]any)
	err := json.Unmarshal([]byte(attributes["logger:context"]), &fields)
	if err != nil {
		m.logger.Warn(ctx, "can not json unmarshal logger context fields during message decoding")

		return ctx, attributes, nil
	}

	ctx = AppendGlobalContextFields(ctx, fields)
	delete(attributes, MessageAttributeLoggerContext)

	return ctx, attributes, nil
}
