package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	AttributeRetry   = "goso.retry"
	AttributeRetryId = "goso.retry.id"
)

//go:generate mockery --name RetryHandler
type RetryHandler interface {
	Put(ctx context.Context, msg *Message) error
}

type RetryHandlerSettings struct {
	After       time.Duration `cfg:"after" default:"1m"`
	MaxAttempts int           `cfg:"max_attempts" default:"3"`
}

type RetryHandlerFactory func(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, RetryHandler, error)

var retryHandlers = map[string]RetryHandlerFactory{}

func NewRetryHandler(ctx context.Context, config cfg.Config, logger log.Logger, consumerSettings *ConsumerRetrySettings, name string) (Input, RetryHandler, error) {
	var ok bool
	var factory RetryHandlerFactory

	if !consumerSettings.Enabled {
		return NewRetryHandlerNoop(ctx, config, logger, name)
	}

	if factory, ok = retryHandlers[consumerSettings.Type]; !ok {
		return nil, nil, fmt.Errorf("there is no retry handler of type %s available", consumerSettings.Type)
	}

	return factory(ctx, config, logger, name)
}

func ConfigurableConsumerRetryKey(name string) string {
	return fmt.Sprintf("%s.retry", ConfigurableConsumerKey(name))
}
