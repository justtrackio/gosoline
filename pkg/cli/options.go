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
	Option func(cli *Cli)
)

func WithAppOptions(option ...application.Option) Option {
	return func(cli *Cli) {
		cli.appOptions = append(cli.appOptions, option...)
	}
}

func WithFlag(flag Flag) Option {
	return func(cli *Cli) {
		cli.flags = append(cli.flags, flag)
	}
}

func WithHelp(name string, description string) Option {
	return func(cli *Cli) {
		cli.name = name
		cli.description = description

		cli.Cmd(Cmd{
			Name:        "help",
			Description: "Help about any command",
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
