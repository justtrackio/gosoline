package cfg

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"github.com/jeremywohl/flatten"
	"golang.org/x/exp/maps"
)

func DebugConfig(config Config, logger Logger) error {
	settings := config.AllSettings()
	flattened, err := flatten.Flatten(settings, "", flatten.DotStyle)
	if err != nil {
		return fmt.Errorf("can not flatten config settings")
	}

	hashValues := make([]string, len(flattened))
	keys := maps.Keys(flattened)
	sort.Strings(keys)

	for i, key := range keys {
		hashValues[i] = fmt.Sprintf("%v=%v", key, flattened[key])
		logger.Info("cfg %s", hashValues[i])
	}

	hashString := strings.Join(hashValues, ";")
	hashBytes := md5.Sum([]byte(hashString))

	logger.Info("cfg fingerprint: %x", hashBytes)

	return nil
}
