package application

import (
	"context"
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type App struct {
	configOptions        []ConfigOption
	configPostProcessors []cfg.PostProcessor
	kernelOptions        []KernelOption
	loggerOptions        []LoggerOption
	setupOptions         []SetupOption
}

func (a *App) addConfigOption(opt ConfigOption) {
	a.configOptions = append(a.configOptions, opt)
}

func (a *App) addKernelOption(opt KernelOption) {
	a.kernelOptions = append(a.kernelOptions, opt)
}

func (a *App) addLoggerOption(opt LoggerOption) {
	a.loggerOptions = append(a.loggerOptions, opt)
}

func (a *App) addSetupOption(opt SetupOption) {
	a.setupOptions = append(a.setupOptions, opt)
}

func Default(options ...Option) kernel.Kernel {
	defaults := []Option{
		WithHttpHealthCheck,
		// use an indirection for the default error handler. Otherwise, changes
		// to the default are not reflected in the config error handler
		WithConfigErrorHandlers(callDefaultErrorHandler),
		WithConfigFile("./config.dist.yml", "yml"),
		WithConfigFileFlag,
		WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		WithConfigSanitizers(cfg.TimeSanitizer),
		WithMetadataServer,
		WithConsumerMessagesPerRunnerMetrics,
		WithLoggerGroupTag,
		WithLoggerApplicationTag,
		WithLoggerContextFieldsMessageEncoder,
		WithLoggerContextFieldsResolver(log.ContextFieldsResolver),
		WithLoggerHandlersFromConfig,
		WithLoggerMetricHandler,
		WithLoggerSentryHandler(log.SentryContextConfigProvider, log.SentryContextEcsMetadataProvider),
		WithMetrics,
		WithProducerDaemon,
		WithTaskRunner,
		WithProfiling,
		WithTracing,
		WithUTCClock(true),
	}

	options = append(defaults, options...)

	return New(options...)
}

func New(options ...Option) kernel.Kernel {
	var err error
	var ker kernel.Kernel

	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	logger := log.NewLogger()

	if ker, err = NewWithInterfaces(ctx, config, logger, options...); err != nil {
		defaultErrorHandler("can not initialize the app: %w", err)
	}

	return ker
}

func NewWithInterfaces(ctx context.Context, config cfg.GosoConf, logger log.GosoLogger, options ...Option) (kernel.Kernel, error) {
	var err error
	var cfgPostProcessors map[string]int

	app := &App{
		configOptions: make([]ConfigOption, 0),
		loggerOptions: make([]LoggerOption, 0),
		kernelOptions: make([]KernelOption, 0),
	}

	for _, opt := range options {
		opt(app)
	}

	for _, opt := range app.configOptions {
		if err = opt(config); err != nil {
			return nil, fmt.Errorf("can not apply config options on application: %w", err)
		}
	}

	if cfgPostProcessors, err = cfg.ApplyPostProcessors(config); err != nil {
		return nil, fmt.Errorf("can not apply post processor on config: %w", err)
	}

	for _, opt := range app.loggerOptions {
		if err = opt(config, logger); err != nil {
			return nil, fmt.Errorf("can not apply logger options on application: %w", err)
		}
	}

	// switch the default error handler to use our logger - this ensures any application name or other setting we already
	// configured for our logger gets picked up by the default error handler. We can only configure this here after we
	// set up our logger successfully (otherwise the logger might not write any messages at all).
	withDefaultErrorHandler(func(msg string, args ...interface{}) {
		logger.Error(msg, args...)
		os.Exit(1)
	})

	for name, priority := range cfgPostProcessors {
		logger.Info("applied priority %d config post processor '%s'", priority, name)
	}

	for _, opt := range app.setupOptions {
		if err = opt(config, logger); err != nil {
			return nil, fmt.Errorf("can not apply setup options on application: %w", err)
		}
	}

	kernelOptions := make([]kernel.Option, len(app.kernelOptions))

	for i := 0; i < len(app.kernelOptions); i++ {
		kernelOptions[i] = app.kernelOptions[i](config)
	}

	return kernel.BuildKernel(ctx, config, logger, kernelOptions)
}
