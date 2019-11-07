package application

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"strings"
)

type App struct {
	configOptions []ConfigOption
	kernelOptions []KernelOption
	loggerOptions []LoggerOption
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

func ApiDefaultOptions(options ...Option) []Option {
	defaults := []Option{
		WithConfigErrorHandlers(defaultErrorHandler),
		WithConfigFile("./config.dist.yml", "yml"),
		WithConfigFileFlag,
		WithConfigEnvKeyReplacer(strings.NewReplacer(".", "_")),
		WithLoggerFormat(mon.FormatGelfFields),
		WithLoggerApplicationTag,
		WithLoggerTagsFromConfig,
		WithLoggerSettingsFromConfig,
		WithLoggerContextFieldsResolver(mon.ContextLoggerFieldsResolver, tracing.ContextTraceFieldsResolver),
		WithLoggerMetricHook,
		WithLoggerSentryHook(mon.SentryExtraConfigProvider, mon.SentryExtraEcsMetadataProvider),
		WithMetricDaemon,
	}

	return append(defaults, options...)
}

func DefaultOptions(options ...Option) []Option {
	return append(ApiDefaultOptions(options...), WithApiHealthCheck)
}

// Default options for the kernel if you define your own apiserver.
// It does the same as Default, but does not include an health check.
// Your apiserver will provide the health check instead.
func ApiDefault(options ...Option) kernel.Kernel {
	return New(ApiDefaultOptions(options...)...)
}

func Default(options ...Option) kernel.Kernel {
	return New(DefaultOptions(options...)...)
}

func New(options ...Option) kernel.Kernel {
	app := &App{
		configOptions: make([]ConfigOption, 0),
		loggerOptions: make([]LoggerOption, 0),
		kernelOptions: make([]KernelOption, 0),
	}

	for _, opt := range options {
		opt(app)
	}

	config := cfg.New()
	for _, opt := range app.configOptions {
		if err := opt(config); err != nil {
			defaultErrorHandler(err, "can not apply config options on application")
		}
	}

	logger := mon.NewLogger()
	for _, opt := range app.loggerOptions {
		if err := opt(config, logger); err != nil {
			defaultErrorHandler(err, "can not apply logger options on application")
		}
	}

	settings := &kernel.Settings{}
	config.UnmarshalKey("kernel", settings)

	k := kernel.New(config, logger, settings)
	for _, opt := range app.kernelOptions {
		if err := opt(config, k); err != nil {
			defaultErrorHandler(err, "can not apply kernel options on application")
		}
	}

	return k
}
