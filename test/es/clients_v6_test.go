// +build integration

package es_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/es"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"

	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
)

func getMocks() (cfg.Config, mon.Logger) {
	config := cfg.NewWithDefaultClients("es_test")

	logger := new(monMocks.Logger)

	logger.On("Fatal").Return(nil)
	logger.On("Info", "creating client ", config.GetString("es_test_v6_type"), " for host ", config.GetString("es_test_v6_endpoint")).Return(nil)
	logger.On("Info", "creating client ", config.GetString("es_test_v7_type"), " for host ", config.GetString("es_test_v7_endpoint")).Return(nil)
	logger.On("WithFields", mock.AnythingOfType("map[string]interface {}")).Return(logger)

	return config, logger
}

func TestNewClientV6(t *testing.T) {
	defer test.Shutdown()

	test.Boot()

	config, logger := getMocks()

	clientV6 := es.NewClientV6(config, logger, "test_v6")

	res, err := clientV6.Info()

	assert.NoError(t, err, "can't get Info from ElasticSearch")
	assert.NotEqual(t, res.IsError(), nil, "response with error")
}

func TestGetAwsClientV6(t *testing.T) {
	config, logger := getMocks()

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
