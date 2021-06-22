package application

import (
	"fmt"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
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
		WithApiHealthCheck,
		WithConfigErrorHandlers(defaultErrorHandler),
		WithConfigFile("./config.dist.yml", "yml"),
		WithConfigFileFlag,
		WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		WithConfigSanitizers(cfg.TimeSanitizer),
		WithConfigServer,
		WithConsumerMessagesPerRunnerMetrics,
		WithKernelSettingsFromConfig,
		WithLoggerApplicationTag,
		WithLoggerContextFieldsMessageEncoder,
		WithLoggerContextFieldsResolver(log.ContextLoggerFieldsResolver),
		WithLoggerHandlersFromConfig,
		WithLoggerMetricHandler,
		WithLoggerSentryHandler(log.SentryContextConfigProvider, log.SentryContextEcsMetadataProvider),
		WithMetricDaemon,
		WithProducerDaemon,
		WithTracing,
		WithUTCClock(true),
	}

	options = append(defaults, options...)

	return New(options...)
}

func New(options ...Option) kernel.Kernel {
	var err error
	config := cfg.New()
	logger := log.NewLogger()
	var ker kernel.Kernel

	if ker, err = NewWithInterfaces(config, logger, options...); err != nil {
		defaultErrorHandler("can initialize the app: %w", err)
	}

	return ker
}

func NewWithInterfaces(config cfg.GosoConf, logger log.GosoLogger, options ...Option) (kernel.Kernel, error) {
	var err error
	var ker kernel.GosoKernel
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

	for name, priority := range cfgPostProcessors {
		logger.Info("applied priority %d config post processor '%s'", priority, name)
	}

	for _, opt := range app.setupOptions {
		if err = opt(config, logger); err != nil {
			return nil, fmt.Errorf("can not apply setup options on application: %w", err)
		}
	}

	if ker, err = kernel.New(config, logger); err != nil {
		return nil, fmt.Errorf("can not create kernel: %w", err)
	}

	for _, opt := range app.kernelOptions {
		if err := opt(config, ker); err != nil {
			return nil, fmt.Errorf("can not apply kernel options on application: %w", err)
		}
	}

	return ker, nil
}
