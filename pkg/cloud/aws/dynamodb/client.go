package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(options *dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	ListTagsOfResource(ctx context.Context, params *dynamodb.ListTagsOfResourceInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTagsOfResourceOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	TransactGetItems(ctx context.Context, params *dynamodb.TransactGetItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactGetItemsOutput, error)
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	UpdateTable(ctx context.Context, params *dynamodb.UpdateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateTableOutput, error)
	UpdateTimeToLive(ctx context.Context, params *dynamodb.UpdateTimeToLiveInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateTimeToLiveOutput, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
	// Allows you to disable the client's validation of response integrity using CRC32
	// checksum. Enabled by default.
	DisableValidateResponseChecksum bool `cfg:"disable_validate_response_checksum" default:"false"`
	// Allows you to enable the client's support for compressed gzip responses.
	// Disabled by default.
	EnableAcceptEncodingGzip bool `cfg:"enable_accept_encoding_gzip" default:"false"`
	// Configures the way we purge a table (when loading fixtures)
	//  - scan: Scan the table and perform batch deletes for every item. Slower, but does not modify infrastructure.
	//  - drop_table: Delete the table and create a new one using the settings provided to the repository.
	PurgeType string `cfg:"purge_type" default:"scan" validate:"oneof=scan drop_table"`
	// When using PurgeType "scan", configure the number of parallel workers scanning and deleting items.
	// Uses the number of CPU cores when set to 0.
	PurgeParallelism int `cfg:"purge_parallelism" default:"0" validate:"min=0"`
}

type ClientConfig struct {
	Settings     ClientSettings
	LoadOptions  []func(options *awsCfg.LoadOptions) error
	RetryOptions []func(*retry.StandardOptions)
}

func (c ClientConfig) GetSettings() gosoAws.ClientSettings {
	return c.Settings.ClientSettings
}

func (c ClientConfig) GetLoadOptions() []func(options *awsCfg.LoadOptions) error {
	return c.LoadOptions
}

func (c ClientConfig) GetRetryOptions() []func(*retry.StandardOptions) {
	return c.RetryOptions
}

type ClientOption func(cfg *ClientConfig)

type clientAppCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*dynamodb.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*dynamodb.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*dynamodb.Client, error) {
	clientCfg := &ClientConfig{}
	if err := gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "dynamodb", name); err != nil {
		return nil, fmt.Errorf("failed to unmarshal client settings: %w", err)
	}

	clientCfg.RetryOptions = []func(*retry.StandardOptions){
		gosoAws.RetryWithRetryables([]retry.IsErrorRetryable{
			&RetryOnTransactionConflict{},
			&RetryOnConditionalCheckFailed{},
		}),
	}

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsConfig, func(options *dynamodb.Options) {
		options.BaseEndpoint = gosoAws.NilIfEmpty(clientCfg.Settings.Endpoint)
		options.DisableValidateResponseChecksum = clientCfg.Settings.DisableValidateResponseChecksum
		options.EnableAcceptEncodingGzip = clientCfg.Settings.EnableAcceptEncodingGzip
	})

	gosoAws.LogNewClientCreated(ctx, logger, "dynamodb", name, clientCfg.Settings.ClientSettings)

	return client, nil
}
