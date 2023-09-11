package aws_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/justtrackio/gosoline/pkg/cfg/mocks"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestCredentialsTestSuite(t *testing.T) {
	suite.Run(t, new(CredentialsTestSuite))
}

type CredentialsTestSuite struct {
	suite.Suite
	ctx    context.Context
	config *mocks.Config
}

func (s *CredentialsTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.config = new(mocks.Config)
}

func (s *CredentialsTestSuite) TestNoConfiguredProvider() {
	s.config.On("HasPrefix", "cloud.aws.credentials").Return(false)

	provider, err := gosoAws.GetCredentialsProvider(s.ctx, s.config, gosoAws.ClientSettings{})

	s.NoError(err)
	s.Nil(provider, "there should be no provider returned")
}

func (s *CredentialsTestSuite) TestStaticCredentialsProvider() {
	s.config.On("HasPrefix", "cloud.aws.credentials").Return(true)
	s.config.On("UnmarshalKey", "cloud.aws.credentials", mock.AnythingOfType("*aws.Credentials")).Run(func(args mock.Arguments) {
		credentials := args.Get(1).(*gosoAws.Credentials)

		credentials.AccessKeyID = "AccessKeyID"
		credentials.SecretAccessKey = "SecretAccessKey"
		credentials.SessionToken = "SessionToken"
	})

	provider, err := gosoAws.GetCredentialsProvider(s.ctx, s.config, gosoAws.ClientSettings{})
	s.NoError(err)
	s.IsType(credentials.StaticCredentialsProvider{}, provider, "the provider should be a static one")

	expected := aws.Credentials{
		AccessKeyID:     "AccessKeyID",
		SecretAccessKey: "SecretAccessKey",
		SessionToken:    "SessionToken",
		Source:          "StaticCredentials",
		CanExpire:       false,
		Expires:         time.Time{},
	}
	credentials, err := provider.Retrieve(s.ctx)
	s.NoError(err)
	s.Equal(expected, credentials)
}

func (s *CredentialsTestSuite) TestAssumeRoleCredentialsProvider() {
	provider, err := gosoAws.GetCredentialsProvider(s.ctx, s.config, gosoAws.ClientSettings{
		AssumeRole: "arn:aws:iam::123456789012:role/gosoline-test-role",
	})

	s.NoError(err)
	s.IsType(&stscreds.AssumeRoleProvider{}, provider, "the provider should be a assume role one")
}
