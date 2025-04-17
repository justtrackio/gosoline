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

//go:generate go run github.com/vektra/mockery/v2 --name RetryHandler
type RetryHandler interface {
	Put(ctx context.Context, msg *Message) error
}

type RetryHandlerSettings struct {
	After       time.Duration `cfg:"after" default:"1m"`
	MaxAttempts int           `cfg:"max_attempts" default:"3"`
}

type RetrySettings struct {
	Enabled   bool          `cfg:"enabled"`
	Type      string        `cfg:"type" default:"sqs"`
	GraceTime time.Duration `cfg:"grace_time" default:"10s"`
}

type RetryMetadata struct {
	name           string
	retryConfigKey string
	retrySettings  *RetrySettings
}

type RetryHandlerFactory func(ctx context.Context, config cfg.Config, logger log.Logger, md RetryMetadata) (Input, RetryHandler, error)

var retryHandlers = map[string]RetryHandlerFactory{}

func NewRetryHandler(ctx context.Context, config cfg.Config, logger log.Logger, md RetryMetadata) (Input, RetryHandler, error) {
	var ok bool
	var factory RetryHandlerFactory

	if !md.retrySettings.Enabled {
		return NewRetryHandlerNoop(ctx, config, logger, md)
	}

	if factory, ok = retryHandlers[md.retrySettings.Type]; !ok {
		return nil, nil, fmt.Errorf("there is no retry handler of type %s available", md.retrySettings.Type)
	}

	return factory(ctx, config, logger, md)
}

func ConfigurableConsumerRetryKey(name string) string {
	return fmt.Sprintf("%s.retry", ConfigurableConsumerKey(name))
}

func ConfigurableProducerRetryKey(name string) string {
	return fmt.Sprintf("%s.retry", ConfigurableProducerKey(name))
}
