//go:build integration && fixtures

package s3_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

const (
	basePath      = "test_data/s3_fixtures_test_data"
	basePathPurge = "test_data/s3_fixtures_purge_test_data"
	configName    = "test"
)

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
	err := s.Env().LoadFixtureSets(purgeDisabledFixtureSetsFactory)
	s.NoError(err)

	s3Client := s.Env().S3("default").Client()
	bucketName := s.Env().Config().GetString("blob.test.bucket")

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("nyan_cat2.gif"),
	}
	output, err := s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err := io.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("subDir/nyan_cat1.gif"),
	}
	output, err = s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err = io.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("subDir/nyan_cat.gif"),
	}
	output, err = s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err = io.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))
}

func (s *S3TestSuite) TestS3WithPurge() {
	err := s.Env().LoadFixtureSets(purgeDisabledFixtureSetsFactory)
	s.NoError(err)

	s3Client := s.Env().S3("default").Client()
	bucketName := s.Env().Config().GetString("blob.test.bucket")

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("subDir/nyan_cat1.gif"),
	}
	output, err := s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err := io.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	err = s.Env().LoadFixtureSets(purgeEnabledFixtureSetsFactory)
	s.NoError(err)

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("nyan_cat3.gif"),
	}
	output, err = s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err = io.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("nyan_cat2.gif"),
	}
	output, err = s3Client.GetObject(context.Background(), input)

	var noSuchKey *types.NoSuchKey
	isNoSuchKeyErr := errors.As(err, &noSuchKey)

	s.True(isNoSuchKeyErr)
	s.Nil(output)
}

func purgeDisabledFixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewBlobFixtureWriter(ctx, config, logger, &fixtures.BlobFixturesSettings{
		ConfigName: configName,
		BasePath:   basePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 fixture writer: %w", err)
	}

	return []fixtures.FixtureSet{fixtures.NewSimpleFixtureSet[*fixtures.BlobFixture](nil, writer)}, nil
}

func purgeEnabledFixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewBlobFixtureWriter(ctx, config, logger, &fixtures.BlobFixturesSettings{
		ConfigName: configName,
		BasePath:   basePathPurge,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 fixture writer: %w", err)
	}

	return []fixtures.FixtureSet{fixtures.NewSimpleFixtureSet[*fixtures.BlobFixture](nil, writer, fixtures.WithPurge(true))}, nil
}

func TestS3TestSuite(t *testing.T) {
	suite.Run(t, new(S3TestSuite))
}
