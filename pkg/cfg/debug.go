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

// DebugConfig logs flattened settings and their fingerprint with sensitive values redacted.
// Redaction does not change the underlying configuration.
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
		if isSensitiveConfigKey(key) {
			value = "[REDACTED]"
		}

		hashValues[i] = fmt.Sprintf("%v=%v", key, value)
		logger.Info(ctx, "cfg %s", hashValues[i])
	}

	hashString := strings.Join(hashValues, ";")
	hashBytes := md5.Sum([]byte(hashString))

	logger.Info(ctx, "cfg fingerprint: %x", hashBytes)

	return nil
}

func isSensitiveConfigKey(key string) bool {
	for _, needle := range [...]string{"password", "secret", "token", "key", "dsn", "credential"} {
		for i := 0; i+len(needle) <= len(key); i++ {
			if strings.EqualFold(key[i:i+len(needle)], needle) {
				return true
			}
		}
	}

	return false
}
