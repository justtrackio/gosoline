package test

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type configMap map[interface{}]interface{}

func readConfig() configMap {
	bytes, err := ioutil.ReadFile("config.test.yml")

	if err != nil {
		logErr(err, "could not read config.test.yml")
	}

	config := make(configMap)
	err = yaml.Unmarshal(bytes, &config)

	if err != nil {
		logErr(err, "could not unmarshal config.test.yml")
	}

	return config
}

func checkConfigKey(config configMap, name string, key string) {
	if _, ok := config[key]; ok {
		return
	}

	err := errors.New("missing config key")
	msg := fmt.Sprintf("the config key '%s' is missing for component '%s'", key, name)
	logErr(err, msg)
}

func configString(config configMap, name string, key string) string {
	checkConfigKey(config, name, key)

	if value, ok := config[key].(string); ok {
		return value
	}

	err := errors.New("invalid config value type")
	msg := fmt.Sprintf("the type of the value of config key '%s' for component '%s' should be string", key, name)
	logErr(err, msg)

	return ""
}
