package blob

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/hashicorp/go-multierror"
	"github.com/thoas/go-funk"
)

type Service struct {
	client s3iface.S3API
}

func NewService(config cfg.Config, logger log.Logger) *Service {
	client := ProvideS3Client(config)

	return &Service{
		client: client,
	}
}

func (s *Service) DeleteObjects(bucket string, objects []*s3.Object) error {
	chunks := funk.Chunk(objects, 1000).([][]*s3.Object)

	for _, chunk := range chunks {
		if err := s.deleteChunk(bucket, chunk); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) deleteChunk(bucket string, objects []*s3.Object) error {
	del := &s3.Delete{
		Objects: make([]*s3.ObjectIdentifier, 0),
	}

	for _, obj := range objects {
		objId := &s3.ObjectIdentifier{
			Key: obj.Key,
		}

		del.Objects = append(del.Objects, objId)
	}

	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: del,
	}

	out, err := s.client.DeleteObjects(input)

	if err != nil {
		return fmt.Errorf("wasn't able to delete objects from bucket %s: %w", bucket, err)
	}

	multiErr := &multierror.Error{}
	for _, e := range out.Errors {
		multiErr = multierror.Append(multiErr, fmt.Errorf("wasn't able to delete key %s from bucket %s: (%s) %s", *e.Key, bucket, *e.Code, *e.Message))
	}

	return multiErr.ErrorOrNil()
}

func (s *Service) ListObjects(bucket string, prefix string) ([]*s3.Object, error) {
	objects := make([]*s3.Object, 0, 1024)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	for {
		out, err := s.client.ListObjectsV2(input)

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
