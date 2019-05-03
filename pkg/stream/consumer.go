package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"gopkg.in/tomb.v2"
	"time"
)

//go:generate mockery -name=ConsumerCallback
type ConsumerCallback interface {
	Boot(config cfg.Config, logger mon.Logger)
	Consume(ctx context.Context, msg *Message) error
}

type baseConsumer struct {
	kernel.ForegroundModule

	logger mon.Logger
	tracer tracing.Tracer

	input  Input
	tmb    tomb.Tomb
	ticker *time.Ticker

	name      string
	callback  ConsumerCallback
	processed int
}

func (c *baseConsumer) Boot(config cfg.Config, logger mon.Logger) error {
	c.logger = logger
	c.tracer = tracing.NewAwsTracer(config)

	idleTimeout := config.GetDuration("consumer_idle_timeout")
	c.ticker = time.NewTicker(idleTimeout * time.Second)

	c.callback.Boot(config, logger)

	return nil
}

func (c *baseConsumer) Run(ctx context.Context) error {
	defer c.logger.Info("leaving consumer ", c.name)

	c.tmb.Go(c.input.Run)
	c.tmb.Go(c.consume)

	for {
		select {
		case <-ctx.Done():
			c.input.Stop()
			return c.tmb.Wait()

		case <-c.tmb.Dead():
			c.input.Stop()
			return c.tmb.Err()

		case <-c.ticker.C:
			c.logger.Info(fmt.Sprintf("processed %v messages", c.processed))
			c.processed = 0
		}
	}
}

func (c *baseConsumer) consume() error {
	for {
		msg, ok := <-c.input.Data()

		if !ok {
			return nil
		}

		c.doCallback(msg)
		c.processed++
	}
}

func (c *baseConsumer) doCallback(msg *Message) {
	ctx, trans := c.tracer.StartSpanFromTraceAble(msg, c.name)
	defer trans.Finish()

	err := c.callback.Consume(ctx, msg)

	if err != nil {
		c.logger.Error(err, "an error occurred during the consume operation")
	}
}
