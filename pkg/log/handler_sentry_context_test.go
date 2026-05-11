package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSentrySensitiveKey_MatchesSensitivePatterns(t *testing.T) {
	// All of these must be masked.
	assert.True(t, isSentrySensitiveKey("password"))
	assert.True(t, isSentrySensitiveKey("db_password"))
	assert.True(t, isSentrySensitiveKey("API_SECRET"))
	assert.True(t, isSentrySensitiveKey("auth_token"))
	assert.True(t, isSentrySensitiveKey("api_key"))
	assert.True(t, isSentrySensitiveKey("database_dsn"))
	assert.True(t, isSentrySensitiveKey("user_credential"))
	// Non-sensitive keys must not be masked.
	assert.False(t, isSentrySensitiveKey("host"))
	assert.False(t, isSentrySensitiveKey("port"))
	assert.False(t, isSentrySensitiveKey("username"))
	assert.False(t, isSentrySensitiveKey("region"))
}

func TestMaskSensitiveConfigValues_MasksSensitiveTopLevelKeys(t *testing.T) {
	m := map[string]any{
		"db_password":     "hunter2",
		"api_secret":      "mysecret",
		"auth_token":      "tok123",
		"api_key":         "key123",
		"database_dsn":    "host=localhost",
		"user_credential": "cred",
		"safe_value":      "plaintext",
		"host":            "localhost",
	}
	got := maskSensitiveConfigValues(m)
	assert.Equal(t, "***", got["db_password"])
	assert.Equal(t, "***", got["api_secret"])
	assert.Equal(t, "***", got["auth_token"])
	assert.Equal(t, "***", got["api_key"])
	assert.Equal(t, "***", got["database_dsn"])
	assert.Equal(t, "***", got["user_credential"])
	assert.Equal(t, "plaintext", got["safe_value"])
	assert.Equal(t, "localhost", got["host"])
}

func TestMaskSensitiveConfigValues_MasksNestedMapRecursively(t *testing.T) {
	m := map[string]any{
		"database": map[string]any{
			"password": "secret123",
			"host":     "localhost",
			"inner": map[string]any{
				"token": "tok",
				"port":  5432,
			},
		},
	}
	got := maskSensitiveConfigValues(m)
	db := got["database"].(map[string]any)
	assert.Equal(t, "***", db["password"])
	assert.Equal(t, "localhost", db["host"])
	inner := db["inner"].(map[string]any)
	assert.Equal(t, "***", inner["token"])
	assert.Equal(t, 5432, inner["port"])
}

func TestMaskSensitiveConfigValues_DoesNotMutateOriginal(t *testing.T) {
	m := map[string]any{
		"password": "hunter2",
		"host":     "localhost",
	}
	_ = maskSensitiveConfigValues(m)
	assert.Equal(t, "hunter2", m["password"])
	assert.Equal(t, "localhost", m["host"])
}
