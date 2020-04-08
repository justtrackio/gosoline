package application

import (
	"flag"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/mon/daemon"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

type Option func(app *App)
type ConfigOption func(config cfg.GosoConf) error
type LoggerOption func(config cfg.GosoConf, logger mon.GosoLog) error
type KernelOption func(config cfg.GosoConf, kernel kernel.GosoKernel) error
type SetupOption func(config cfg.GosoConf, logger mon.GosoLog) error

type kernelSettings struct {
	KillTimeout time.Duration `cfg:"killTimeout" default:"10s"`
}

type loggerSettings struct {
	Level           string                 `cfg:"level" default:"info" validate:"required"`
	Format          string                 `cfg:"format" default:"console" validate:"required"`
	TimestampFormat string                 `cfg:"timestamp_format" default:"15:04:05.000" validate:"required"`
	Tags            map[string]interface{} `cfg:"tags"`
}

func WithApiHealthCheck(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, kernel kernel.GosoKernel) error {
		kernel.Add("api-health-check", apiserver.NewApiHealthCheck())
		return nil
	})
}

func WithConfigEnvKeyReplacer(replacer *strings.Replacer) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			if err := config.Option(cfg.WithEnvKeyReplacer(replacer)); err != nil {
				return err
			}

			return nil
		})
	}
}

func WithConfigEnvKeyPrefix(prefix string) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			prefix = strings.Replace(prefix, "-", "_", -1)

			return config.Option(cfg.WithEnvKeyPrefix(prefix))
		})
	}
}

func WithConfigErrorHandlers(handlers ...cfg.ErrorHandler) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithErrorHandlers(handlers...))
		})
	}
}

func WithConfigFile(filePath string, fileType string) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigFile(filePath, fileType))
		})
	}
}

func WithConfigFileFlag(app *App) {
	app.addConfigOption(func(config cfg.GosoConf) error {
		flags := flag.NewFlagSet("cfg", flag.ContinueOnError)

		configFile := flags.String("config", "", "path to a config file")
		err := flags.Parse(os.Args[1:])

		if err != nil {
			return err
		}

		return config.Option(cfg.WithConfigFile(*configFile, "yml"))
	})
}

func WithConfigMap(configMap map[string]interface{}) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigMap(configMap))
		})
	}
}

func WithConfigSanitizers(sanitizers ...cfg.Sanitizer) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithSanitizers(sanitizers...))
		})
	}
}

func WithFixtures(fixtureSets []*fixtures.FixtureSet) Option {
	return func(app *App) {
		app.addSetupOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			loader := fixtures.NewFixtureLoader(config, logger)
			return loader.Load(fixtureSets)
		})
	}
}

func WithKernelSettingsFromConfig(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, k kernel.GosoKernel) error {
		settings := &kernelSettings{}
		config.UnmarshalKey("kernel", settings)

		return k.Option(kernel.KillTimeout(settings.KillTimeout))
	})
}

func WithLoggerApplicationTag(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		if !config.IsSet("app_name") {
			return errors.New("can not get application name from config to set it on logger")
		}

		return logger.Option(mon.WithTags(map[string]interface{}{
			"application": config.GetString("app_name"),
		}))
	})
}

func WithLoggerContextFieldsMessageEncoder() Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			stream.AddDefaultEncodeHandler(mon.NewMessageWithLoggingFieldsEncoder(config, logger))
			return nil
		})
	}
}

func WithLoggerContextFieldsResolver(resolver ...mon.ContextFieldsResolver) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			return logger.Option(mon.WithContextFieldsResolver(resolver...))
		})
	}
}

func WithLoggerFormat(format string) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			return logger.Option(mon.WithFormat(format))
		})
	}
}

func WithLoggerLevel(level string) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			return logger.Option(mon.WithLevel(level))
		})
	}
}

func WithLoggerHook(hook mon.LoggerHook) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			return logger.Option(mon.WithHook(hook))
		})
	}
}

func WithLoggerMetricHook(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		metricHook := daemon.NewMetricHook()
		return logger.Option(mon.WithHook(metricHook))
	})
}

func WithLoggerSentryHook(extraProvider ...mon.SentryExtraProvider) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			var err error

			env := config.GetString("env")
			sentryHook := mon.NewSentryHook(env)

			for _, p := range extraProvider {
				sentryHook, err = p(config, sentryHook)

				if err != nil {
					return errors.Wrap(err, "can not configure LoggerSentryHook")
				}
			}

			return logger.Option(mon.WithHook(sentryHook))
		})
	}
}

func WithLoggerSettingsFromConfig(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		settings := &loggerSettings{}
		config.UnmarshalKey("mon.logger", settings)

		loggerOptions := []mon.LoggerOption{
			mon.WithLevel(settings.Level),
			mon.WithFormat(settings.Format),
			mon.WithTimestampFormat(settings.TimestampFormat),
		}

		return logger.Option(loggerOptions...)
	})
}

func WithLoggerTagsFromConfig(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		settings := &loggerSettings{}
		config.UnmarshalKey("mon.logger", settings)

		return logger.Option(mon.WithTags(settings.Tags))
	})
}

func WithMetricDaemon(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, kernel kernel.GosoKernel) error {
		kernel.Add("metric", daemon.ProvideCwDaemon())
		return nil
	})
}

func WithTracing(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		tracingHook := tracing.NewLoggerErrorHook()

		options := []mon.LoggerOption{
			mon.WithHook(tracingHook),
			mon.WithContextFieldsResolver(tracing.ContextTraceFieldsResolver),
		}

		return logger.Option(options...)
	})

	app.addSetupOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		strategy := tracing.NewTraceIdErrorWarningStrategy(logger)
		stream.AddDefaultEncodeHandler(tracing.NewMessageWithTraceEncoder(strategy))

		return nil
	})
}
