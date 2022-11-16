package cli

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
)

type kernelSettings struct {
	KillTimeout time.Duration `cfg:"killTimeout" default:"10s"`
}

func Run(module kernel.ModuleFactory, otherModuleMaps ...map[string]kernel.ModuleFactory) {
	var err error
	var cfgPostProcessors map[string]int

	configOptions := []cfg.Option{
		cfg.WithErrorHandlers(defaultErrorHandler),
		cfg.WithConfigFile("./config.dist.yml", "yml"),
		cfg.WithConfigFileFlag("config"),
	}

	config := cfg.New()
	if err := config.Option(configOptions...); err != nil {
		defaultErrorHandler("can not initialize the config: %w", err)
		return
	}

	if cfgPostProcessors, err = cfg.ApplyPostProcessors(config); err != nil {
		defaultErrorHandler("can not apply post processor on config: %w", err)
		return
	}

	logger, err := newCliLogger()
	if err != nil {
		defaultErrorHandler("can not initialize the logger: %w", err)
		return
	}

	for name, priority := range cfgPostProcessors {
		logger.Info("applied priority %d config post processor '%s'", priority, name)
	}

	settings := &kernelSettings{}
	config.UnmarshalKey("kernel", settings)

	ctx := appctx.WithContainer(context.Background())

	options := []kernel.Option{
		kernel.WithKillTimeout(settings.KillTimeout),
		kernel.WithModuleFactory("cli", module, kernel.ModuleType(kernel.TypeEssential), kernel.ModuleStage(kernel.StageApplication)),
	}

	for _, otherModuleMap := range otherModuleMaps {
		for name, otherModule := range otherModuleMap {
			options = append(options, kernel.WithModuleFactory(name, otherModule))
		}
	}

	k, err := kernel.BuildKernel(ctx, config, logger, options)
	if err != nil {
		defaultErrorHandler("can not initialize the kernel: %w", err)
		return
	}

	k.Run()
}
