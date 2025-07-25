package schema_registry

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/twmb/franz-go/pkg/sr"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	CheckCompatibility(ctx context.Context, subject string, version int, s sr.Schema) (sr.CheckCompatibilityResult, error)
	LookupSchema(ctx context.Context, subject string, s sr.Schema) (sr.SubjectSchema, error)
}

func NewClient(config cfg.Config, connectionName string) (Client, error) {
	conn, err := connection.ParseSettings(config, connectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", connectionName, err)
	}

	return sr.NewClient(sr.URLs(conn.SchemaRegistryAddress))
}
