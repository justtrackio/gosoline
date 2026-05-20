package cli

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	kernelPkg "github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	Option func(cli *Cli)
)

func WithVersion(version string) Option {
	return func(cli *Cli) {
		cli.Cmd(Cmd{
			Name: "version",
			ModuleFactory: func(ctx context.Context, config cfg.Config, logger log.Logger) (kernelPkg.Module, error) {
				return kernelPkg.NewModuleFunc(func(ctx context.Context) error {
					fmt.Println(version)

					return nil
				}), nil
			},
		})
	}
}

func WithFlag(flag Flag) Option {
	return func(cli *Cli) {
		cli.flags = append(cli.flags, flag)
	}
}

func WithAppOptions(option ...application.Option) Option {
	return func(cli *Cli) {
		cli.appOptions = append(cli.appOptions, option...)
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
