package stream

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type consumerSchemaTestInput struct {
	Input
	settings SchemaSettingsWithEncoding
	err      error
}

func (i *consumerSchemaTestInput) InitSchemaRegistry(_ context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	i.settings = settings

	return NewJsonEncoder(), i.err
}

// Schema configuration need not implement a single-record processing callback.
type consumerSchemaTestCallback struct {
	settings *SchemaSettings
	err      error
}

func (c consumerSchemaTestCallback) GetSchemaSettings() (*SchemaSettings, error) {
	return c.settings, c.err
}

func TestConsumerEncoderUsesIndependentSchemaConfiguration(t *testing.T) {
	input := &consumerSchemaTestInput{}
	schema := &SchemaSettings{}
	encoder, err := newConsumerEncoder(t.Context(), input, consumerSchemaTestCallback{settings: schema}, EncodingAvro)
	require.NoError(t, err)
	require.Equal(t, schema.WithEncoding(EncodingAvro), input.settings)
	// Avro requires the registry-provided encoder; its JSON stand-in makes this
	// round trip verify that schema construction actually configured the codec.
	message, err := encoder.Encode(t.Context(), map[string]int{"value": 42})
	require.NoError(t, err)
	var model map[string]int
	_, _, err = encoder.Decode(t.Context(), message, &model)
	require.NoError(t, err)
	require.Equal(t, map[string]int{"value": 42}, model)
}

func TestConsumerEncoderPropagatesSchemaConfigurationErrors(t *testing.T) {
	expected := errors.New("schema unavailable")
	_, err := newConsumerEncoder(t.Context(), &consumerSchemaTestInput{}, consumerSchemaTestCallback{err: expected}, EncodingAvro)
	require.ErrorIs(t, err, expected)
	_, err = newConsumerEncoder(t.Context(), &consumerSchemaTestInput{err: expected}, consumerSchemaTestCallback{settings: &SchemaSettings{}}, EncodingAvro)
	require.ErrorIs(t, err, expected)
}
