package s3_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/stretchr/testify/assert"
)

func TestResolveDefaultEndpoint(t *testing.T) {
	config := createConfig(t, map[string]interface{}{})

	endpoint, err := s3.ResolveEndpoint(config, "default")
	assert.NoError(t, err, "there should be no error resolving the endpoint")
	assert.Equal(t, "http://localhost:4566", endpoint.URL)
}

func TestResolveAwsEndpoint(t *testing.T) {
	config := createConfig(t, map[string]interface{}{
		"cloud.aws.defaults.endpoint": "",
	})

	endpoint, err := s3.ResolveEndpoint(config, "default")
	assert.NoError(t, err, "there should be no error resolving the endpoint")
	assert.Equal(t, "https://s3.eu-central-1.amazonaws.com", endpoint.URL)
}

func createConfig(t *testing.T, settings map[string]interface{}) cfg.Config {
	config := cfg.New()
	err := config.Option(cfg.WithConfigMap(settings))
	assert.NoError(t, err, "there should be no error on config create")

	return config
}
