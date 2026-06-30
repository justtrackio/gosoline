package cli_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInputWithArgs(t *testing.T) {
	testCases := map[string]struct {
		args          []string
		expectArgs    []string
		expectFlags   []cli.InputFlag
		expectErrPart string
	}{
		"arguments only": {
			args:       []string{"command", "sub-command", "arg_1"},
			expectArgs: []string{"command", "sub-command", "arg_1"},
		},
		"positional arguments are not validated": {
			args: []string{
				"command",
				"/tmp/file.json",
				"foo.bar",
				"https://gitlab.com/NAMESPACE/REPO/-/issues/123",
				"argument!",
			},
			expectArgs: []string{
				"command",
				"/tmp/file.json",
				"foo.bar",
				"https://gitlab.com/NAMESPACE/REPO/-/issues/123",
				"argument!",
			},
		},
		"long flags": {
			args: []string{"--name=value", "--env", "dev", "--verbose", "--long-flag=enabled", "--flag_1=true", "--key-val1=foo=bar", "--key-val2", "foo=baz"},
			expectFlags: []cli.InputFlag{
				{Name: "name", Value: "value"},
				{Name: "env", Value: "dev"},
				{Name: "verbose", Value: "true"},
				{Name: "long-flag", Value: "enabled"},
				{Name: "flag_1", Value: "true"},
				{Name: "key-val1", Value: "foo=bar"},
				{Name: "key-val2", Value: "foo=baz"},
			},
		},
		"short flags": {
			args: []string{"-n", "value", "-v", "-k1=foo=bar", "-k2", "foo=baz"},
			expectFlags: []cli.InputFlag{
				{Name: "n", Value: "value"},
				{Name: "v", Value: "true"},
				{Name: "k1", Value: "foo=bar"},
				{Name: "k2", Value: "foo=baz"},
			},
		},
		"combined short flags": {
			args:       []string{"-abc", "value"},
			expectArgs: []string{"value"},
			expectFlags: []cli.InputFlag{
				{Name: "a", Value: "true"},
				{Name: "b", Value: "true"},
				{Name: "c", Value: "true"},
			},
		},
		"repeated flags are preserved": {
			args: []string{"--env", "dev", "--env=prod", "-v", "-v"},
			expectFlags: []cli.InputFlag{
				{Name: "env", Value: "dev"},
				{Name: "env", Value: "prod"},
				{Name: "v", Value: "true"},
				{Name: "v", Value: "true"},
			},
		},
		"separator stops parsing": {
			args:       []string{"--env", "dev", "--", "--verbose", "-v", "argument!", "-"},
			expectArgs: []string{"--verbose", "-v", "argument!", "-"},
			expectFlags: []cli.InputFlag{
				{Name: "env", Value: "dev"},
			},
		},
		"positional arguments after separator are not validated": {
			args: []string{"command", "--", "/tmp/file.json", "foo.bar", "https://gitlab.com/NAMESPACE/REPO/-/issues/123", "argument!"},
			expectArgs: []string{
				"command",
				"/tmp/file.json",
				"foo.bar",
				"https://gitlab.com/NAMESPACE/REPO/-/issues/123",
				"argument!",
			},
		},
		"empty input": {},
		"standalone dash is invalid": {
			args:          []string{"-"},
			expectErrPart: "invalid cli argument",
		},
		"invalid long flag": {
			args:          []string{"--bad.flag"},
			expectErrPart: "invalid cli flag",
		},
		"hyphen only long flag is invalid": {
			args:          []string{"---"},
			expectErrPart: "invalid cli flag",
		},
		"invalid short flag": {
			args:          []string{"-."},
			expectErrPart: "invalid cli flag",
		},
		"invalid combined short flag": {
			args:          []string{"-ab."},
			expectErrPart: "invalid cli flag",
		},
		"flag value is not validated": {
			args: []string{"--path", "/tmp/file.json", "--url=https://example.com/path?query=1"},
			expectFlags: []cli.InputFlag{
				{Name: "path", Value: "/tmp/file.json"},
				{Name: "url", Value: "https://example.com/path?query=1"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			input, err := cli.NewInputWithArgs(tc.args)

			if tc.expectErrPart != "" {
				assert.Nil(t, input)
				assert.ErrorContains(t, err, tc.expectErrPart)

				return
			}

			require.NoError(t, err)

			if tc.expectArgs == nil {
				assert.Empty(t, input.Arguments)
			} else {
				assert.Equal(t, tc.expectArgs, input.Arguments)
			}

			if tc.expectFlags == nil {
				assert.Empty(t, input.Flags)
			} else {
				assert.Equal(t, tc.expectFlags, input.Flags)
			}
		})
	}
}
