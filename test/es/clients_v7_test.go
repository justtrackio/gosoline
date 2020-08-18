// +build integration

package es_test

import (
	"github.com/applike/gosoline/pkg/es"
	"github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewClient(t *testing.T) {
	configFilePath := "config-v7.test.yml"

	mocks, err := test.Boot(configFilePath)
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	clientV7 := mocks.ProvideElasticsearchV6Client("metrics_v7", "default")

	res, err := clientV7.Info()

	assert.NoError(t, err, "can't get Info from ElasticSearch")
	assert.NotEqual(t, res.IsError(), nil, "response with error")
}

func TestGetAwsClient(t *testing.T) {
	config, logger := getMocks("config-v7.test.yml")

	endpointKey := "es_test_v7_aws_endpoint"
	if !config.IsSet(endpointKey) {
		t.Skipf("%s missed in config", endpointKey)
		return
	}

	clientAwsV7, err := es.GetAwsClientV6(logger, config.GetString(endpointKey))
	assert.NoError(t, err, "can't get Info from ElasticSearch")

	res, err := clientAwsV7.Info()
	assert.NoError(t, err, "can't get Info from ElasticSearch at AWS")
	assert.NotEqual(t, res.IsError(), nil, "response with error")
}
