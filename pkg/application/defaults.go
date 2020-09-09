package application

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"strings"
)

var DefaultMinimalAppOptions = []Option{
	WithUTCClock(true),
	WithConfigErrorHandlers(defaultErrorHandler),
	WithConfigFile("./config.dist.yml", "yml"),
	WithConfigFileFlag,
	WithConfigEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	WithConfigSanitizers(cfg.TimeSanitizer),
	WithLoggerApplicationTag,
	WithLoggerTagsFromConfig,
	WithLoggerSettingsFromConfig,
	WithLoggerContextFieldsMessageEncoder(),
	WithLoggerContextFieldsResolver(mon.ContextLoggerFieldsResolver),
	WithKernelSettingsFromConfig,
}

var DefaultServiceAppOptions = append(DefaultMinimalAppOptions, []Option{
	WithConfigServer,
	WithLoggerMetricHook,
	WithLoggerSentryHook(mon.SentryExtraConfigProvider, mon.SentryExtraEcsMetadataProvider),
	WithApiHealthCheck,
	WithMetricDaemon,
	WithProducerDaemon,
	WithTracing,
}...)

var DefaultCliApp = append(DefaultMinimalAppOptions, []Option{}...)
