package lambda

import (
	"os"
	"strings"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/stream"
	awsLambda "github.com/aws/aws-lambda-go/lambda"
)

type HandlerFactory func(config cfg.Config, logger log.Logger) (interface{}, error)

func Start(handlerFactory HandlerFactory, defaultConfig ...map[string]interface{}) {
	clock.WithUseUTC(true)

	logHandler := log.NewHandlerIoWriter(log.LevelInfo, []string{}, log.FormatterConsole, "", os.Stdout)
	loggerOptions := []log.Option{
		log.WithHandlers(logHandler),
		log.WithContextFieldsResolver(log.ContextLoggerFieldsResolver),
	}

	logger := log.NewLogger()

	if err := logger.Option(loggerOptions...); err != nil {
		logger.Error("failed to apply logger options: %w", err)
		os.Exit(1)
	}

	// configure and create config
	configOptions := []cfg.Option{
		cfg.WithEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
		cfg.WithSanitizers(cfg.TimeSanitizer),
		cfg.WithErrorHandlers(func(msg string, args ...interface{}) {
			logger.Error(msg, args...)
			os.Exit(1)
		}),
	}

	for _, defaults := range defaultConfig {
		configOptions = append(configOptions, cfg.WithConfigMap(defaults))
	}

	config := cfg.New()
	if err := config.Option(configOptions...); err != nil {
		logger.Error("failed to apply logger options: %w", err)
		os.Exit(1)
	}

	stream.AddDefaultEncodeHandler(log.NewMessageWithLoggingFieldsEncoder(config, logger))

	// create handler function and give lambda control
	lambdaHandler, err := handlerFactory(config, logger)
	if err != nil {
		logger.Error("failed to create lambda handler: %w", err)
		os.Exit(1)
	}

	awsLambda.Start(lambdaHandler)
}
