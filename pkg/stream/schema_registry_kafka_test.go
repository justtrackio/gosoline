package stream_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/exec"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	schemaRegistryMocks "github.com/justtrackio/gosoline/pkg/kafka/schema-registry/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSchemaSettings_WithEncodingPreservesAutoRegister(t *testing.T) {
	settings := stream.SchemaSettings{AutoRegister: true}

	withEncoding := settings.WithEncoding(stream.EncodingJson)

	assert.True(t, withEncoding.AutoRegister)
	assert.Equal(t, stream.EncodingJson, withEncoding.Encoding)
}

func TestInitKafkaSchemaRegistry_UsesLookupOnlyByDefault(t *testing.T) {
	backoff := exec.BackoffSettings{}
	service := schemaRegistryMocks.NewService(t)
	settings := stream.SchemaSettingsWithEncoding{
		Subject:  "test-subject",
		Schema:   `{"type":"object"}`,
		Encoding: stream.EncodingJson,
		Model:    &struct{}{},
	}

	service.EXPECT().GetSubjectSchemaId(mock.Anything, settings.Subject, settings.Schema, schemaRegistry.Json).Return(11, nil).Once()

	encoder, err := stream.InitKafkaSchemaRegistry(t.Context(), log.NewLogger(), settings, backoff, service)

	assert.NoError(t, err)
	assert.NotNil(t, encoder)
}

func TestInitKafkaSchemaRegistry_UsesGetOrCreateWhenAutoRegisterEnabled(t *testing.T) {
	backoff := exec.BackoffSettings{}
	service := schemaRegistryMocks.NewService(t)
	settings := stream.SchemaSettingsWithEncoding{
		Subject:      "test-subject",
		Schema:       `{"type":"object"}`,
		Encoding:     stream.EncodingJson,
		AutoRegister: true,
		Model:        &struct{}{},
	}

	service.EXPECT().GetOrCreateSubjectSchemaId(mock.Anything, settings.Subject, settings.Schema, schemaRegistry.Json).Return(12, nil).Once()

	encoder, err := stream.InitKafkaSchemaRegistry(t.Context(), log.NewLogger(), settings, backoff, service)

	assert.NoError(t, err)
	assert.NotNil(t, encoder)
}
