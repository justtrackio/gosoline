package cfg

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSensitiveConfigKey_MatchesSensitivePatterns(t *testing.T) {
	assert.True(t, isSensitiveConfigKey("password"))
	assert.True(t, isSensitiveConfigKey("db_password"))
	assert.True(t, isSensitiveConfigKey("API_SECRET"))
	assert.True(t, isSensitiveConfigKey("auth_token"))
	assert.True(t, isSensitiveConfigKey("api_key"))
	assert.True(t, isSensitiveConfigKey("database_dsn"))
	assert.True(t, isSensitiveConfigKey("user_credential"))
	assert.True(t, isSensitiveConfigKey("AWS_ACCESS_KEY"))
	// Non-sensitive keys must not be masked.
	assert.False(t, isSensitiveConfigKey("host"))
	assert.False(t, isSensitiveConfigKey("port"))
	assert.False(t, isSensitiveConfigKey("username"))
	assert.False(t, isSensitiveConfigKey("region"))
}

// captureLogger records every logged message.
type captureLogger struct {
	lines []string
}

func (l *captureLogger) Info(_ context.Context, format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func (l *captureLogger) Error(_ context.Context, format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func (l *captureLogger) hasLine(substr string) bool {
	for _, line := range l.lines {
		if strings.Contains(line, substr) {
			return true
		}
	}

	return false
}

// TestDebugConfig_MasksSensitiveKeysInLog verifies that DebugConfig logs "***"
// for sensitive config keys, preventing secret leakage in log output.
func TestDebugConfig_MasksSensitiveKeysInLog(t *testing.T) {
	config := New(map[string]any{
		"db_password": "hunter2",
		"host":        "localhost",
	})

	logger := &captureLogger{}
	err := DebugConfig(context.Background(), config, logger)
	require.NoError(t, err)

	// Sensitive value must be masked in the log.
	assert.False(t, logger.hasLine("hunter2"),
		"real password must not appear in log output")
	assert.True(t, logger.hasLine("***"),
		"masked placeholder must appear in log output")
	// Safe value must be logged as-is.
	assert.True(t, logger.hasLine("localhost"),
		"non-sensitive value must appear in log output")
}

// TestDebugConfig_FingerprintIsEmitted verifies that the fingerprint log line
// is emitted (proving DebugConfig completed) even when sensitive keys are present.
func TestDebugConfig_FingerprintIsEmitted(t *testing.T) {
	config := New(map[string]any{
		"api_secret": "topsecret",
	})

	logger := &captureLogger{}
	err := DebugConfig(context.Background(), config, logger)
	require.NoError(t, err)

	assert.True(t, logger.hasLine("fingerprint"),
		"fingerprint line must be emitted")
	// The real secret must NOT appear in any log line.
	assert.False(t, logger.hasLine("topsecret"),
		"real secret must not appear anywhere in log output")
}
