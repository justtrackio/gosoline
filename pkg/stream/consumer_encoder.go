package stream

import (
	"context"
	"fmt"
)

func newConsumerEncoder(ctx context.Context, input Input, schemaCallback SchemaSettingsAwareCallback, encoding EncodingType) (MessageEncoder, error) {
	var err error
	var schemaSettings *SchemaSettings
	var externalEncoder MessageBodyEncoder

	encoderSettings := &MessageEncoderSettings{Encoding: encoding}

	schemaRegistryAwareInput, isSchemaRegistryAwareInput := input.(SchemaRegistryAwareInput)
	if !isSchemaRegistryAwareInput || schemaCallback == nil {
		return NewMessageEncoder(encoderSettings), nil
	}

	if schemaSettings, err = schemaCallback.GetSchemaSettings(); err != nil {
		return nil, err
	}

	if schemaSettings == nil {
		return NewMessageEncoder(encoderSettings), nil
	}

	if externalEncoder, err = schemaRegistryAwareInput.InitSchemaRegistry(ctx, schemaSettings.WithEncoding(encoding)); err != nil {
		return nil, fmt.Errorf("failed to initialize schema registry: %w", err)
	}

	encoderSettings.ExternalEncoder = externalEncoder

	return NewMessageEncoder(encoderSettings), nil
}
