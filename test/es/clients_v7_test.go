// +build integration

package es_test

import (
	"github.com/applike/gosoline/pkg/es"
	"github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func init() {
	log.SetFlags(0)
}

func TestNewClient(t *testing.T) {
	defer test.Shutdown()

	test.Boot()

	config, logger := getMocks()

	clientV7 := es.NewClient(config, logger, "test_v7")

	res, err := clientV7.Info()

	assert.NoError(t, err, "can't get Info from ElasticSearch")
	assert.NotEqual(t, res.IsError(), nil, "response with error")
}

func TestGetAwsClient(t *testing.T) {
	config, logger := getMocks()

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
