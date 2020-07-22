package application

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"strings"
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
		WithConfigErrorHandlers(defaultErrorHandler),
		WithConfigFile("./config.dist.yml", "yml"),
		WithConfigFileFlag,
		WithConfigEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
		WithConfigSanitizers(cfg.TimeSanitizer),
		WithConfigServer,
		WithLoggerFormat(mon.FormatGelfFields),
		WithLoggerApplicationTag,
		WithLoggerTagsFromConfig,
		WithLoggerSettingsFromConfig,
		WithLoggerContextFieldsMessageEncoder(),
		WithLoggerContextFieldsResolver(mon.ContextLoggerFieldsResolver),
		WithLoggerMetricHook,
		WithLoggerSentryHook(mon.SentryExtraConfigProvider, mon.SentryExtraEcsMetadataProvider),
		WithKernelSettingsFromConfig,
		WithApiHealthCheck,
		WithMetricDaemon,
		WithTracing,
	}

	options = append(defaults, options...)

	return New(options...)
}

func New(options ...Option) kernel.Kernel {
	config := cfg.New()
	logger := mon.NewLogger()
	ker := kernel.New(config, logger)

	return NewWithInterfaces(config, logger, ker, options...)
}

func NewWithInterfaces(config cfg.GosoConf, logger mon.GosoLog, ker kernel.GosoKernel, options ...Option) kernel.Kernel {
	app := &App{
		configOptions: make([]ConfigOption, 0),
		loggerOptions: make([]LoggerOption, 0),
		kernelOptions: make([]KernelOption, 0),
	}

	for _, opt := range options {
		opt(app)
	}

	for _, opt := range app.configOptions {
		if err := opt(config); err != nil {
			defaultErrorHandler(err, "can not apply config options on application")
		}
	}

	for _, opt := range app.loggerOptions {
		if err := opt(config, logger); err != nil {
			defaultErrorHandler(err, "can not apply logger options on application")
		}
	}

	if err := cfg.ApplyPostProcessors(config, logger); err != nil {
		defaultErrorHandler(err, "can not apply post processor on config")
	}

	for _, opt := range app.setupOptions {
		if err := opt(config, logger); err != nil {
			defaultErrorHandler(err, "can not apply setup options on application")
		}
	}

	for _, opt := range app.kernelOptions {
		if err := opt(config, ker); err != nil {
			defaultErrorHandler(err, "can not apply kernel options on application")
		}
	}

	return ker
}
