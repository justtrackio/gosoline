package aws_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/stretchr/testify/suite"
)

func TestCredentialsTestSuite(t *testing.T) {
	suite.Run(t, new(CredentialsTestSuite))
}

type CredentialsTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *CredentialsTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *CredentialsTestSuite) TestStaticCredentialsProvider() {
	provider, err := gosoAws.GetCredentialsProvider(s.ctx, gosoAws.ClientSettings{
		Credentials: gosoAws.Credentials{
			AccessKeyID:     "AccessKeyID",
			SecretAccessKey: "SecretAccessKey",
			SessionToken:    "SessionToken",
		},
	})
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
	provider, err := gosoAws.GetCredentialsProvider(s.ctx, gosoAws.ClientSettings{
		AssumeRole: "arn:aws:iam::123456789012:role/gosoline-test-role",
	})

	s.NoError(err)
	s.IsType(&stscreds.AssumeRoleProvider{}, provider, "the provider should be a assume role one")
}
