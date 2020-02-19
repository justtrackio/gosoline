// +build integration

package es_test

import (
	"github.com/applike/gosoline/pkg/es"
	"github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewClientV6(t *testing.T) {
	defer test.Shutdown()

	configFilePath := "config-v6.test.yml"

	test.Boot(configFilePath)

	config, logger := getMocks(configFilePath)

	clientV6 := es.NewClientV6(config, logger, "test_v6")

	res, err := clientV6.Info()

	assert.NoError(t, err, "can't get Info from ElasticSearch")
	assert.NotEqual(t, res.IsError(), nil, "response with error")
}

func TestGetAwsClientV6(t *testing.T) {
	config, logger := getMocks("config-v6.test.yml")

	endpointKey := "es_test_v6_aws_endpoint"
	if !config.IsSet(endpointKey) {
		t.Skipf("%s missed in config", endpointKey)
		return
	}

	clientAwsV6, err := es.GetAwsClientV6(logger, config.GetString(endpointKey))
	assert.NoError(t, err, "can't get Info from ElasticSearch")

	res, err := clientAwsV6.Info()
	assert.NoError(t, err, "can't get Info from ElasticSearch at AWS")
	assert.NotEqual(t, res.IsError(), nil, "response with error")
}
