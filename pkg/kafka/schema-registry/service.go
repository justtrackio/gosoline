package schema_registry

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/twmb/franz-go/pkg/sr"
)

//go:generate go run github.com/vektra/mockery/v2 --name Service
type Service interface {
	GetSubjectSchemaId(ctx context.Context, subject string, schema string, schemaType SchemaType) (int, error)
}

type service struct {
	client Client
}

func NewService(config cfg.Config, connectionName string) (Service, error) {
	client, err := NewClient(config, connectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema registry client: %w", err)
	}

	return &service{
		client: client,
	}, nil
}

func (s service) GetSubjectSchemaId(ctx context.Context, subject string, schema string, schemaType SchemaType) (int, error) {
	var registrySchemaType sr.SchemaType

	switch schemaType {
	case Avro:
		registrySchemaType = sr.TypeAvro
	case Json:
		registrySchemaType = sr.TypeJSON
	case Protobuf:
		registrySchemaType = sr.TypeProtobuf
	default:
		return 0, fmt.Errorf("unknown schema type: %v", schemaType)
	}

	registrySchema := sr.Schema{
		Schema: schema,
		Type:   registrySchemaType,
	}

	subjectSchema, err := s.client.LookupSchema(ctx, subject, registrySchema)
	if err != nil {
		return 0, fmt.Errorf("failed to lookup subject schema: %w", err)
	}

	return subjectSchema.ID, nil
}
