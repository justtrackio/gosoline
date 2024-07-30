package s3

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Client
type Client interface {
	CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(options *s3.Options)) (*s3.GetObjectOutput, error)
	ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	PutObjectTagging(ctx context.Context, params *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	UploadPart(context.Context, *s3.UploadPartInput, ...func(*s3.Options)) (*s3.UploadPartOutput, error)
	CreateMultipartUpload(context.Context, *s3.CreateMultipartUploadInput, ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	CompleteMultipartUpload(context.Context, *s3.CompleteMultipartUploadInput, ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	AbortMultipartUpload(context.Context, *s3.AbortMultipartUploadInput, ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
}

type clientAppCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*s3.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*s3.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*s3.Client, error) {
	clientCfg := GetClientConfig(config, name, optFns...)

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = clientCfg.Settings.UsePathStyle
	})

	gosoAws.LogNewClientCreated(ctx, logger, "s3", name, clientCfg.Settings.ClientSettings)

	return client, nil
}

func GetClientConfig(config cfg.Config, name string, optFns ...ClientOption) *ClientConfig {
	clientCfg := &ClientConfig{}
	gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "s3", name)

	for _, opt := range optFns {
		opt(clientCfg)
	}

	return clientCfg
}

func ResolveEndpoint(config cfg.Config, name string, optFns ...ClientOption) (aws.Endpoint, error) {
	clientCfg := GetClientConfig(config, name, optFns...)
	gosoResolver := gosoAws.EndpointResolver(clientCfg.Settings.Endpoint)

	endpoint, err := gosoResolver.ResolveEndpoint("s3", clientCfg.Settings.Region)

	if nf := (&aws.EndpointNotFoundError{}); !errors.As(err, &nf) {
		return endpoint, err
	}

	return s3.NewDefaultEndpointResolver().ResolveEndpoint(clientCfg.Settings.Region, s3.EndpointResolverOptions{})
}
