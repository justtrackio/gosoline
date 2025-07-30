//go:build integration && fixtures

package blob_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

const (
	basePath   = "test_data/fixtures_test_data"
	configName = "test"
)

func TestS3TestSuite(t *testing.T) {
	suite.Run(t, new(S3TestSuite))
}

type S3TestSuite struct {
	suite.Suite
}

func (s *S3TestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
	}
}

func (s *S3TestSuite) TestS3() {
	err := s.Env().LoadFixtureSet(purgeDisabledFixtureSetsFactory)
	s.NoError(err)

	s3Client := s.Env().S3("default").Client()
	bucketName, err := s.Env().Config().GetString("blob.test.bucket")
	s.NoError(err)

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("nyan_cat2.gif"),
	}
	output, err := s3Client.GetObject(s.T().Context(), input)
	s.NoError(err)

	body, err := io.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("subDir/nyan_cat1.gif"),
	}
	output, err = s3Client.GetObject(s.T().Context(), input)
	s.NoError(err)

	body, err = io.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("subDir/nyan_cat.gif"),
	}
	output, err = s3Client.GetObject(s.T().Context(), input)
	s.NoError(err)

	body, err = io.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))
}

func purgeDisabledFixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	reader, err := blob.NewFileReader(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file reader writer: %w", err)
	}

	writer, err := blob.NewBlobFixtureWriter(ctx, config, logger, &blob.BlobFixturesSettings{
		ConfigName: configName,
		Reader:     reader,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create blob fixture writer: %w", err)
	}

	return []fixtures.FixtureSet{fixtures.NewSimpleFixtureSet[*blob.BlobFixture](nil, writer)}, nil
}
