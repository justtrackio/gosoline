package lambda

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	awsLambda "github.com/aws/aws-lambda-go/lambda"
	"strings"
)

type Handler func(config cfg.Config, logger mon.Logger) interface{}

func Start(handler Handler, defaultConfig ...map[string]interface{}) {
	clock.WithUseUTC(true)

	// configure logger
	loggerOptions := []mon.LoggerOption{
		// logs for lambda functions already provide timestamps, so we don't need these
		mon.WithContextFieldsResolver(mon.ContextLoggerFieldsResolver),
		mon.WithStdoutOutput(mon.FormatConsole, mon.AllLogLevels()),
	}

	logger := mon.NewLogger()
	if err := logger.Option(loggerOptions...); err != nil {
		logger.Fatal(err, "failed to apply logger options")
	}

	// configure and create config
	configOptions := []cfg.Option{
		cfg.WithEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
		cfg.WithSanitizers(cfg.TimeSanitizer),
		cfg.WithErrorHandlers(logger.Fatalf),
	}

	for _, defaults := range defaultConfig {
		configOptions = append(configOptions, cfg.WithConfigMap(defaults))
	}

	config := cfg.New()
	if err := config.Option(configOptions...); err != nil {
		logger.Fatal(err, "failed to apply config options")
	}

	stream.AddDefaultEncodeHandler(mon.NewMessageWithLoggingFieldsEncoder(config, logger))

	// create handler function and give lambda control
	lambdaHandler := handler(config, logger)
	awsLambda.Start(lambdaHandler)
}
