package cli_test

import (
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cli"
	"github.com/stretchr/testify/suite"
)

func TestBlueprintTestSuite(t *testing.T) {
	suite.Run(t, new(BlueprintTestSuite))
}

type BlueprintTestSuite struct {
	suite.Suite
}

func (s *BlueprintTestSuite) newInput(args []string, flags []cli.InputFlag) *cli.Input {
	return &cli.Input{
		Arguments: args,
		Flags:     flags,
	}
}

func (s *BlueprintTestSuite) newBlueprint(router *cli.Router, input *cli.Input) *cli.Blueprint {
	bp, err := cli.NewBlueprint(router, input)
	s.Require().NoError(err)

	return bp
}

func (s *BlueprintTestSuite) assertBlueprintError(router *cli.Router, input *cli.Input, expected string) {
	bp, err := cli.NewBlueprint(router, input)

	s.Nil(bp)
	s.EqualError(err, expected)
}

func (s *BlueprintTestSuite) newBlueprintWithGlobalFlags(router *cli.Router, input *cli.Input, globalFlags []cli.Flag) *cli.Blueprint {
	bp, err := cli.NewBlueprint(router, input, globalFlags...)
	s.Require().NoError(err)

	return bp
}

func (s *BlueprintTestSuite) TestSimpleCommand() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "serve"})

	bp := s.newBlueprint(router, s.newInput([]string{"serve"}, nil))

	s.Equal([]string{"serve"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestCommandWithArgs() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "run"})

	bp := s.newBlueprint(router, s.newInput([]string{"run", "file1", "file2"}, nil))

	s.Equal([]string{"run"}, bp.Cmd)
	s.Equal([]string{"file1", "file2"}, bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestGroupThenCommand() {
	router := cli.NewRouter(nil)
	apiGroup := router.Group(cli.Group{Name: "api"})
	apiGroup.Cmd(cli.Cmd{Name: "serve"})

	bp := s.newBlueprint(router, s.newInput([]string{"api", "serve"}, nil))

	s.Equal([]string{"api", "serve"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestNestedGroups() {
	router := cli.NewRouter(nil)
	apiGroup := router.Group(cli.Group{Name: "api"})
	v2Group := apiGroup.Group(cli.Group{Name: "v2"})
	v2Group.Cmd(cli.Cmd{Name: "serve"})

	bp := s.newBlueprint(router, s.newInput([]string{"api", "v2", "serve"}, nil))

	s.Equal([]string{"api", "v2", "serve"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestGroupCommandWithArgs() {
	router := cli.NewRouter(nil)
	apiGroup := router.Group(cli.Group{Name: "api"})
	apiGroup.Cmd(cli.Cmd{Name: "get"})

	bp := s.newBlueprint(router, s.newInput([]string{"api", "get", "users", "123"}, nil))

	s.Equal([]string{"api", "get"}, bp.Cmd)
	s.Equal([]string{"users", "123"}, bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestGroupAndCommandAppOptions() {
	groupOption := application.WithConfigSetting("group", "api")
	cmdOption := application.WithConfigSetting("command", "serve")
	router := cli.NewRouter(nil)
	apiGroup := router.Group(cli.Group{Name: "api", AppOptions: []application.Option{groupOption}})
	apiGroup.Cmd(cli.Cmd{Name: "serve", AppOptions: []application.Option{cmdOption}})

	bp := s.newBlueprint(router, s.newInput([]string{"api", "serve"}, nil))

	s.Require().Len(bp.AppOptions, 2)
	s.Equal(reflect.ValueOf(groupOption).Pointer(), reflect.ValueOf(bp.AppOptions[0]).Pointer())
	s.Equal(reflect.ValueOf(cmdOption).Pointer(), reflect.ValueOf(bp.AppOptions[1]).Pointer())
}

func (s *BlueprintTestSuite) TestDefaultCommandAppOptions() {
	cmdOption := application.WithConfigSetting("command", "default")
	router := cli.NewRouter(nil)
	router.DefaultCmd(cli.Cmd{Name: "default", AppOptions: []application.Option{cmdOption}})

	bp := s.newBlueprint(router, s.newInput(nil, nil))

	s.Require().Len(bp.AppOptions, 1)
	s.Equal(reflect.ValueOf(cmdOption).Pointer(), reflect.ValueOf(bp.AppOptions[0]).Pointer())
}

func (s *BlueprintTestSuite) TestNoMatchingCommand() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "serve"})

	s.assertBlueprintError(router, s.newInput([]string{"unknown"}, nil), `unknown command "unknown"`)
}

func (s *BlueprintTestSuite) TestNoMatchingCommandWithPunctuation() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "serve"})

	s.assertBlueprintError(router, s.newInput([]string{"unknown!"}, nil), `unknown command "unknown!"`)
}

func (s *BlueprintTestSuite) TestInvalidConfiguredNames() {
	testCases := map[string]struct {
		setup       func(router *cli.Router)
		args        []string
		expectedErr string
	}{
		"command": {
			setup: func(router *cli.Router) {
				router.Cmd(cli.Cmd{Name: "bad.command"})
			},
			args:        []string{"bad.command"},
			expectedErr: `invalid cli command "bad.command": must contain only alphanumeric characters, hyphens, or underscores`,
		},
		"group": {
			setup: func(router *cli.Router) {
				router.Group(cli.Group{Name: "bad/group"})
			},
			args:        []string{"bad/group"},
			expectedErr: `invalid cli group "bad/group": must contain only alphanumeric characters, hyphens, or underscores`,
		},
		"nested command": {
			setup: func(router *cli.Router) {
				group := router.Group(cli.Group{Name: "api"})
				group.Cmd(cli.Cmd{Name: "bad.command"})
			},
			args:        []string{"api", "bad.command"},
			expectedErr: `invalid cli command "bad.command": must contain only alphanumeric characters, hyphens, or underscores`,
		},
		"default command": {
			setup: func(router *cli.Router) {
				router.DefaultCmd(cli.Cmd{Name: "bad!"})
			},
			expectedErr: `invalid cli command "bad!": must contain only alphanumeric characters, hyphens, or underscores`,
		},
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			router := cli.NewRouter(nil)
			tc.setup(router)

			s.assertBlueprintError(router, s.newInput(tc.args, nil), tc.expectedErr)
		})
	}
}

func (s *BlueprintTestSuite) TestEmptyInput() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "serve"})

	bp := s.newBlueprint(router, s.newInput(nil, nil))

	s.Empty(bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestFlagsFromShortKey() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env", Default: "dev"},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, []cli.InputFlag{{Name: "e", Value: "prod"}}))

	s.Equal([]string{"run"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "env", Value: "prod"}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestGlobalFlags() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "run"})

	bp := s.newBlueprintWithGlobalFlags(router, s.newInput([]string{"run"}, []cli.InputFlag{{Name: "global", Value: "value"}}), []cli.Flag{
		{Short: "g", Long: "global"},
	})

	s.Equal([]string{"run"}, bp.Cmd)
	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "global", Value: "value"}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestCommandFlagHasHighestPriority() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Long: "env", Default: "command"},
		},
	})

	bp := s.newBlueprintWithGlobalFlags(router, s.newInput([]string{"run"}, nil), []cli.Flag{
		{Long: "env", Default: "global"},
	})

	s.Len(bp.Flags, 2)
	s.Equal(cli.BlueprintFlag{Key: "env", Value: "global"}, bp.Flags[0])
	s.Equal(cli.BlueprintFlag{Key: "env", Value: "command"}, bp.Flags[1])
}

func (s *BlueprintTestSuite) TestFlagsFromLongKey() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env"},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, []cli.InputFlag{{Name: "env", Value: "staging"}}))

	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "env", Value: "staging"}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestFlagsWithDefault() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env", Default: "dev"},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, nil))

	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "env", Value: "dev"}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestFlagsWithCustomKey() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env", CfgKey: "custom.env.key"},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, []cli.InputFlag{{Name: "env", Value: "prod"}}))

	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "env", CustomKey: "custom.env.key", Value: "prod"}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestFlagEmptyValueOmitted() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env"},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, nil))

	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestStringFlagLastValueWins() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env", Kind: cli.FlagKindString},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, []cli.InputFlag{
		{Name: "env", Value: "from-long"},
		{Name: "e", Value: "from-short"},
	}))

	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "env", Value: "from-short"}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestUnsupportedFlagKind() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Long: "env", Kind: cli.FlagKind("unknown")},
		},
	})

	s.assertBlueprintError(router, s.newInput([]string{"run"}, nil), `unsupported cli flag kind "unknown" for flag "env"`)
}

func (s *BlueprintTestSuite) TestMultipleFlags() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env"},
			{Short: "v", Long: "verbose"},
			{Short: "o", Long: "output"},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, []cli.InputFlag{
		{Name: "env", Value: "prod"},
		{Name: "verbose", Value: "true"},
	}))

	s.Len(bp.Flags, 2)

	flagsByKey := make(map[string]cli.BlueprintFlag)
	for _, f := range bp.Flags {
		flagsByKey[f.Key] = f
	}

	s.Contains(flagsByKey, "env")
	s.Equal("prod", flagsByKey["env"].Value)
	s.Contains(flagsByKey, "verbose")
	s.Equal("true", flagsByKey["verbose"].Value)
	s.NotContains(flagsByKey, "output")
}

func (s *BlueprintTestSuite) TestListFlagCombinesShortAndLongValuesInOrder() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "i", Long: "include", Kind: cli.FlagKindList},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, []cli.InputFlag{
		{Name: "include", Value: "first"},
		{Name: "i", Value: "second"},
		{Name: "include", Value: "third"},
	}))

	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "include", Value: []string{"first", "second", "third"}}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestListFlagWithDefault() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "i", Long: "include", Kind: cli.FlagKindList, Default: "default"},
		},
	})

	bp := s.newBlueprint(router, s.newInput([]string{"run"}, nil))

	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "include", Value: []string{"default"}}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestOnlyGroupNoCommand() {
	router := cli.NewRouter(nil)
	router.Group(cli.Group{Name: "api"})

	s.assertBlueprintError(router, s.newInput([]string{"api", "unknown"}, nil), `unknown command "api unknown"`)
}
