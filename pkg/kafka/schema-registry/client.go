package schema_registry

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/twmb/franz-go/pkg/sr"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	CheckCompatibility(ctx context.Context, subject string, version int, s sr.Schema) (sr.CheckCompatibilityResult, error)
	LookupSchema(ctx context.Context, subject string, s sr.Schema) (sr.SubjectSchema, error)
}

func NewClient(connection connection.Settings) (Client, error) {
	return sr.NewClient(sr.URLs(connection.SchemaRegistryAddress))
}
