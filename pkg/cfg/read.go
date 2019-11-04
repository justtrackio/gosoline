package cfg

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

func readConfigFromFile(cfg *config, filePath string, fileType string) error {
	if filePath == "" {
		return nil
	}

	bytes, err := ioutil.ReadFile(filePath)

	if err != nil {
		return errors.Wrapf(err, "can not read config file %s", filePath)
	}

	settings := make(map[string]interface{})
	err = yaml.Unmarshal(bytes, &settings)

	if err != nil {
		return errors.Wrapf(err, "can not unmarshal config file %s", filePath)
	}

	return cfg.mergeSettings(settings)
}
