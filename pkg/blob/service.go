package blob

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Service struct {
	logger   log.Logger
	client   *s3.Client
	settings *Settings
}

func NewService(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (*Service, error) {
	var err error
	var client *s3.Client

	if client, err = gosoS3.ProvideClient(ctx, config, logger, settings.ClientName); err != nil {
		return nil, fmt.Errorf("can not create s3 client with name %s: %w", settings.ClientName, err)
	}

	return &Service{
		logger:   logger,
		client:   client,
		settings: settings,
	}, nil
}

func (l *Service) CreateBucket(ctx context.Context) error {
	if _, err := l.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(l.settings.Bucket)}); err == nil {
		return nil
	}

	_, err := l.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(l.settings.Bucket),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(l.settings.Region), // This is required when using region specific endpoints
		},
	})

	if isBucketAlreadyExistsError(err) {
		l.logger.Info("s3 bucket %s did already exist", l.settings.Bucket)

		return nil
	}

	if err != nil {
		return fmt.Errorf("could not create s3 bucket %s: %w", l.settings.Bucket, err)
	}

	l.logger.Info("created s3 bucket %s", l.settings.Bucket)

	return nil
}

func (l *Service) Purge(ctx context.Context) error {
	var err error
	var out *s3.ListObjectsOutput

	input := &s3.ListObjectsInput{
		Bucket: aws.String(l.settings.Bucket),
		Prefix: aws.String(l.settings.Prefix),
	}

	for {
		if out, err = l.client.ListObjects(ctx, input); err != nil {
			return err
		}

		if len(out.Contents) == 0 {
			return nil
		}

		objects := funk.Map(out.Contents, func(object types.Object) types.ObjectIdentifier {
			return types.ObjectIdentifier{
				Key: object.Key,
			}
		})

		deleteInput := &s3.DeleteObjectsInput{
			Bucket: aws.String(l.settings.Bucket),
			Delete: &types.Delete{
				Objects: objects,
			},
		}

		if _, err = l.client.DeleteObjects(ctx, deleteInput); err != nil {
			return fmt.Errorf("could not delete objects: %w", err)
		}

		if !*out.IsTruncated {
			break
		}

		input.Marker = out.NextMarker
	}

	return nil
}

func (s *Service) DeleteObjects(ctx context.Context, bucket string, objects []*types.Object) error {
	chunks := funk.Chunk(objects, 1000)

	for _, chunk := range chunks {
		if err := s.deleteChunk(ctx, bucket, chunk); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) deleteChunk(ctx context.Context, bucket string, objects []*types.Object) error {
	del := &types.Delete{
		Objects: make([]types.ObjectIdentifier, 0),
	}

	for _, obj := range objects {
		objId := types.ObjectIdentifier{
			Key: obj.Key,
		}

		del.Objects = append(del.Objects, objId)
	}

	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: del,
	}

	out, err := s.client.DeleteObjects(ctx, input)
	if err != nil {
		return fmt.Errorf("wasn't able to delete objects from bucket %s: %w", bucket, err)
	}

	multiErr := &multierror.Error{}
	for _, e := range out.Errors {
		multiErr = multierror.Append(multiErr, fmt.Errorf("wasn't able to delete key %s from bucket %s: (%s) %s", *e.Key, bucket, *e.Code, *e.Message))
	}

	return multiErr.ErrorOrNil()
}

func (s *Service) ListObjects(ctx context.Context, bucket string, prefix string) ([]types.Object, error) {
	objects := make([]types.Object, 0, 1024)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	for {
		out, err := s.client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("can not list objects in s3 bucket: %w", err)
		}

		objects = append(out.Contents, objects...)

		if out.NextContinuationToken == nil {
			break
		}

		input.ContinuationToken = out.NextContinuationToken
	}

	return objects, nil
}

func isBucketAlreadyExistsError(err error) bool {
	var bucketAlreadyExists *types.BucketAlreadyExists
	var bucketAlreadyOwnedByYou *types.BucketAlreadyOwnedByYou

	if errors.As(err, &bucketAlreadyExists) || errors.As(err, &bucketAlreadyOwnedByYou) {
		return true
	}

	return false
}
