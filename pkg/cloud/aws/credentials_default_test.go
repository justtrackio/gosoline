//go:build !integration
// +build !integration

package aws_test

import (
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
)

func (s *CredentialsTestSuite) TestNoConfiguredProvider() {
	provider, err := gosoAws.GetCredentialsProvider(s.ctx, gosoAws.ClientSettings{})

	s.NoError(err)
	s.Nil(provider, "there should be no provider returned")
}
