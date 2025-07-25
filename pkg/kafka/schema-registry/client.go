package schema_registry

import (
	"context"

	"github.com/twmb/franz-go/pkg/sr"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	CreateSchema(ctx context.Context, subject string, s sr.Schema) (sr.SubjectSchema, error)
	LookupSchema(ctx context.Context, subject string, s sr.Schema) (sr.SubjectSchema, error)
}

func NewClient(schemaRegistryAddress string) (Client, error) {
	return sr.NewClient(sr.URLs(schemaRegistryAddress))
}
