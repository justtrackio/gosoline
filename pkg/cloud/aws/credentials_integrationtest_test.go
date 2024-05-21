//go:build integration
// +build integration

package aws_test

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
)

func (s *CredentialsTestSuite) TestNoConfiguredProvider() {
	s.config.On("HasPrefix", "cloud.aws.credentials").Return(false)

	provider, err := gosoAws.GetCredentialsProvider(s.ctx, s.config, gosoAws.ClientSettings{})
	s.NoError(err)
	s.IsType(credentials.StaticCredentialsProvider{}, provider, "the provider should be a static one")

	expected := aws.Credentials{
		AccessKeyID:     gosoAws.DefaultAccessKeyID,
		SecretAccessKey: gosoAws.DefaultSecretAccessKey,
		SessionToken:    gosoAws.DefaultToken,
		Source:          "StaticCredentials",
		CanExpire:       false,
		Expires:         time.Time{},
	}

	credentials, err := provider.Retrieve(s.ctx)
	s.NoError(err)
	s.Equal(expected, credentials)
}
