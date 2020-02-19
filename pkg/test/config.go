package test

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type configInput map[string]interface{}

type testConfig struct {
	Mocks []configInput `mapstructure:"mocks"`
}

func readConfig(configFilename string) *testConfig {
	bytes, err := ioutil.ReadFile(configFilename)

	if err != nil {
		logErr(err, fmt.Sprintf("could not read %s", configFilename))
	}

	input := make(configInput)
	err = yaml.Unmarshal(bytes, &input)

	if err != nil {
		logErr(err, fmt.Sprintf("could not unmarshal %s", configFilename))
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
