package blob_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/suite"
)

func TestReadStoreSettingsTestSuite(t *testing.T) {
	suite.Run(t, new(ReadStoreSettingsTestSuite))
}

type ReadStoreSettingsTestSuite struct {
	suite.Suite

	config cfg.GosoConf
}

func (s *ReadStoreSettingsTestSuite) SetupTest() {
	s.config = cfg.New()
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"env":       "test",
			"name":      "blob-app",
			"namespace": "{app.tags.project}.{app.env}.{app.tags.family}",
			"tags": map[string]any{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "grp",
			},
		},
	}))
	s.Require().NoError(err, "base config creation should not fail")
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_WithDefaults() {
	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("my_store", settings.BucketId)
	s.Equal("justtrack-test-gosoline", settings.Bucket)
	s.Equal("eu-central-1", settings.Region)
	s.Equal("default", settings.ClientName)
	s.Equal("", settings.Prefix)
	s.Equal("test", settings.Env)
	s.Equal("blob-app", settings.Name)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_WithExplicitBucket() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.my_store": map[string]any{
			"bucket": "my-explicit-bucket",
		},
	}))
	s.Require().NoError(err, "config update should not fail")

	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("my_store", settings.BucketId)
	s.Equal("my-explicit-bucket", settings.Bucket)
	s.Equal("eu-central-1", settings.Region)
	s.Equal("default", settings.ClientName)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_WithExplicitRegion() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.my_store": map[string]any{
			"region": "us-west-2",
		},
	}))
	s.Require().NoError(err, "config update should not fail")

	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("my_store", settings.BucketId)
	s.Equal("justtrack-test-gosoline", settings.Bucket)
	s.Equal("us-west-2", settings.Region)
	s.Equal("default", settings.ClientName)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_WithCustomClientName() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.my_store": map[string]any{
			"client_name": "custom_client",
		},
		"cloud.aws.s3.clients.custom_client": map[string]any{
			"region": "ap-southeast-1",
		},
	}))
	s.Require().NoError(err, "config update should not fail")

	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("my_store", settings.BucketId)
	s.Equal("justtrack-test-gosoline", settings.Bucket)
	s.Equal("ap-southeast-1", settings.Region)
	s.Equal("custom_client", settings.ClientName)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_WithPrefix() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.my_store": map[string]any{
			"prefix": "uploads/",
		},
	}))
	s.Require().NoError(err, "config update should not fail")

	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("my_store", settings.BucketId)
	s.Equal("justtrack-test-gosoline", settings.Bucket)
	s.Equal("eu-central-1", settings.Region)
	s.Equal("default", settings.ClientName)
	s.Equal("uploads/", settings.Prefix)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_WithBucketPattern() {
	config := cfg.New()
	err := config.Option(cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"env":  "prod",
			"name": "uploader",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
		"cloud.aws.s3.clients.default.naming.bucket_pattern": "{app.env}-{app.name}-{bucketId}",
	}))
	s.Require().NoError(err, "config creation should not fail")

	settings, err := blob.ReadStoreSettings(config, "images")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("images", settings.BucketId)
	s.Equal("prod-uploader-images", settings.Bucket)
	s.Equal("eu-central-1", settings.Region)
	s.Equal("default", settings.ClientName)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_WithClientSpecificBucketPattern() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.my_store": map[string]any{
			"client_name": "special_client",
		},
		"cloud.aws.s3.clients.special_client.naming.bucket_pattern": "{app.tags.project}-{bucketId}-bucket",
	}))
	s.Require().NoError(err, "config update should not fail")

	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("my_store", settings.BucketId)
	s.Equal("justtrack-my_store-bucket", settings.Bucket)
	s.Equal("eu-central-1", settings.Region)
	s.Equal("special_client", settings.ClientName)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_WithDefaultsFromBlobDefault() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.default": map[string]any{
			"client_name": "shared_client",
			"prefix":      "default/",
		},
		"blob.my_store": map[string]any{
			"bucket": "my-bucket",
		},
	}))
	s.Require().NoError(err, "config update should not fail")

	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("my_store", settings.BucketId)
	s.Equal("my-bucket", settings.Bucket)
	s.Equal("eu-central-1", settings.Region)
	s.Equal("shared_client", settings.ClientName)
	s.Equal("default/", settings.Prefix)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_OverrideDefaultsFromBlobDefault() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.default": map[string]any{
			"client_name": "shared_client",
			"prefix":      "default/",
		},
		"blob.my_store": map[string]any{
			"client_name": "override_client",
			"prefix":      "custom/",
		},
		"cloud.aws.s3.clients.override_client": map[string]any{
			"region": "us-east-1",
		},
	}))
	s.Require().NoError(err, "config update should not fail")

	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("my_store", settings.BucketId)
	s.Equal("justtrack-test-gosoline", settings.Bucket)
	s.Equal("us-east-1", settings.Region)
	s.Equal("override_client", settings.ClientName)
	s.Equal("custom/", settings.Prefix)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_AllFieldsPopulated() {
	config := cfg.New()
	err := config.Option(cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"env":  "production",
			"name": "blob-service",
			"tags": map[string]any{
				"project": "enterprise",
				"family":  "storage",
				"group":   "backend",
			},
		},
		"blob.full_store": map[string]any{
			"bucket":      "custom-bucket-name",
			"region":      "us-west-1",
			"client_name": "production_client",
			"prefix":      "data/v1/",
		},
	}))
	s.Require().NoError(err, "config creation should not fail")

	settings, err := blob.ReadStoreSettings(config, "full_store")
	s.NoError(err, "ReadStoreSettings should not return an error")
	s.NotNil(settings, "settings should not be nil")

	s.Equal("full_store", settings.BucketId)
	s.Equal("custom-bucket-name", settings.Bucket)
	s.Equal("us-west-1", settings.Region)
	s.Equal("production_client", settings.ClientName)
	s.Equal("data/v1/", settings.Prefix)
	s.Equal("production", settings.Env)
	s.Equal("blob-service", settings.Name)
}

func (s *ReadStoreSettingsTestSuite) TestReadStoreSettings_GettersWork() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.my_store": map[string]any{
			"client_name": "test_client",
		},
	}))
	s.Require().NoError(err, "config update should not fail")

	settings, err := blob.ReadStoreSettings(s.config, "my_store")
	s.Require().NoError(err, "ReadStoreSettings should not return an error")

	// Test getter methods
	s.Equal("my_store", settings.GetBucketId())
	s.Equal("test_client", settings.GetClientName())

	identity := settings.GetIdentity()
	s.Equal("test", identity.Env)
	s.Equal("blob-app", identity.Name)
}
