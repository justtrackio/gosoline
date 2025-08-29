package main

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHelloWorldModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &HelloWorldModule{
		logger: logger.WithChannel("hello-world"),
	}, nil
}

type HelloWorldModule struct {
	logger  log.Logger
	healthy atomic.Bool
}

func (h *HelloWorldModule) IsHealthy(ctx context.Context) (bool, error) {
	return h.healthy.Load(), nil
}

func (h *HelloWorldModule) Run(ctx context.Context) error {
	timer := clock.NewRealTimer(time.Second * 3)
	<-timer.Chan()

	h.healthy.Store(true)

	h.logger.Info(ctx, "Hello World")

	return nil
}
