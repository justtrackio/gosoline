package cfg

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"github.com/jeremywohl/flatten"
	"github.com/justtrackio/gosoline/pkg/funk"
)

var debugSensitivePatterns = []string{
	"password", "secret", "token", "key", "dsn", "credential",
}

func isSensitiveConfigKey(key string) bool {
	lower := strings.ToLower(key)
	for _, pattern := range debugSensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

func DebugConfig(ctx context.Context, config Config, logger Logger) error {
	settings := config.AllSettings()
	flattened, err := flatten.Flatten(settings, "", flatten.DotStyle)
	if err != nil {
		return fmt.Errorf("can not flatten config settings")
	}

	hashValues := make([]string, len(flattened))
	keys := funk.Keys(flattened)
	sort.Strings(keys)

	for i, key := range keys {
		value := flattened[key]
		// Use the real value for the fingerprint so config changes are always detectable.
		hashValues[i] = fmt.Sprintf("%v=%v", key, value)
		// Mask sensitive values in the log output.
		if isSensitiveConfigKey(key) {
			value = "***"
		}
		logger.Info(ctx, "cfg %v=%v", key, value)
	}

	hashString := strings.Join(hashValues, ";")
	hashBytes := md5.Sum([]byte(hashString))

	logger.Info(ctx, "cfg fingerprint: %x", hashBytes)

	return nil
}
