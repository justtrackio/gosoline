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

// DebugConfig prints the effective config values to the logger while redacting sensitive or secret values.
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
		entry, redact := shouldRedact(key, flattened[key])
		if redact {
			entry = key + "=[REDACTED]"
		}
		hashValues[i] = entry
		logger.Info(ctx, "cfg %s", hashValues[i])
	}

	hashString := strings.Join(hashValues, ";")
	hashBytes := md5.Sum([]byte(hashString))

	logger.Info(ctx, "cfg fingerprint: %x", hashBytes)

	return nil
}

// shouldRedact checks keys before rendering values, so known sensitive values are
// never formatted. It returns the full entry for reuse without formatting twice.
func shouldRedact(key string, value any) (entry string, redact bool) {
	if isSensitiveConfigKey(key) {
		return "", true
	}

	if text, ok := value.(string); ok {
		if isSensitiveConfigValue(text) {
			return "", true
		}

		return key + "=" + text, false
	}

	entry = fmt.Sprintf("%v=%v", key, value)

	return entry, isSensitiveConfigValue(entry[len(key)+1:])
}

// isSensitiveConfigKey matches six case-insensitive substrings in the leaf or its
// list owner, ignoring trailing ASCII numeric indices. Only exact Authorization
// and api_auth_basic_users segments redact entire subtrees, not unknown containers.
func isSensitiveConfigKey(key string) bool {
	leaf := key
	for segment := range strings.SplitSeq(key, ".") {
		if strings.EqualFold(segment, "api_auth_basic_users") ||
			strings.EqualFold(segment, "Authorization") {
			return true
		}
		if !isASCIINumericIndex(segment) {
			leaf = segment
		}
	}
	leaf = strings.ToLower(leaf)
	for _, needle := range [...]string{"password", "secret", "token", "key", "dsn", "credential"} {
		if strings.Contains(leaf, needle) {
			return true
		}
	}

	return false
}

// isASCIINumericIndex recognizes a nonempty sequence of ASCII digits, equivalent
// to ^\d+$ without interpreting Unicode digits as flattened list indices.
func isASCIINumericIndex(segment string) bool {
	return segment != "" && strings.IndexFunc(segment, func(r rune) bool {
		return r < '0' || r > '9'
	}) == -1
}

const (
	minConfigEntropyBytes = 12
	configEntropySlack    = 0.1
	maxConfigEntropy      = 4.5
)

// isSensitiveConfigValue checks the rendered value and maximal ASCII token runs.
// Padding '=' and JWT dots separate runs; the already-checked whole value is skipped.
// It may mask public high-entropy values and miss short or low-entropy secrets.
func isSensitiveConfigValue(value string) bool {
	if len(value) < minConfigEntropyBytes {
		return false
	}

	if hasConfigValueEntropy(value) {
		return true
	}

	for tokenStart := 0; tokenStart < len(value); {
		for tokenStart < len(value) && !isBase64TokenByte(value[tokenStart]) {
			tokenStart++
		}
		tokenEnd := tokenStart
		for tokenEnd < len(value) && isBase64TokenByte(value[tokenEnd]) {
			tokenEnd++
		}
		isWholeValue := tokenStart == 0 && tokenEnd == len(value)
		if !isWholeValue && hasConfigValueEntropy(value[tokenStart:tokenEnd]) {
			return true
		}
		tokenStart = tokenEnd + 1
	}

	return false
}

// isBase64TokenByte recognizes standard and URL-safe Base64 alphabet membership,
// excluding '=' padding. It does not validate Base64 encoding.
func isBase64TokenByte(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' ||
		b >= '0' && b <= '9' || b == '_' || b == '-' || b == '+' || b == '/'
}

// hasConfigValueEntropy requires at least 12 bytes and byte Shannon entropy
// >= min(4.5, log2(byte length)-0.1), considering the value only, never its key.
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
