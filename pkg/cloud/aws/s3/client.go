package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name PresignClient
type PresignClient interface {
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	AbortMultipartUpload(context.Context, *s3.AbortMultipartUploadInput, ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
	CompleteMultipartUpload(context.Context, *s3.CompleteMultipartUploadInput, ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	CreateMultipartUpload(context.Context, *s3.CreateMultipartUploadInput, ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(options *s3.Options)) (*s3.GetObjectOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(options *s3.Options)) (*s3.HeadBucketOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	PutObjectTagging(ctx context.Context, params *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, error)
	UploadPart(context.Context, *s3.UploadPartInput, ...func(*s3.Options)) (*s3.UploadPartOutput, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
	// Allows you to enable the client to use path-style addressing, i.e.,
	// https://s3.amazonaws.com/BUCKET/KEY . By default, the S3 client will use virtual
	// hosted bucket addressing when possible( https://BUCKET.s3.amazonaws.com/KEY ).
	UsePathStyle bool `cfg:"usePathStyle" default:"true"`
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

type (
	clientAppCtxKey        string
	presignClientAppCtxKey string
)

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*s3.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*s3.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func ProvidePresignClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*s3.PresignClient, error) {
	return appctx.Provide(ctx, presignClientAppCtxKey(name), func() (*s3.PresignClient, error) {
		return NewPresignClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*s3.Client, error) {
	clientCfg, err := GetClientConfig(config, name, optFns...)
	if err != nil {
		return nil, err
	}

	var awsConfig aws.Config
	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = gosoAws.NilIfEmpty(clientCfg.Settings.Endpoint)
		o.UsePathStyle = clientCfg.Settings.UsePathStyle
	})

	gosoAws.LogNewClientCreated(ctx, logger, "s3", name, clientCfg.Settings.ClientSettings)

	return client, nil
}

func NewPresignClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*s3.PresignClient, error) {
	client, err := ProvideClient(ctx, config, logger, name, optFns...)
	if err != nil {
		return nil, fmt.Errorf("can not initialize client: %w", err)
	}

	pClient := s3.NewPresignClient(client)

	// Since this is not really a new client,
	// but uses an s3 client of the same name,
	// no logging of new client creation here.

	return pClient, nil
}

func GetClientConfig(config cfg.Config, name string, optFns ...ClientOption) (*ClientConfig, error) {
	clientCfg := &ClientConfig{}
	if err := gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "s3", name); err != nil {
		return nil, fmt.Errorf("failed to unmarshal S3 client settings: %w", err)
	}

	for _, opt := range optFns {
		opt(clientCfg)
	}

	return clientCfg, nil
}

func ResolveEndpoint(config cfg.Config, name string, optFns ...ClientOption) (string, error) {
	clientCfg, err := GetClientConfig(config, name, optFns...)
	if err != nil {
		return "", err
	}

	if clientCfg.Settings.Endpoint != "" {
		return clientCfg.Settings.Endpoint, nil
	}

	endpoint, err := s3.NewDefaultEndpointResolver().ResolveEndpoint(clientCfg.Settings.Region, s3.EndpointResolverOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to resolve s3 endpoint: %w", err)
	}

	return endpoint.URL, nil
}
