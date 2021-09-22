package lambda

import (
	"context"
	"os"

	awsLambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type HandlerFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (interface{}, error)

func Start(handlerFactory HandlerFactory, configOptions ...cfg.Option) {
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
	mergedConfigOptions := append([]cfg.Option{
		cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		cfg.WithSanitizers(cfg.TimeSanitizer),
		cfg.WithErrorHandlers(func(msg string, args ...interface{}) {
			logger.Error(msg, args...)
			os.Exit(1)
		}),
	}, configOptions...)

	config := cfg.New()
	if err := config.Option(mergedConfigOptions...); err != nil {
		logger.Error("failed to apply config options: %w", err)

		os.Exit(1)
	}

	if cfgPostProcessors, err := cfg.ApplyPostProcessors(config); err != nil {
		logger.Error("can not apply post processor on config: %w", err)

		os.Exit(1)
	} else {
		for name, priority := range cfgPostProcessors {
			logger.Debug("applied priority %d config post processor '%s'", priority, name)
		}
	}

	stream.AddDefaultEncodeHandler(log.NewMessageWithLoggingFieldsEncoder(config, logger))

	ctx := appctx.WithContainer(context.Background())

	// create handler function and give lambda control
	lambdaHandler, err := handlerFactory(ctx, config, logger)
	if err != nil {
		logger.Error("failed to create lambda handler: %w", err)

		os.Exit(1)
	}

	awsLambda.Start(lambdaHandler)
}
