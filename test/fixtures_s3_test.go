//go:build integration && fixtures
// +build integration,fixtures

package test_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/justtrackio/gosoline/pkg/fixtures/writers"
	s32 "github.com/justtrackio/gosoline/pkg/fixtures/writers/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

const (
	basePath      = "test_data/s3_fixtures_test_data"
	basePathPurge = "test_data/s3_fixtures_purge_test_data"
	configName    = "test"
)

type FixturesS3Suite struct {
	suite.Suite
	ctx        context.Context
	bucketName string
}

func (s *FixturesS3Suite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("test_configs/config.fixtures_s3.test.yml"),
	}
}

func (s *FixturesS3Suite) SetupTest() (err error) {
	s.ctx = context.Background()
	s.bucketName = s.Env().Config().GetString("blobstore.test.bucket")
	return
}

func TestFixturesS3Suite(t *testing.T) {
	s := new(FixturesS3Suite)
	suite.Run(t, s)
}

func (s *FixturesS3Suite) TestS3() {
	fs := s3DisabledPurgeFixtures()
	err := s.Env().LoadFixtures(fs)
	s.NoError(err)

	s3Client := s.Env().S3("default").Client()

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String("nyan_cat2.gif"),
	}
	output, err := s3Client.GetObject(s.ctx, input)
	s.NoError(err)

	body, err := ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String("subDir/nyan_cat1.gif"),
	}
	output, err = s3Client.GetObject(s.ctx, input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String("subDir/nyan_cat.gif"),
	}
	output, err = s3Client.GetObject(s.ctx, input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))
}

func (s *FixturesS3Suite) TestS3WithPurge() {
	fs := s3DisabledPurgeFixtures()
	err := s.Env().LoadFixtures(fs)
	s.NoError(err)

	s3Client := s.Env().S3("default").Client()

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("subDir/nyan_cat1.gif")),
	}
	output, err := s3Client.GetObject(s.ctx, input)
	s.NoError(err)

	body, err := ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	fs = s3EnabledPurgeFixtures()
	err = s.Env().LoadFixtures(fs)
	s.NoError(err)

	input = &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("nyan_cat3.gif")),
	}
	output, err = s3Client.GetObject(s.ctx, input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("nyan_cat2.gif")),
	}
	output, err = s3Client.GetObject(s.ctx, input)

	var noSuchKey *types.NoSuchKey
	isNoSuchKeyErr := errors.As(err, &noSuchKey)

	s.True(isNoSuchKeyErr)
	s.Nil(output)
}

func s3DisabledPurgeFixtures() []*writers.FixtureSet {
	return []*writers.FixtureSet{
		{
			Enabled: true,
			Writer: s32.BlobFixtureWriterFactory(&s32.BlobFixturesSettings{
				ConfigName: configName,
				BasePath:   basePath,
			}),
			Fixtures: nil,
		},
	}
}

func s3EnabledPurgeFixtures() []*writers.FixtureSet {
	return []*writers.FixtureSet{
		{
			Enabled: true,
			Purge:   true,
			Writer: s32.BlobFixtureWriterFactory(&s32.BlobFixturesSettings{
				ConfigName: configName,
				BasePath:   basePathPurge,
			}),
			Fixtures: nil,
		},
	}
}
