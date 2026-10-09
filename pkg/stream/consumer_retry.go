package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func newConsumerSingleRetryHandler(ctx context.Context, config cfg.Config, logger log.Logger, input Input, settings *ConsumerRetrySettings, name string) (Input, RetryHandler, error) {
	var err error
	var retryInput Input
	var retryHandler RetryHandler

	if retryingInput, ok := input.(RetryingInput); ok {
		settings.Enabled = true

		retryInput, retryHandler = retryingInput.GetRetryHandler()

		return retryInput, retryHandler, nil
	}

	if retryInput, retryHandler, err = NewRetryHandler(ctx, config, logger, settings, name); err != nil {
		return nil, nil, fmt.Errorf("can not create retry handler: %w", err)
	}

	return retryInput, retryHandler, nil
}

// newConsumerBatchRetryHandler creates configured retries independently of primary-input redelivery.
func newConsumerBatchRetryHandler(ctx context.Context, config cfg.Config, logger log.Logger, _ Input, settings *ConsumerRetrySettings, name string) (Input, RetryHandler, error) {
	// Primary messages are acknowledged before business processing, so their
	// transport's native redelivery cannot own batch retries.
	return NewRetryHandler(ctx, config, logger, settings, name)
}
