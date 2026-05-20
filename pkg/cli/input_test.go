package cli_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cli"
	"github.com/stretchr/testify/assert"
)

func TestNewInput_ArgumentsOnly(t *testing.T) {
	input := cli.NewInputWithArgs([]string{"command", "sub-command"})

	assert.Equal(t, []string{"command", "sub-command"}, input.Arguments)
	assert.Empty(t, input.Flags)
}

func TestNewInput_LongFlags(t *testing.T) {
	input := cli.NewInputWithArgs([]string{"--name=value", "--env", "dev", "--verbose"})

	assert.Empty(t, input.Arguments)
	assert.Equal(t, map[string]string{
		"env":     "dev",
		"name":    "value",
		"verbose": "true",
	}, input.Flags)
}

func TestNewInput_ShortFlags(t *testing.T) {
	input := cli.NewInputWithArgs([]string{"-n", "value", "-v"})

	assert.Empty(t, input.Arguments)
	assert.Equal(t, map[string]string{
		"n": "value",
		"v": "true",
	}, input.Flags)
}

func TestNewInput_CombinedShortFlags(t *testing.T) {
	input := cli.NewInputWithArgs([]string{"-abc", "value"})

	assert.Equal(t, []string{"value"}, input.Arguments)
	assert.Equal(t, map[string]string{
		"a": "true",
		"b": "true",
		"c": "true",
	}, input.Flags)
}

func TestNewInput_RepeatedFlagsLastValueWins(t *testing.T) {
	input := cli.NewInputWithArgs(([]string{"--env", "dev", "--env=prod", "-v", "-v"}))

	assert.Empty(t, input.Arguments)
	assert.Equal(t, map[string]string{
		"env": "prod",
		"v":   "true",
	}, input.Flags)
}

func TestNewInput_SeparatorStopsFlagParsing(t *testing.T) {
	input := cli.NewInputWithArgs([]string{"--env", "dev", "--", "--verbose", "-v", "argument"})

	assert.Equal(t, []string{"--verbose", "-v", "argument"}, input.Arguments)
	assert.Equal(t, map[string]string{
		"env": "dev",
	}, input.Flags)
}

func TestNewInput_EmptyInput(t *testing.T) {
	input := cli.NewInputWithArgs(nil)

	assert.Empty(t, input.Arguments)
	assert.Empty(t, input.Flags)
}
