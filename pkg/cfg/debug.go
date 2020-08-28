package cfg

import (
	"fmt"
	"github.com/jeremywohl/flatten"
	"github.com/thoas/go-funk"
	"sort"
)

func DebugConfig(config Config, logger Logger) error {
	settings := config.AllSettings()
	flattened, err := flatten.Flatten(settings, "", flatten.DotStyle)

	if err != nil {
		return fmt.Errorf("can not flatten config settings")
	}

	keys := funk.Keys(flattened).([]string)
	sort.Strings(keys)

	for _, key := range keys {
		logger.Infof("cfg %v=%v", key, flattened[key])
	}

	return nil
}
