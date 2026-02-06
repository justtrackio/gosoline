package blob_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/suite"
)

func TestUrlBuilderTestSuite(t *testing.T) {
	suite.Run(t, new(UrlBuilderTestSuite))
}

type UrlBuilderTestSuite struct {
	suite.Suite

	config cfg.GosoConf
}

func (s *UrlBuilderTestSuite) SetupTest() {
	s.config = cfg.New()
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"env":       "test",
			"name":      "uploader",
			"namespace": "{app.tags.project}.{app.env}.{app.tags.family}",
			"tags": map[string]any{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "grp",
			},
		},
	}))

	s.NoError(err, "there should be no error on config create")
}

func (s *UrlBuilderTestSuite) TestLocalstack() {
	builder, err := blob.NewUrlBuilder(s.config, "my_store")
	s.NoError(err, "there should be no error on builder create")

	url, err := builder.GetAbsoluteUrl("my_file.bin")
	s.NoError(err, "there should be no error on GetAbsoluteUrl")
	s.Equal("http://localhost:4566/justtrack-test-gosoline/my_file.bin", url)
}

func (s *UrlBuilderTestSuite) TestAws() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"cloud.aws.defaults.endpoint":               "",
		"cloud.aws.s3.clients.default.usePathStyle": false,
	}))

	s.NoError(err, "there should be no error on config create")

	builder, err := blob.NewUrlBuilder(s.config, "my_store")
	s.NoError(err, "there should be no error on builder create")

	url, err := builder.GetAbsoluteUrl("my_file.bin")
	s.NoError(err, "there should be no error on GetAbsoluteUrl")
	s.Equal("https://justtrack-test-gosoline.s3.eu-central-1.amazonaws.com/my_file.bin", url)
}

func (s *UrlBuilderTestSuite) TestWithCustomBucket() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.my_store": map[string]any{
			"bucket": "my-custom-bucket",
		},
		"cloud.aws.defaults.endpoint":               "",
		"cloud.aws.s3.clients.default.usePathStyle": false,
	}))

	s.NoError(err, "there should be no error on config create")

	builder, err := blob.NewUrlBuilder(s.config, "my_store")
	s.NoError(err, "there should be no error on builder create")

	url, err := builder.GetAbsoluteUrl("my_file.bin")
	s.NoError(err, "there should be no error on GetAbsoluteUrl")
	s.Equal("https://my-custom-bucket.s3.eu-central-1.amazonaws.com/my_file.bin", url)
}

func (s *UrlBuilderTestSuite) TestWithBucketPattern() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"blob.my_store.client_name":                           "my_store",
		"cloud.aws.s3.clients.my_store.naming.bucket_pattern": "{app.env}-{app.name}-bucket",
		"cloud.aws.defaults.endpoint":                         "",
		"cloud.aws.s3.clients.default.usePathStyle":           false,
	}))

	s.NoError(err, "there should be no error on config create")

	builder, err := blob.NewUrlBuilder(s.config, "my_store")
	s.NoError(err, "there should be no error on builder create")

	url, err := builder.GetAbsoluteUrl("my_file.bin")
	s.NoError(err, "there should be no error on GetAbsoluteUrl")
	s.Equal("https://test-uploader-bucket.s3.eu-central-1.amazonaws.com/my_file.bin", url)
}

func (s *UrlBuilderTestSuite) TestWithDefaultBucketPattern() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"cloud.aws.s3.clients.default.naming.bucket_pattern": "{app.env}-{app.name}-default",
		"cloud.aws.defaults.endpoint":                        "",
		"cloud.aws.s3.clients.default.usePathStyle":          false,
	}))

	s.NoError(err, "there should be no error on config create")

	builder, err := blob.NewUrlBuilder(s.config, "my_store")
	s.NoError(err, "there should be no error on builder create")

	url, err := builder.GetAbsoluteUrl("my_file.bin")
	s.NoError(err, "there should be no error on GetAbsoluteUrl")
	s.Equal("https://test-uploader-default.s3.eu-central-1.amazonaws.com/my_file.bin", url)
}
