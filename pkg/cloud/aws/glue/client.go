package glue

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	BatchCreatePartition(ctx context.Context, params *glue.BatchCreatePartitionInput, optFns ...func(*glue.Options)) (*glue.BatchCreatePartitionOutput, error)
	BatchDeletePartition(ctx context.Context, params *glue.BatchDeletePartitionInput, optFns ...func(*glue.Options)) (*glue.BatchDeletePartitionOutput, error)
	BatchGetPartition(ctx context.Context, params *glue.BatchGetPartitionInput, optFns ...func(*glue.Options)) (*glue.BatchGetPartitionOutput, error)
	BatchUpdatePartition(ctx context.Context, params *glue.BatchUpdatePartitionInput, optFns ...func(*glue.Options)) (*glue.BatchUpdatePartitionOutput, error)
	CreateDatabase(ctx context.Context, params *glue.CreateDatabaseInput, optFns ...func(*glue.Options)) (*glue.CreateDatabaseOutput, error)
	CreateTable(ctx context.Context, params *glue.CreateTableInput, optFns ...func(*glue.Options)) (*glue.CreateTableOutput, error)
	DeleteDatabase(ctx context.Context, params *glue.DeleteDatabaseInput, optFns ...func(*glue.Options)) (*glue.DeleteDatabaseOutput, error)
	DeletePartition(ctx context.Context, params *glue.DeletePartitionInput, optFns ...func(*glue.Options)) (*glue.DeletePartitionOutput, error)
	DeleteTable(ctx context.Context, params *glue.DeleteTableInput, optFns ...func(*glue.Options)) (*glue.DeleteTableOutput, error)
	GetDatabase(ctx context.Context, params *glue.GetDatabaseInput, optFns ...func(*glue.Options)) (*glue.GetDatabaseOutput, error)
	GetDatabases(ctx context.Context, params *glue.GetDatabasesInput, optFns ...func(*glue.Options)) (*glue.GetDatabasesOutput, error)
	GetPartition(ctx context.Context, params *glue.GetPartitionInput, optFns ...func(*glue.Options)) (*glue.GetPartitionOutput, error)
	GetPartitions(ctx context.Context, params *glue.GetPartitionsInput, optFns ...func(*glue.Options)) (*glue.GetPartitionsOutput, error)
	GetTable(ctx context.Context, params *glue.GetTableInput, optFns ...func(*glue.Options)) (*glue.GetTableOutput, error)
	GetTables(ctx context.Context, params *glue.GetTablesInput, optFns ...func(*glue.Options)) (*glue.GetTablesOutput, error)
	UpdateDatabase(ctx context.Context, params *glue.UpdateDatabaseInput, optFns ...func(*glue.Options)) (*glue.UpdateDatabaseOutput, error)
	UpdatePartition(ctx context.Context, params *glue.UpdatePartitionInput, optFns ...func(*glue.Options)) (*glue.UpdatePartitionOutput, error)
	UpdateTable(ctx context.Context, params *glue.UpdateTableInput, optFns ...func(*glue.Options)) (*glue.UpdateTableOutput, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
}

type ClientConfig struct {
	Settings    ClientSettings
	LoadOptions []func(options *awsCfg.LoadOptions) error
}

func (c ClientConfig) GetSettings() gosoAws.ClientSettings {
	return c.Settings.ClientSettings
}

func (c ClientConfig) GetLoadOptions() []func(options *awsCfg.LoadOptions) error {
	return c.LoadOptions
}

func (c ClientConfig) GetRetryOptions() []func(*retry.StandardOptions) {
	return nil
}

type ClientOption func(cfg *ClientConfig)

type clientAppCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*glue.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*glue.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*glue.Client, error) {
	var err error
	var clientCfg *ClientConfig
	var awsConfig aws.Config

	if clientCfg, awsConfig, err = NewConfig(ctx, config, logger, name, optFns...); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := glue.NewFromConfig(awsConfig, func(options *glue.Options) {
		options.BaseEndpoint = gosoAws.NilIfEmpty(clientCfg.Settings.Endpoint)
	})

	gosoAws.LogNewClientCreated(ctx, logger, "glue", name, clientCfg.Settings.ClientSettings)

	return client, nil
}

func NewConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*ClientConfig, aws.Config, error) {
	clientCfg := &ClientConfig{}
	if err := gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "glue", name); err != nil {
		return nil, aws.Config{}, fmt.Errorf("failed to unmarshal Glue client settings: %w", err)
	}

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, aws.Config{}, fmt.Errorf("can not initialize config: %w", err)
	}

	return clientCfg, awsConfig, nil
}
