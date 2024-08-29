package aws_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/justtrackio/gosoline/pkg/cfg"
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
	tests := map[string]map[string]interface{}{
		"using cloud.aws.defaults.credentials": {
			"defaults": map[string]interface{}{
				"credentials": map[string]interface{}{
					"access_key_id":     "AccessKeyID",
					"secret_access_key": "SecretAccessKey",
					"session_token":     "SessionToken",
				},
			},
		},
		"using credentials from client config": {
			"ddb": map[string]interface{}{
				"clients": map[string]interface{}{
					"default": map[string]interface{}{
						"credentials": map[string]interface{}{
							"access_key_id":     "AccessKeyID",
							"secret_access_key": "SecretAccessKey",
							"session_token":     "SessionToken",
						},
					},
				},
			},
		},
	}

	for name, values := range tests {
		s.Run(name, func() {
			settings := s.unmarshalClientSettings(values)

			provider, err := gosoAws.GetCredentialsProvider(s.ctx, settings)
			s.NoError(err)
			s.IsType(credentials.StaticCredentialsProvider{}, provider, "the provider should be a static one")

			expected := aws.Credentials{
				AccessKeyID:     "AccessKeyID",
				SecretAccessKey: "SecretAccessKey",
				SessionToken:    "SessionToken",
				Source:          "StaticCredentials",
			}
			credentials, err := provider.Retrieve(s.ctx)
			s.NoError(err)
			s.Equal(expected, credentials)
		})
	}
}

func (s *CredentialsTestSuite) TestAssumeRoleCredentialsProvider() {
	tests := map[string]map[string]interface{}{
		"using cloud.aws.defaults.assume_role": {
			"defaults": map[string]interface{}{
				"assume_role": "arn:aws:iam::123456789012:role/gosoline-test-role",
			},
		},
		"using role from client config": {
			"ddb": map[string]interface{}{
				"clients": map[string]interface{}{
					"default": map[string]interface{}{
						"assume_role": "arn:aws:iam::123456789012:role/gosoline-test-role",
					},
				},
			},
		},
	}

	for name, values := range tests {
		s.Run(name, func() {
			settings := s.unmarshalClientSettings(values)
			provider, err := gosoAws.GetCredentialsProvider(s.ctx, settings)

			s.NoError(err)
			s.IsType(&stscreds.AssumeRoleProvider{}, provider, "the provider should be a assume role one")
		})
	}
}

func (s *CredentialsTestSuite) TestProfileCredentials() {
	tests := map[string]map[string]interface{}{
		"using cloud.aws.defaults.profile": {
			"defaults": map[string]interface{}{
				"profile": "sdlc-dev-account",
			},
		},
		"using profile from client config": {
			"ddb": map[string]interface{}{
				"clients": map[string]interface{}{
					"default": map[string]interface{}{
						"profile": "sdlc-dev-account",
					},
				},
			},
		},
	}

	for name, values := range tests {
		s.Run(name, func() {
			settings := s.unmarshalClientSettings(values)

			option, err := gosoAws.GetCredentialsOption(s.ctx, settings)
			s.NoError(err)

			awsLoadOptions := &config.LoadOptions{}
			err = option(awsLoadOptions)
			s.NoError(err)

			s.Equal("sdlc-dev-account", awsLoadOptions.SharedConfigProfile)
		})
	}
}

func (s *CredentialsTestSuite) unmarshalClientSettings(values map[string]interface{}) gosoAws.ClientSettings {
	settings := gosoAws.ClientSettings{}
	config := cfg.New(map[string]interface{}{
		"cloud": map[string]interface{}{
			"aws": values,
		},
	})

	gosoAws.UnmarshalClientSettings(config, &settings, "ddb", "default")

	return settings
}
