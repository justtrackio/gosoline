package mon

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/jonboulle/clockwork"
	"io"
	"os"
	"sync"
)

func NewLogger(config cfg.Config, tags Tags) Logger {
	level := config.GetString("log_level")
	format := config.GetString("log_format")
	configTags := config.GetStringMapString("log_tags")

	if len(level) == 0 {
		level = Info
	}

	if len(format) == 0 {
		format = FormatGelf
	}

	for k, v := range configTags {
		tags[k] = v
	}

	configValues := ConfigValues{}
	for _, k := range config.AllKeys() {
		configValues[k] = config.Get(k)
	}

	sentryHook := NewSentryHook(config)
	metricHook := NewMetricHook()

	logger := NewLoggerWithInterfaces(clockwork.NewRealClock(), os.Stdout, level, format, tags, configValues)
	logger.addHook(sentryHook)
	logger.addHook(metricHook)

	return logger
}

func NewLoggerWithInterfaces(clock clockwork.Clock, out io.Writer, level string, format string, tags Tags, configValues ConfigValues) *logger {
	fields := make(Fields)
	for k, v := range tags {
		fields[k] = v
	}

	logger := &logger{
		clock:         clock,
		output:        out,
		outputLck:     &sync.Mutex{},
		hooks:         make([]LoggerHook, 0),
		level:         levelPrio(level),
		format:        format,
		tags:          tags,
		configValues:  configValues,
		channel:       ChannelDefault,
		fields:        fields,
		contextFields: ContextFields{},
		ecsLck:        &sync.Mutex{},
		ecsAvailable:  false,
		ecsMetadata:   make(EcsMetadata),
	}

	logger.checkEcsMetadataAvailability()
	logger.readEcsMetadata()

	return logger
}
