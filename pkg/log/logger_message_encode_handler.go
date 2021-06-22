package log

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/spf13/cast"
	"time"
)

const MessageAttributeLoggerContext = "logger:context"

type MessageWithLoggingFieldsEncoder struct {
	logger Logger
}

func NewMessageWithLoggingFieldsEncoder(_ cfg.Config, logger Logger) *MessageWithLoggingFieldsEncoder {
	logger = NewSamplingLogger(logger, time.Minute)

	return NewMessageWithLoggingFieldsEncoderWithInterfaces(logger)
}

func NewMessageWithLoggingFieldsEncoderWithInterfaces(logger Logger) *MessageWithLoggingFieldsEncoder {
	return &MessageWithLoggingFieldsEncoder{
		logger: logger,
	}
}

func (m MessageWithLoggingFieldsEncoder) Encode(ctx context.Context, _ interface{}, attributes map[string]interface{}) (context.Context, map[string]interface{}, error) {
	fields := ContextLoggerFieldsResolver(ctx)

	if len(fields) == 0 {
		return ctx, attributes, nil
	}

	stringAble := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		if _, err := cast.ToStringE(v); err != nil {
			m.logger.Warn("omitting logger context field %s of type %T during message encoding", k, v)
			continue
		}

		stringAble[k] = v
	}

	encodedFields, err := json.Marshal(stringAble)

	if err != nil {
		m.logger.Warn("can not json marshal logger context fields during message encoding")
		return ctx, attributes, nil
	}

	attributes[MessageAttributeLoggerContext] = string(encodedFields)

	return ctx, attributes, nil
}

func (m MessageWithLoggingFieldsEncoder) Decode(ctx context.Context, _ interface{}, attributes map[string]interface{}) (context.Context, map[string]interface{}, error) {
	var str string
	var ok bool

	if _, ok = attributes[MessageAttributeLoggerContext]; !ok {
		return ctx, attributes, nil
	}

	if str, ok = attributes["logger:context"].(string); !ok {
		m.logger.Warn("encoded logger context fields should be of type string but instead are of type %T", attributes["logger:context"])
		return ctx, attributes, nil
	}

	fields := make(map[string]interface{})
	err := json.Unmarshal([]byte(str), &fields)

	if err != nil {
		m.logger.Warn("can not json unmarshal logger context fields during message decoding")
		return ctx, attributes, nil
	}

	ctx = AppendLoggerContextField(ctx, fields)
	delete(attributes, MessageAttributeLoggerContext)

	return ctx, attributes, nil
}
