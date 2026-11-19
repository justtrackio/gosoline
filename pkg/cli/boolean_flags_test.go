package cli_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBooleanInput(t *testing.T) {
	flags := []cli.Flag{
		{Long: "worktrees", Short: "w", Kind: cli.FlagKindBool},
		{Long: "quiet", Short: "q", Kind: cli.FlagKindBool},
		{Long: "label", Short: "l", Kind: cli.FlagKindString},
	}
	for _, tc := range []struct {
		name      string
		args      []string
		wantArgs  []string
		wantFlags []cli.InputFlag
	}{
		{"before positionals", []string{"clone", "--worktrees", "prh:marketing", "wt-test"}, []string{"clone", "prh:marketing", "wt-test"}, []cli.InputFlag{{Name: "worktrees", Value: "true"}}},
		{"before command", []string{"--worktrees", "clone", "topic"}, []string{"clone", "topic"}, []cli.InputFlag{{Name: "worktrees", Value: "true"}}},
		{"short switch", []string{"-w", "clone"}, []string{"clone"}, []cli.InputFlag{{Name: "w", Value: "true"}}},
		{"explicit false", []string{"--worktrees=false", "clone"}, []string{"clone"}, []cli.InputFlag{{Name: "worktrees", Value: "false"}}},
		{"short explicit false", []string{"-w=false", "clone"}, []string{"clone"}, []cli.InputFlag{{Name: "w", Value: "false"}}},
		{"false is positional", []string{"--worktrees", "false"}, []string{"false"}, []cli.InputFlag{{Name: "worktrees", Value: "true"}}},
		{"combined switches", []string{"-wq", "clone"}, []string{"clone"}, []cli.InputFlag{{Name: "w", Value: "true"}, {Name: "q", Value: "true"}}},
		{"string flag", []string{"--label", "bug", "--worktrees", "clone"}, []string{"clone"}, []cli.InputFlag{{Name: "label", Value: "bug"}, {Name: "worktrees", Value: "true"}}},
		{"separator", []string{"--worktrees", "--", "--quiet", "-w"}, []string{"--quiet", "-w"}, []cli.InputFlag{{Name: "worktrees", Value: "true"}}},
		{"help before command", []string{"--help", "clone"}, []string{"clone"}, []cli.InputFlag{{Name: "help", Value: "true"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, err := cli.NewInputWithArgs(tc.args, flags...)
			require.NoError(t, err)
			assert.Equal(t, tc.wantArgs, input.Arguments)
			assert.Equal(t, tc.wantFlags, input.Flags)
		})
	}
}

func TestBooleanBlueprint(t *testing.T) {
	for _, tc := range []struct {
		name         string
		args         []string
		defaultValue string
		want         bool
		wantError    bool
	}{
		{name: "absent defaults to false"},
		{name: "default true", defaultValue: "true", want: true},
		{name: "bare switch", args: []string{"--enabled"}, want: true},
		{name: "explicit false", args: []string{"--enabled=false"}, defaultValue: "true"},
		{name: "last occurrence wins", args: []string{"--enabled=false", "--enabled"}, want: true},
		{name: "invalid value", args: []string{"--enabled=wrong"}, wantError: true},
		{name: "empty value", args: []string{"--enabled="}, wantError: true},
		{name: "invalid default", defaultValue: "wrong", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flag := cli.Flag{Long: "enabled", Kind: cli.FlagKindBool, Default: tc.defaultValue, CfgKey: "feature.enabled"}
			router := cli.NewRouter(nil)
			router.Cmd(cli.Cmd{Name: "run", Flags: []cli.Flag{flag}})
			input, err := cli.NewInputWithArgs(append([]string{"run"}, tc.args...), flag)
			require.NoError(t, err)
			blueprint, err := cli.NewBlueprint(router, input)
			if tc.wantError {
				require.ErrorContains(t, err, "invalid boolean value")

				return
			}
			require.NoError(t, err)
			require.Len(t, blueprint.Flags, 1)
			assert.Equal(t, tc.want, blueprint.Flags[0].Value)
			assert.Equal(t, "feature.enabled", blueprint.Flags[0].CustomKey)
		})
	}
}

func TestRouterBooleanFlagScopes(t *testing.T) {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "clone", Flags: []cli.Flag{{Long: "worktrees", Kind: cli.FlagKindBool}}})
	issues := router.Group(cli.Group{Name: "issues", Flags: []cli.Flag{{Long: "verbose", Kind: cli.FlagKindBool}}})
	issues.DefaultCmd(cli.Cmd{Name: "issues", Flags: []cli.Flag{{Long: "list", Short: "l", Kind: cli.FlagKindBool}}})
	issues.Cmd(cli.Cmd{Name: "create", Flags: []cli.Flag{{Long: "labels", Short: "l", Kind: cli.FlagKindList}}})
	globals := []cli.Flag{{Long: "yes", Short: "y", Kind: cli.FlagKindBool}, {Long: "config", Short: "c"}}

	for _, tc := range []struct {
		name     string
		args     []string
		wantCmd  []string
		wantArgs []string
		key      string
		value    any
	}{
		{"original clone", []string{"clone", "--worktrees", "prh:marketing", "wt-test"}, []string{"clone"}, []string{"prh:marketing", "wt-test"}, "worktrees", true},
		{"command flag before command", []string{"--worktrees", "clone", "topic"}, []string{"clone"}, []string{"topic"}, "worktrees", true},
		{"global before command", []string{"--yes", "clone", "topic"}, []string{"clone"}, []string{"topic"}, "yes", true},
		{"default command alias", []string{"issues", "-l"}, []string{"issues"}, []string{}, "list", true},
		{"alias before default command", []string{"-l", "issues"}, []string{"issues"}, []string{}, "list", true},
		{"string alias", []string{"issues", "create", "-l", "bug", "-l", "backend"}, []string{"issues", "create"}, []string{}, "labels", []string{"bug", "backend"}},
		{"string alias before subcommand", []string{"issues", "-l", "bug", "create"}, []string{"issues", "create"}, []string{}, "labels", []string{"bug"}},
		{"string alias before command", []string{"-l", "bug", "issues", "create"}, []string{"issues", "create"}, []string{}, "labels", []string{"bug"}},
		{"label equals command name", []string{"-l", "issues", "issues", "create"}, []string{"issues", "create"}, []string{}, "labels", []string{"issues"}},
		{"group boolean before subcommand", []string{"issues", "--verbose", "create"}, []string{"issues", "create"}, []string{}, "verbose", true},
		{"global string before command", []string{"--config", "settings.yaml", "clone", "topic"}, []string{"clone"}, []string{"topic"}, "config", "settings.yaml"},
		{"combined aliases", []string{"issues", "-yl"}, []string{"issues"}, []string{}, "list", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, err := cli.NewInputWithRouter(tc.args, router, globals...)
			require.NoError(t, err)
			blueprint, err := cli.NewBlueprint(router, input, globals...)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCmd, blueprint.Cmd)
			assert.Equal(t, tc.wantArgs, blueprint.Args)
			for _, flag := range blueprint.Flags {
				if flag.Key == tc.key {
					assert.Equal(t, tc.value, flag.Value)

					return
				}
			}
			t.Fatalf("missing flag %s", tc.key)
		})
	}
}

func TestRouterBooleanDefaultAndFallback(t *testing.T) {
	router := cli.NewRouter(nil)
	router.DefaultCmd(cli.Cmd{Name: "default", Flags: []cli.Flag{{Long: "enabled", Kind: cli.FlagKindBool}}})
	router.Cmd(cli.Cmd{Name: "run", Flags: []cli.Flag{{Long: "enabled", Kind: cli.FlagKindBool}}})

	input, err := cli.NewInputWithRouter([]string{"--enabled"}, router)
	require.NoError(t, err)
	blueprint, err := cli.NewBlueprint(router, input)
	require.NoError(t, err)
	require.Len(t, blueprint.Flags, 1)
	assert.Equal(t, true, blueprint.Flags[0].Value)

	input, err = cli.NewInputWithRouter([]string{"--enabled", "run"}, router)
	require.NoError(t, err)
	blueprint, err = cli.NewBlueprint(router, input)
	require.NoError(t, err)
	assert.Equal(t, []string{"run"}, blueprint.Cmd)

	input, err = cli.NewInputWithRouter([]string{"unknown"}, router)
	require.NoError(t, err)
	_, err = cli.NewBlueprint(router, input)
	require.ErrorContains(t, err, "unknown command")

	_, err = cli.NewInputWithRouter([]string{"--bad.flag"}, router)
	require.ErrorContains(t, err, "invalid cli flag")
}

func TestExplicitCommandBeforeDefaultScope(t *testing.T) {
	router := cli.NewRouter(nil)
	router.DefaultCmd(cli.Cmd{Name: "default", Flags: []cli.Flag{{Long: "label", Short: "l"}}})
	router.Cmd(cli.Cmd{Name: "run", Flags: []cli.Flag{{Long: "list", Short: "l", Kind: cli.FlagKindBool}}})

	input, err := cli.NewInputWithRouter([]string{"-l", "run"}, router)
	require.NoError(t, err)
	blueprint, err := cli.NewBlueprint(router, input)
	require.NoError(t, err)
	assert.Equal(t, []string{"run"}, blueprint.Cmd)
	require.Len(t, blueprint.Flags, 1)
	assert.Equal(t, true, blueprint.Flags[0].Value)
}
