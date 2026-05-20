package cli_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cli"
	"github.com/stretchr/testify/suite"
)

func TestBlueprintTestSuite(t *testing.T) {
	suite.Run(t, new(BlueprintTestSuite))
}

type BlueprintTestSuite struct {
	suite.Suite
}

func (s *BlueprintTestSuite) newInput(args []string, flags map[string]string) *cli.Input {
	if flags == nil {
		flags = make(map[string]string)
	}

	return &cli.Input{
		Arguments: args,
		Flags:     flags,
	}
}

func (s *BlueprintTestSuite) TestSimpleCommand() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "serve"})

	bp := cli.NewBlueprint(router, s.newInput([]string{"serve"}, nil))

	s.Equal([]string{"serve"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestCommandWithArgs() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "run"})

	bp := cli.NewBlueprint(router, s.newInput([]string{"run", "file1", "file2"}, nil))

	s.Equal([]string{"run"}, bp.Cmd)
	s.Equal([]string{"file1", "file2"}, bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestGroupThenCommand() {
	router := cli.NewRouter(nil)
	apiGroup := router.Group(cli.Group{Name: "api"})
	apiGroup.Cmd(cli.Cmd{Name: "serve"})

	bp := cli.NewBlueprint(router, s.newInput([]string{"api", "serve"}, nil))

	s.Equal([]string{"api", "serve"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestNestedGroups() {
	router := cli.NewRouter(nil)
	apiGroup := router.Group(cli.Group{Name: "api"})
	v2Group := apiGroup.Group(cli.Group{Name: "v2"})
	v2Group.Cmd(cli.Cmd{Name: "serve"})

	bp := cli.NewBlueprint(router, s.newInput([]string{"api", "v2", "serve"}, nil))

	s.Equal([]string{"api", "v2", "serve"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestGroupCommandWithArgs() {
	router := cli.NewRouter(nil)
	apiGroup := router.Group(cli.Group{Name: "api"})
	apiGroup.Cmd(cli.Cmd{Name: "get"})

	bp := cli.NewBlueprint(router, s.newInput([]string{"api", "get", "users", "123"}, nil))

	s.Equal([]string{"api", "get"}, bp.Cmd)
	s.Equal([]string{"users", "123"}, bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestNoMatchingCommand() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "serve"})

	bp := cli.NewBlueprint(router, s.newInput([]string{"unknown"}, nil))

	s.Empty(bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestEmptyInput() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{Name: "serve"})

	bp := cli.NewBlueprint(router, s.newInput(nil, nil))

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

	bp := cli.NewBlueprint(router, s.newInput([]string{"run"}, map[string]string{"e": "prod"}))

	s.Equal([]string{"run"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "env", Value: "prod"}, bp.Flags[0])
}

func (s *BlueprintTestSuite) TestFlagsFromLongKey() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env"},
		},
	})

	bp := cli.NewBlueprint(router, s.newInput([]string{"run"}, map[string]string{"env": "staging"}))

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

	bp := cli.NewBlueprint(router, s.newInput([]string{"run"}, nil))

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

	bp := cli.NewBlueprint(router, s.newInput([]string{"run"}, map[string]string{"env": "prod"}))

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

	bp := cli.NewBlueprint(router, s.newInput([]string{"run"}, nil))

	s.Empty(bp.Flags)
}

func (s *BlueprintTestSuite) TestShortFlagTakesPrecedence() {
	router := cli.NewRouter(nil)
	router.Cmd(cli.Cmd{
		Name: "run",
		Flags: []cli.Flag{
			{Short: "e", Long: "env"},
		},
	})

	bp := cli.NewBlueprint(router, s.newInput([]string{"run"}, map[string]string{
		"e":   "from-short",
		"env": "from-long",
	}))

	s.Len(bp.Flags, 1)
	s.Equal(cli.BlueprintFlag{Key: "env", Value: "from-short"}, bp.Flags[0])
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

	bp := cli.NewBlueprint(router, s.newInput([]string{"run"}, map[string]string{
		"env":     "prod",
		"verbose": "true",
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

func (s *BlueprintTestSuite) TestOnlyGroupNoCommand() {
	router := cli.NewRouter(nil)
	router.Group(cli.Group{Name: "api"})

	bp := cli.NewBlueprint(router, s.newInput([]string{"api", "unknown"}, nil))

	s.Equal([]string{"api"}, bp.Cmd)
	s.Empty(bp.Args)
	s.Empty(bp.Flags)
}
