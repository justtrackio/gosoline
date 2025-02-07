package cfg

import (
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/encoding/yaml"
)

type unmarshaller func(data []byte, v any) error

var unmarshallers = map[string]unmarshaller{
	"json": json.Unmarshal,
	"yaml": yaml.Unmarshal,
	"yml":  yaml.Unmarshal,
}

func readConfigFromBytes(cfg *config, bytes []byte, format string) error {
	var ok bool
	var err error
	var unmarshal unmarshaller

	settings := make(map[string]any)

	if unmarshal, ok = unmarshallers[format]; !ok {
		return fmt.Errorf("unknown format: %s", format)
	}

	if err = unmarshal(bytes, &settings); err != nil {
		return fmt.Errorf("can not unmarshal config bytes of format %q: %w", format, err)
	}

	return cfg.mergeMsi(".", settings)
}

func readConfigFromFile(cfg *config, filePath string, fileType string) error {
	if filePath == "" {
		return nil
	}

	var err, errWd error
	var bytes []byte
	var wd string

	if bytes, err = os.ReadFile(filePath); err != nil {
		if wd, errWd = os.Getwd(); errWd != nil {
			wd = errWd.Error()
		}

		return fmt.Errorf("can not read config file %q in directory %q: %w", filePath, wd, err)
	}

	if err = readConfigFromBytes(cfg, bytes, fileType); err != nil {
		return fmt.Errorf("can not unmarshal config file %q: %w", filePath, err)
	}

	return nil
}
