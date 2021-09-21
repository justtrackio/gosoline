package cfg

import (
	"io/ioutil"

	"github.com/justtrackio/gosoline/pkg/encoding/yaml"
	"github.com/pkg/errors"
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

	return cfg.mergeMsi(".", settings)
}
