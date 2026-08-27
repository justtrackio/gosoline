package email

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/require"
)

func TestDefaultMaxEncodedMessageSizeMatchesConfig(t *testing.T) {
	t.Parallel()

	config := cfg.New(map[string]any{
		"email": map[string]any{
			"test": map[string]any{"from_address": "sender@example.com"},
		},
	})

	settings := &emailSettings{}
	require.NoError(t, config.UnmarshalKey("email.test", settings))
	require.Equal(t, defaultMaxEncodedMessageSize, settings.MaxEncodedMessageSize)
	require.NoError(t, settings.validate())
}

func TestEmailSettingsValidate(t *testing.T) {
	t.Parallel()

	require.EqualError(t, emailSettings{MaxEncodedMessageSize: 0}.validate(), "max encoded message size must be positive, got 0")
	require.EqualError(t, emailSettings{MaxEncodedMessageSize: -1}.validate(), "max encoded message size must be positive, got -1")
	require.NoError(t, emailSettings{MaxEncodedMessageSize: 1}.validate())
}
