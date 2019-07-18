package test

import (
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type configInput map[string]interface{}

type testConfig struct {
	Mocks map[string]configInput `mapstructure:"mocks"`
}

func readConfig() *testConfig {
	bytes, err := ioutil.ReadFile("config.test.yml")

	if err != nil {
		logErr(err, "could not read config.test.yml")
	}

	input := make(configInput)
	err = yaml.Unmarshal(bytes, &input)

	if err != nil {
		logErr(err, "could not unmarshal config.test.yml")
	}

	config := &testConfig{}
	unmarshalConfig(input, config)

	return config
}

func unmarshalConfig(input interface{}, output interface{}) {
	err = mapstructure.WeakDecode(input, output)

	if err != nil {
		logErr(err, "can not decode config for test")
	}
}
