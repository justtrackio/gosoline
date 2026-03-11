package schema_registry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/twmb/franz-go/pkg/sr"
)

//go:generate go run github.com/vektra/mockery/v2 --name Service
type Service interface {
	GetSubjectSchemaId(ctx context.Context, subject string, schema string, schemaType SchemaType) (int, error)
	GetOrCreateSubjectSchemaId(ctx context.Context, subject string, schema string, schemaType SchemaType) (int, error)
}

type service struct {
	client Client
}

func NewService(connection connection.Settings) (Service, error) {
	client, err := NewClient(connection.SchemaRegistryAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema registry client: %w", err)
	}

	return NewServiceWithInterfaces(client), nil
}

func NewServiceWithInterfaces(client Client) Service {
	return &service{
		client: client,
	}
}

func (s service) GetSubjectSchemaId(ctx context.Context, subject string, schema string, schemaType SchemaType) (int, error) {
	registrySchema, err := buildRegistrySchema(schema, schemaType)
	if err != nil {
		return 0, err
	}

	subjectSchema, err := s.client.LookupSchema(ctx, subject, registrySchema)
	if err != nil {
		return 0, fmt.Errorf("failed to lookup subject schema: %w", err)
	}

	return subjectSchema.ID, nil
}

func (s service) GetOrCreateSubjectSchemaId(ctx context.Context, subject string, schema string, schemaType SchemaType) (int, error) {
	registrySchema, err := buildRegistrySchema(schema, schemaType)
	if err != nil {
		return 0, err
	}

	subjectSchema, err := s.client.LookupSchema(ctx, subject, registrySchema)
	if err == nil {
		return subjectSchema.ID, nil
	}

	if !isSchemaLookupMiss(err) {
		return 0, fmt.Errorf("failed to lookup subject schema: %w", err)
	}

	subjectSchema, err = s.client.CreateSchema(ctx, subject, registrySchema)
	if err == nil {
		return subjectSchema.ID, nil
	}

	if !shouldRetryLookupAfterCreate(err) {
		return 0, fmt.Errorf("failed to create subject schema: %w", err)
	}

	subjectSchema, lookupErr := s.client.LookupSchema(ctx, subject, registrySchema)
	if lookupErr != nil {
		return 0, fmt.Errorf("failed to lookup subject schema after create conflict: %w", lookupErr)
	}

	return subjectSchema.ID, nil
}

func buildRegistrySchema(schema string, schemaType SchemaType) (sr.Schema, error) {
	var registrySchemaType sr.SchemaType

	switch schemaType {
	case Avro:
		registrySchemaType = sr.TypeAvro
	case Json:
		registrySchemaType = sr.TypeJSON
	case Protobuf:
		registrySchemaType = sr.TypeProtobuf
	default:
		return sr.Schema{}, fmt.Errorf("unknown schema type: %v", schemaType)
	}

	return sr.Schema{
		Schema: schema,
		Type:   registrySchemaType,
	}, nil
}

func isSchemaLookupMiss(err error) bool {
	var responseError *sr.ResponseError
	if !errors.As(err, &responseError) {
		return false
	}

	schemaError := responseError.SchemaError()

	return errors.Is(schemaError, sr.ErrSchemaNotFound) || errors.Is(schemaError, sr.ErrSubjectNotFound)
}

func shouldRetryLookupAfterCreate(err error) bool {
	var responseError *sr.ResponseError
	if !errors.As(err, &responseError) {
		return false
	}

	return isAlreadyRegisteredConflict(responseError)
}

func isAlreadyRegisteredConflict(err *sr.ResponseError) bool {
	if err.StatusCode != 409 {
		return false
	}

	message := strings.ToLower(err.Message)

	// Equivalent schemas are usually handled idempotently by the registry, but some
	// deployments can still surface conflict responses for concurrent registrations.
	// Only retry lookup for conflict messages that explicitly indicate an existing or
	// equivalent schema so real incompatibility errors still fail fast.
	return strings.Contains(message, "already exist") ||
		strings.Contains(message, "already registered") ||
		strings.Contains(message, "equivalent")
}
