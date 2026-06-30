package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/funk"
	kernelPkg "github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type cmdNotFoundError struct {
	cmd      []string
	helpPath []string
}

func (e *cmdNotFoundError) Error() string {
	return fmt.Sprintf("unknown command %q", strings.Join(e.cmd, " "))
}

type Cli struct {
	*Router

	name        string
	description string

	input      *Input
	flags      []Flag
	cliOptions []Option
	appOptions []application.Option
}

func NewCli(options ...Option) *Cli {
	router := NewRouter(nil)

	defaultAppOptions := []application.Option{
		application.WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		application.WithConfigSanitizers(cfg.TimeSanitizer),
	}

	return &Cli{
		Router:     router,
		flags:      nil,
		cliOptions: options,
		appOptions: defaultAppOptions,
	}
}

func (c *Cli) Run() {
	var err error

	if c.input, err = NewInput(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, opt := range c.cliOptions {
		opt(c)
	}

	if hasHelpFlag(c.input) {
		if err := c.writeHelp(os.Stdout, c.input.Arguments...); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		return
	}

	blueprint, err := NewBlueprint(c.Router, c.input, c.flags...)
	if err != nil {
		if cmdErr, ok := err.(*cmdNotFoundError); ok {
			if err := c.writeCmdNotFound(os.Stderr, os.Stdout, cmdErr); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			os.Exit(1)
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	appOptions := funk.Concat(c.appOptions, blueprint.AppOptions)
	appOptions = append(appOptions, application.WithConfigSetting("cli.cmd", blueprint.Cmd))
	appOptions = append(appOptions, application.WithConfigSetting("cli.args", blueprint.Args))

	for _, flag := range blueprint.Flags {
		appOptions = append(appOptions, flag.AppOptions...)

		key := fmt.Sprintf("cli.flags.%s", strings.ReplaceAll(flag.Key, "-", "_"))
		appOptions = append(appOptions, application.WithConfigSetting(key, flag.Value))

		if flag.CustomKey != "" {
			appOptions = append(appOptions, application.WithConfigSetting(flag.CustomKey, flag.Value))
		}
	}

	c.runApp(appOptions)
}

func (c *Cli) runApp(appOptions []application.Option) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "dev",
			"name": "gosoline",
		},
	})

	logger := log.NewLoggerWithInterfaces(clock.NewRealClock(), []log.Handler{})

	var err error
	var ker kernelPkg.Kernel
	if ker, err = application.NewWithInterfaces(ctx, config, logger, appOptions...); err != nil {
		fmt.Printf("can not build application: %v\n", err)
		os.Exit(1)
	}

	ker.Run()
}
