package cfg

import (
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/encoding/yaml"
)

func readConfigFromFile(cfg *config, filePath string, fileType string) error {
	if filePath == "" {
		return nil
	}

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		wd, errWd := os.Getwd()
		if errWd != nil {
			wd = errWd.Error()
		}

		return fmt.Errorf("can not read config file %s in directory %s: %w", filePath, wd, err)
	}

	settings := make(map[string]interface{})
	err = yaml.Unmarshal(bytes, &settings)
	if err != nil {
		return fmt.Errorf("can not unmarshal config file %s: %w", filePath, err)
	}

	return cfg.mergeMsi(".", settings)
}
