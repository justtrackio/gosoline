package cfg

import (
	"context"
	"crypto/md5"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jeremywohl/flatten"
	"github.com/justtrackio/gosoline/pkg/funk"
)

// DebugConfig logs flattened settings and an MD5 fingerprint of the same redacted
// entries. It recognizes case-insensitive password/secret/token/key/dsn/credential
// substrings in the leaf key, ignoring trailing ASCII numeric list indices.
// Exact api_auth_basic_users and Authorization path segments mask their subtrees.
// Independently, rendered values or embedded ASCII token runs of at least 12 bytes
// are masked when byte Shannon entropy is >= min(4.5, log2(byte length)-0.1).
// This heuristic requires no opt-in: public high-entropy values may be masked,
// while short, low-entropy or hex-only secrets and unknown credential containers
// may remain visible. It is not a secrecy guarantee. Getters, stored settings and
// decoded values remain unchanged.
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
			hashValues[i] = key + "=[REDACTED]"
		} else if text, ok := value.(string); ok && isSensitiveConfigValue(text) {
			hashValues[i] = key + "=[REDACTED]"
		} else {
			entry := fmt.Sprintf("%v=%v", key, value)
			if _, ok := value.(string); !ok && isSensitiveConfigValue(entry[len(key)+1:]) {
				entry = key + "=[REDACTED]"
			}
			hashValues[i] = entry
		}
		logger.Info(ctx, "cfg %s", hashValues[i])
	}

	hashString := strings.Join(hashValues, ";")
	hashBytes := md5.Sum([]byte(hashString))

	logger.Info(ctx, "cfg fingerprint: %x", hashBytes)

	return nil
}

func isSensitiveConfigKey(key string) bool {
	leafStart, leafEnd := 0, len(key)

	for start := 0; start <= len(key); {
		end := start
		for end < len(key) && key[end] != '.' {
			end++
		}
		segment := key[start:end]
		if strings.EqualFold(segment, "api_auth_basic_users") ||
			strings.EqualFold(segment, "Authorization") {
			return true
		}
		numeric := segment != ""
		for i := 0; i < len(segment) && numeric; i++ {
			numeric = segment[i] >= '0' && segment[i] <= '9'
		}
		if !numeric {
			leafStart, leafEnd = start, end
		}
		start = end + 1
	}
	leaf := key[leafStart:leafEnd]
	for _, needle := range [...]string{"password", "secret", "token", "key", "dsn", "credential"} {
		for i := 0; i+len(needle) <= len(leaf); i++ {
			if strings.EqualFold(leaf[i:i+len(needle)], needle) {
				return true
			}
		}
	}

	return false
}

const (
	minConfigEntropyBytes = 12
	configEntropySlack    = 0.1
	maxConfigEntropy      = 4.5
)

// isSensitiveConfigValue examines only the rendered value, never its key.
// Separators such as '=' and '.' keep assignments and JWT sections independent.
func isSensitiveConfigValue(value string) bool {
	if len(value) < minConfigEntropyBytes {
		return false
	}

	if hasConfigValueEntropy(value) {
		return true
	}

	for start := 0; start < len(value); {
		for start < len(value) && !isConfigTokenByte(value[start]) {
			start++
		}
		end := start
		for end < len(value) && isConfigTokenByte(value[end]) {
			end++
		}
		if end-start >= minConfigEntropyBytes && (start != 0 || end != len(value)) && hasConfigValueEntropy(value[start:end]) {
			return true
		}
		start = end + 1
	}

	return false
}

func isConfigTokenByte(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' ||
		b >= '0' && b <= '9' || b == '_' || b == '-' || b == '+' || b == '/'
}

func hasConfigValueEntropy(value string) bool {
	if len(value) < minConfigEntropyBytes {
		return false
	}

	var counts [256]int
	for i := 0; i < len(value); i++ {
		counts[value[i]]++
	}

	var weightedCounts float64
	for i := range counts {
		count := counts[i]
		if count > 1 {
			weightedCounts += float64(count) * math.Log2(float64(count))
		}
	}
	logLength := math.Log2(float64(len(value)))
	entropy := logLength - weightedCounts/float64(len(value))

	return entropy >= math.Min(maxConfigEntropy, logLength-configEntropySlack)
}
