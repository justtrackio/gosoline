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
	configOptions := []cfg.Option{
		cfg.WithErrorHandlers(defaultErrorHandler),
		cfg.WithConfigFile("./config.dist.yml", "yml"),
		cfg.WithConfigFileFlag("config"),
	}

	config := cfg.New()
	if err := config.Option(configOptions...); err != nil {
		defaultErrorHandler("can not initialize the config: %w", err)
	}

	logger, err := newCliLogger()
	if err != nil {
		defaultErrorHandler("can not initialize the logger: %w", err)
	}

	settings := &kernelSettings{}
	config.UnmarshalKey("kernel", settings)

	ctx := appctx.WithContainer(context.Background())

	k, err := kernel.New(ctx, config, logger, kernel.KillTimeout(settings.KillTimeout))
	if err != nil {
		defaultErrorHandler("can not initialize the kernel: %w", err)
	}

	k.Add("cli", module, kernel.ModuleType(kernel.TypeEssential), kernel.ModuleStage(kernel.StageApplication))
	for _, otherModuleMap := range otherModuleMaps {
		for name, otherModule := range otherModuleMap {
			k.Add(name, otherModule)
		}
	}
	k.Run()
}
