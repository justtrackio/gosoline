package s3_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type NamingTestSuite struct {
	suite.Suite
}

func TestNamingTestSuite(t *testing.T) {
	suite.Run(t, new(NamingTestSuite))
}

func (s *NamingTestSuite) TestGetBucketName() {
	appConfig := map[string]any{
		"env":       "test",
		"name":      "app",
		"namespace": "{app.env}-{app.tags.project}-{app.tags.family}-{app.tags.group}",
		"tags": map[string]any{
			"project": "proj",
			"family":  "fam",
			"group":   "grp",
		},
	}

	tests := []struct {
		name     string
		config   map[string]any
		settings *s3.BucketNameSettings
		expected string
	}{
		{
			name: "default",
			config: map[string]any{
				"app": appConfig,
			},
			settings: &s3.BucketNameSettings{
				AppIdentity: cfg.AppIdentity{},
				ClientName:  "default",
				BucketId:    "bucketId",
			},
			expected: "proj-test-fam",
		},
		{
			name: "specific client",
			config: map[string]any{
				"app": appConfig,
				"cloud.aws.s3.clients.specific.naming.bucket_pattern": "{app.name}-{bucketId}",
			},
			settings: &s3.BucketNameSettings{
				AppIdentity: cfg.AppIdentity{
					Env:  "test",
					Name: "app",
				},
				ClientName: "specific",
				BucketId:   "my_bucket",
			},
			expected: "app-my_bucket",
		},
		{
			name: "default pattern override",
			config: map[string]any{
				"app": appConfig,
				"cloud.aws.s3.clients.default.naming.bucket_pattern": "{app.env}-{bucketId}",
			},
			settings: &s3.BucketNameSettings{
				AppIdentity: cfg.AppIdentity{
					Env: "test",
				},
				ClientName: "default",
				BucketId:   "bucketId",
			},
			expected: "test-bucketId",
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			config := cfg.New()
			err := config.Option(cfg.WithConfigMap(test.config))
			assert.NoError(s.T(), err)

			name, err := s3.GetBucketName(config, test.settings)
			assert.NoError(s.T(), err)
			assert.Equal(s.T(), test.expected, name)
		})
	}
}
