package cfg_test

import (
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errSettingsValidation = errors.New("invalid settings")

type pointerValidatedSettings struct {
	Mode  string `cfg:"mode"`
	Value int    `cfg:"value" validate:"gte=0"`
	calls int
}

func (s *pointerValidatedSettings) Validate() error {
	s.calls++
	if s.Mode == "invalid" {
		return errSettingsValidation
	}

	return nil
}

type valueValidatedSettings struct {
	Value int `cfg:"value"`
}

func (s valueValidatedSettings) Validate() error {
	if s.Value <= 0 {
		return errSettingsValidation
	}

	return nil
}

func TestGetSettings_PointerValidator(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "valid"},
		{name: "invalid", mode: "invalid", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := cfg.New(map[string]any{"settings": map[string]any{"mode": tc.mode, "value": 42}})
			settings, err := cfg.GetSettings[pointerValidatedSettings](config, "settings")
			assert.Equal(t, 42, settings.Value)
			assert.Equal(t, 1, settings.calls)

			if tc.wantErr {
				require.ErrorContains(t, err, "validation failed for key: settings")
				assert.ErrorIs(t, err, errSettingsValidation)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetSettings_ValueValidator(t *testing.T) {
	for _, value := range []int{0, 42} {
		config := cfg.New(map[string]any{"settings": map[string]any{"value": value}})
		settings, err := cfg.GetSettings[valueValidatedSettings](config, "settings")
		assert.Equal(t, value, settings.Value)

		if value == 0 {
			require.ErrorContains(t, err, "validation failed for key: settings")
			assert.ErrorIs(t, err, errSettingsValidation)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestGetSettings_WithoutValidator(t *testing.T) {
	type settings struct {
		Value int `cfg:"value"`
	}
	config := cfg.New(map[string]any{"settings": map[string]any{"value": 42}})
	result, err := cfg.GetSettings[settings](config, "settings")
	require.NoError(t, err)
	assert.Equal(t, 42, result.Value)
}

func TestGetSettings_SkipsValidatorOnUnmarshalError(t *testing.T) {
	for _, value := range []any{"not a number", -1} {
		config := cfg.New(map[string]any{"settings": map[string]any{"value": value}})
		settings, err := cfg.GetSettings[pointerValidatedSettings](config, "settings")
		require.Error(t, err)
		assert.Zero(t, settings.calls)
	}
}
