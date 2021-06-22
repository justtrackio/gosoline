//+build integration fixtures

package test_test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/test/suite"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"io/ioutil"
	"testing"
)

const (
	basePath      = "test_data/s3_fixtures_test_data"
	basePathPurge = "test_data/s3_fixtures_purge_test_data"
	configName    = "test"
)

type FixturesS3Suite struct {
	suite.Suite
	logger     log.Logger
	client     s3iface.S3API
	loader     fixtures.FixtureLoader
	bucketName string
}

func (s *FixturesS3Suite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigFile("test_configs/config.fixtures_s3.test.yml"),
		suite.WithEnvSetup(func() error {
			s.logger = s.Env().Logger()
			s.bucketName = s.Env().Config().GetString("blobstore.test.bucket")

			return nil
		}),
	}
}

func (s *FixturesS3Suite) SetupTest() error {
	s.loader = fixtures.NewFixtureLoader(s.Env().Config(), s.logger)
	s.client = blob.ProvideS3Client(s.Env().Config())

	return nil
}

func TestFixturesS3Suite(t *testing.T) {
	s := new(FixturesS3Suite)
	suite.Run(t, s)
}

func (s *FixturesS3Suite) TestS3(app suite.AppUnderTest) {
	fs := s3DisabledPurgeFixtures()
	err := s.loader.Load(fs)
	s.NoError(err)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("nyan_cat2.gif")),
	}
	output, err := s.client.GetObject(input)
	s.NoError(err)

	body, err := ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("subDir/nyan_cat1.gif")),
	}
	output, err = s.client.GetObject(input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("subDir/nyan_cat.gif")),
	}
	output, err = s.client.GetObject(input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))
}

func (s *FixturesS3Suite) TestS3WithPurge(app suite.AppUnderTest) {
	fs := s3DisabledPurgeFixtures()
	err := s.loader.Load(fs)
	s.NoError(err)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("subDir/nyan_cat1.gif")),
	}
	output, err := s.client.GetObject(input)
	s.NoError(err)

	body, err := ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	fs = s3EnabledPurgeFixtures()
	err = s.loader.Load(fs)
	s.NoError(err)

	input = &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("nyan_cat3.gif")),
	}
	output, err = s.client.GetObject(input)
	s.NoError(err)

	body, err = ioutil.ReadAll(output.Body)

	s.NoError(err)
	s.Equal(28092, len(body))

	input = &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fmt.Sprint("nyan_cat2.gif")),
	}
	output, err = s.client.GetObject(input)

	awsErr, ok := err.(awserr.Error)

	s.True(ok)
	s.Equal(s3.ErrCodeNoSuchKey, awsErr.Code())
	s.Nil(output.Body)
}

func s3DisabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer: fixtures.BlobFixtureWriterFactory(&fixtures.BlobFixturesSettings{
				ConfigName: configName,
				BasePath:   basePath,
			}),
			Fixtures: nil,
		},
	}
}

func s3EnabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
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
}
