//go:build !integration
// +build !integration

package aws_test

import (
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
)

func (s *CredentialsTestSuite) TestNoConfiguredProvider() {
	s.config.On("HasPrefix", "cloud.aws.credentials").Return(false)

	provider, err := gosoAws.GetCredentialsProvider(s.ctx, s.config, gosoAws.ClientSettings{})

	s.NoError(err)
	s.Nil(provider, "there should be no provider returned")
}
