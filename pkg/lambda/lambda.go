package lambda

import (
	"context"

	awsLambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type HandlerFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (any, error)

func Start(handlerFactory HandlerFactory, configOptions ...cfg.Option) {
	var err error
	var cfgPostProcessors map[string]int
	var handlers []log.Handler
	var lambdaHandler any

	clock.WithUseUTC(true)
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	logger := log.NewLogger()

	// configure and create config
	mergedConfigOptions := append([]cfg.Option{
		cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		cfg.WithSanitizers(cfg.TimeSanitizer),
		cfg.WithErrorHandlers(defaultErrorHandler),
	}, configOptions...)

	if err = config.Option(mergedConfigOptions...); err != nil {
		defaultErrorHandler("failed to apply config options: %w", err)

		return
	}

	if cfgPostProcessors, err = cfg.ApplyPostProcessors(config); err != nil {
		defaultErrorHandler("can not apply post processor on config: %w", err)

		return
	}

	if handlers, err = log.NewHandlersFromConfig(config); err != nil {
		defaultErrorHandler("can not create handlers from config: %w", err)

		return
	}

	loggerOptions := []log.Option{
		log.WithHandlers(handlers...),
		log.WithContextFieldsResolver(log.ContextFieldsResolver),
	}

	if err = logger.Option(loggerOptions...); err != nil {
		defaultErrorHandler("failed to apply logger options: %w", err)

		return
	}

	for name, priority := range cfgPostProcessors {
		logger.Debug("applied priority %d config post processor '%s'", priority, name)
	}

	stream.AddDefaultEncodeHandler(log.NewMessageWithLoggingFieldsEncoder(config, logger))

	// create handler function and give lambda control
	if lambdaHandler, err = handlerFactory(ctx, config, logger); err != nil {
		defaultErrorHandler("failed to create lambda handler: %w", err)

		return
	}

	awsLambda.Start(lambdaHandler)
}
