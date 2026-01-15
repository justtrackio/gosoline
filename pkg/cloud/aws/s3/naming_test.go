package s3_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/stretchr/testify/suite"
)

func TestGetBucketNameTestSuite(t *testing.T) {
	suite.Run(t, new(GetBucketNameTestSuite))
}

type GetBucketNameTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    s3.BucketNameSettings
}

func (s *GetBucketNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.settings = s3.BucketNameSettings{
		Identity: cfg.Identity{
			Name:      "producer",
			Env:       "test",
			Namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}",
			Tags: cfg.Tags{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "group",
			},
		},
		ClientName: "default",
		BucketId:   "my-bucket",
	}

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *GetBucketNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *GetBucketNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *GetBucketNameTestSuite) TestDefault() {
	name, err := s3.GetBucketName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group", name)
}

func (s *GetBucketNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.s3.clients.default.naming.bucket_pattern": "{app.name}-{bucketId}",
	})

	name, err := s3.GetBucketName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-my-bucket", name)
}

func (s *GetBucketNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.s3.clients.specific.naming.bucket_pattern": "{app.name}-{bucketId}",
	})

	name, err := s3.GetBucketName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-my-bucket", name)
}

func (s *GetBucketNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.s3.clients.default.naming.bucket_pattern": "{app.name}-{bucketId}",
	})

	name, err := s3.GetBucketName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-my-bucket", name)
}

func (s *GetBucketNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_S3_CLIENTS_SPECIFIC_NAMING_BUCKET_PATTERN": "!nodecode {app.name}-{bucketId}",
	})

	name, err := s3.GetBucketName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-my-bucket", name)
}

func (s *GetBucketNameTestSuite) TestUnknownPlaceholderReturnsError() {
	s.setupConfig(map[string]any{
		"cloud.aws.s3.clients.default.naming.bucket_pattern": "{project}-{bucketId}",
	})

	_, err := s3.GetBucketName(s.config, s.settings)
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder {project}")
}

func (s *GetBucketNameTestSuite) TestMissingTagsOnlyFailsIfPatternRequiresThem() {
	// BucketPattern doesn't use tags, so missing tags should not cause error
	s.settings.Identity.Tags = nil
	s.settings.Identity.Namespace = "{app.env}"
	s.setupConfig(map[string]any{
		"cloud.aws.s3.clients.default.naming.bucket_pattern": "{app.env}-{bucketId}",
	})

	name, err := s3.GetBucketName(s.config, s.settings)
	s.NoError(err)
	s.Equal("test-my-bucket", name)
}

func (s *GetBucketNameTestSuite) TestCustomDelimiter() {
	s.setupConfig(map[string]any{
		"cloud.aws.s3.clients.default.naming.bucket_pattern": "{app.namespace}.{bucketId}",
		"cloud.aws.s3.clients.default.naming.delimiter":      ".",
	})

	name, err := s3.GetBucketName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack.test.gosoline.group.my-bucket", name)
}
