package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSecrets_SensitiveTopLevelKeys(t *testing.T) {
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
	got := maskSecrets(m)
	assert.Equal(t, "***", got["db_password"])
	assert.Equal(t, "***", got["api_secret"])
	assert.Equal(t, "***", got["auth_token"])
	assert.Equal(t, "***", got["api_key"])
	assert.Equal(t, "***", got["database_dsn"])
	assert.Equal(t, "***", got["user_credential"])
	assert.Equal(t, "plaintext", got["safe_value"])
	assert.Equal(t, "localhost", got["host"])
}

func TestMaskSecrets_NestedMapIsMaskedRecursively(t *testing.T) {
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
	got := maskSecrets(m)
	db := got["database"].(map[string]any)
	assert.Equal(t, "***", db["password"])
	assert.Equal(t, "localhost", db["host"])
	inner := db["inner"].(map[string]any)
	assert.Equal(t, "***", inner["token"])
	assert.Equal(t, 5432, inner["port"])
}

func TestMaskSecrets_DoesNotMutateOriginal(t *testing.T) {
	m := map[string]any{
		"password": "hunter2",
		"host":     "localhost",
	}
	_ = maskSecrets(m)
	assert.Equal(t, "hunter2", m["password"])
	assert.Equal(t, "localhost", m["host"])
}

func TestIsSensitiveKey_CaseInsensitive(t *testing.T) {
	assert.True(t, isSensitiveKey("PASSWORD"))
	assert.True(t, isSensitiveKey("API_SECRET"))
	assert.True(t, isSensitiveKey("Auth_Token"))
	assert.False(t, isSensitiveKey("host"))
	assert.False(t, isSensitiveKey("port"))
	assert.False(t, isSensitiveKey("username"))
}
