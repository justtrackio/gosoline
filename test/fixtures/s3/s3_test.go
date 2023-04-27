//go:build integration && fixtures

package s3_test

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/justtrackio/gosoline/pkg/fixtures"
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
	err := s.Env().LoadFixtureBuilderFactories(fixtures.SimpleFixtureBuilderFactory(s3DisabledPurgeFixtures))
	s.NoError(err)

	s3Client := s.Env().S3("default").Client()
	bucketName := s.Env().Config().GetString("blob.test.bucket")

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("nyan_cat2.gif"),
	}
	output, err := s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err := ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("subDir/nyan_cat1.gif"),
	}
	output, err = s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("subDir/nyan_cat.gif"),
	}
	output, err = s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))
}

func (s *S3TestSuite) TestS3WithPurge() {
	err := s.Env().LoadFixtureBuilderFactories(fixtures.SimpleFixtureBuilderFactory(s3DisabledPurgeFixtures))
	s.NoError(err)

	s3Client := s.Env().S3("default").Client()
	bucketName := s.Env().Config().GetString("blob.test.bucket")

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("subDir/nyan_cat1.gif"),
	}
	output, err := s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err := ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	err = s.Env().LoadFixtureBuilderFactories(fixtures.SimpleFixtureBuilderFactory(s3EnabledPurgeFixtures))
	s.NoError(err)

	input = &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("nyan_cat3.gif"),
	}
	output, err = s3Client.GetObject(context.Background(), input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

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

var s3DisabledPurgeFixtures = []*fixtures.FixtureSet{
	{
		Enabled: true,
		Writer: fixtures.BlobFixtureWriterFactory(&fixtures.BlobFixturesSettings{
			ConfigName: configName,
			BasePath:   basePath,
		}),
		Fixtures: nil,
	},
}

var s3EnabledPurgeFixtures = []*fixtures.FixtureSet{
	{
		Enabled: true,
		Purge:   true,
		Writer: fixtures.BlobFixtureWriterFactory(&fixtures.BlobFixturesSettings{
			ConfigName: configName,
			BasePath:   basePathPurge,
		}),
		Fixtures: nil,
	},
}

func TestS3TestSuite(t *testing.T) {
	suite.Run(t, new(S3TestSuite))
}
