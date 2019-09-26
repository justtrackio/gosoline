package application

import (
	"flag"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/pkg/errors"
	"os"
	"strings"
)

type Option func(app *App)
type ConfigOption func(config cfg.GosoConf) error
type LoggerOption func(config cfg.GosoConf, logger mon.GosoLog) error
type KernelOption func(config cfg.GosoConf, kernel kernel.Kernel) error

func WithApiHealthCheck(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, kernel kernel.Kernel) error {
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

func WithConfigFile(filePath string, fileType string) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigFile(filePath, fileType))
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

func WithLoggerMetricHook(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		metricHook := mon.NewMetricHook()
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
		if !config.IsSet("log_level") {
			return errors.New("need config key 'log_level' to set level on the logger")
		}

		if !config.IsSet("log_format") {
			return errors.New("need config key 'log_format' to set format on the logger")
		}

		level := config.GetString("log_level")
		format := config.GetString("log_format")

		return logger.Option(mon.WithLevel(level), mon.WithFormat(format))
	})
}

func WithLoggerTagsFromConfig(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		if !config.IsSet("log_tags") {
			return errors.New("need config key 'log_tags' to set tags from the config on the logger")
		}

		tags := config.GetStringMap("log_tags")

		return logger.Option(mon.WithTags(tags))
	})
}

func WithMetricDaemon(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, kernel kernel.Kernel) error {
		kernel.Add("metric", mon.ProvideCwDaemon())
		return nil
	})
}
