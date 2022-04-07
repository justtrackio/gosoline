package blob

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/funk"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Service struct {
	client gosoS3.Client
}

func NewService(ctx context.Context, config cfg.Config, logger log.Logger) (*Service, error) {
	s3Client, err := gosoS3.ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create s3 client default: %w", err)
	}

	return &Service{
		client: s3Client,
	}, nil
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
