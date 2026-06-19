package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	kernelPkg "github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Cli struct {
	*Router

	input      *Input
	flags      []Flag
	cliOptions []Option
	appOptions []application.Option
}

func NewCli(options ...Option) *Cli {
	input := NewInput()
	router := NewRouter(nil)

	defaultAppOptions := []application.Option{
		application.WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		application.WithConfigSanitizers(cfg.TimeSanitizer),
	}

	return &Cli{
		Router:     router,
		input:      input,
		flags:      nil,
		cliOptions: options,
		appOptions: defaultAppOptions,
	}
}

func (c *Cli) Run() {
	for _, opt := range c.cliOptions {
		opt(c)
	}

	blueprint := NewBlueprint(c.Router, c.input)
	appOptions := c.processArgs(blueprint)

	appOptions = append(c.appOptions, appOptions...)
	appOptions = append(appOptions, application.WithConfigSetting("cli.cmd", blueprint.Cmd))
	appOptions = append(appOptions, application.WithConfigSetting("cli.args", blueprint.Args))

	for _, flag := range blueprint.Flags {
		appOptions = append(appOptions, flag.AppOptions...)

		key := fmt.Sprintf("cli.flags.%s", flag.Key)
		appOptions = append(appOptions, application.WithConfigSetting(key, flag.Value))

		if flag.CustomKey != "" {
			appOptions = append(appOptions, application.WithConfigSetting(flag.CustomKey, flag.Value))
		}
	}

	c.runApp(appOptions)
}

func (c *Cli) processArgs(blueprint *Blueprint) []application.Option {
	var selected bool
	var selectedCmd Cmd

	router := c.Router
	defaultCmd := router.defaultCmd
	appOptions := make([]application.Option, 0)

	for _, arg := range blueprint.Cmd {
		if group, ok := router.groups[arg]; ok {
			defaultCmd = group.child.defaultCmd
			appOptions = append(appOptions, group.AppOptions...)
			router = group.child

			continue
		}

		if cmd, ok := router.cmds[arg]; ok {
			selected = true
			selectedCmd = cmd

			break
		}
	}

	if !selected {
		selectedCmd = defaultCmd
	}

	appOptions = append(appOptions, selectedCmd.AppOptions...)

	return appOptions
}

func (c *Cli) runApp(appOptions []application.Option) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "dev",
			"name": "gosoline",
		},
	})

	logger := log.NewLoggerWithInterfaces(clock.NewRealClock(), []log.Handler{LogHandler{}})

	var err error
	var ker kernelPkg.Kernel
	if ker, err = application.NewWithInterfaces(ctx, config, logger, appOptions...); err != nil {
		fmt.Printf("can not build application: %v\n", err)
		os.Exit(1)
	}

	ker.Run()
}
