package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	kernelPkg "github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	// Option configures a Cli before it parses input and runs the selected command.
	Option func(cli *Cli)
)

// WithAppOptions adds gosoline application options that apply to every command.
func WithAppOptions(option ...application.Option) Option {
	return func(cli *Cli) {
		cli.appOptions = append(cli.appOptions, option...)
	}
}

// WithFlag registers a global CLI flag available to every command.
func WithFlag(flag Flag) Option {
	return func(cli *Cli) {
		cli.flags = append(cli.flags, flag)
	}
}

// WithHelpLineLength configures the maximum rendered help line length. Use a negative value to disable wrapping.
func WithHelpLineLength(length int) Option {
	return func(cli *Cli) {
		cli.helpLineLength = length
	}
}

// WithHelp configures top-level help text and registers the built-in help command.
func WithHelp(name string, description string) Option {
	return func(cli *Cli) {
		cli.name = name
		cli.description = description

		cli.Cmd(Cmd{
			Name:        "help",
			Description: "help about any command. call help <command> [<subcommand>] for further usage.",
			AppOptions: []application.Option{
				application.WithModuleFactory("main", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernelPkg.Module, error) {
					return kernelPkg.NewModuleFunc(func(ctx context.Context) error {
						return cli.writeHelp(os.Stdout, trimHelpCommand(cli.input.Arguments)...)
					}), nil
				}),
			},
		})
	}
}

// WithRunFunc adapts a module run function factory into a gosoline module factory.
func WithRunFunc(f func(ctx context.Context, config cfg.Config, logger log.Logger) (kernelPkg.ModuleRunFunc, error)) kernelPkg.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernelPkg.Module, error) {
		var err error
		var run kernelPkg.ModuleRunFunc

		if run, err = f(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not build run function: %w", err)
		}

		return kernelPkg.NewModuleFunc(run), nil
	}
}

// WithVersion registers a built-in version command that prints the provided version string.
func WithVersion(version string) Option {
	return func(cli *Cli) {
		cli.Cmd(Cmd{
			Name:        "version",
			Description: "Show version information.",
			AppOptions: []application.Option{
				application.WithModuleFactory("main", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernelPkg.Module, error) {
					return kernelPkg.NewModuleFunc(func(ctx context.Context) error {
						fmt.Println(version)

						return nil
					}), nil
				}),
			},
		})
	}
}
