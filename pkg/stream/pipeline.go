package stream

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"gopkg.in/tomb.v2"
	"sync"
	"time"
)

type PipelineCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	Process(ctx context.Context, messages []*Message) ([]*Message, error)
}

type PipelineSettings struct {
	Interval  time.Duration
	BatchSize int
}

type Pipeline struct {
	kernel.ForegroundModule

	logger   mon.Logger
	tmb      tomb.Tomb
	lck      sync.Mutex
	input    Input
	output   Output
	ticker   *time.Ticker
	batch    []*Message
	callback PipelineCallback
	settings *PipelineSettings
}

func NewPipeline(callback PipelineCallback) *Pipeline {
	return &Pipeline{
		callback: callback,
	}
}

func (p *Pipeline) Boot(config cfg.Config, logger mon.Logger) error {
	err := p.callback.Boot(config, logger)

	if err != nil {
		return err
	}

	input := NewConfigurableInput(config, logger, "pipeline")
	output := NewConfigurableOutput(config, logger, "pipeline")

	settings := &PipelineSettings{
		Interval:  config.GetDuration("pipeline_interval") * time.Second,
		BatchSize: config.GetInt("pipeline_batch_size"),
	}

	return p.BootWithInterfaces(logger, input, output, settings)
}

func (p *Pipeline) BootWithInterfaces(logger mon.Logger, input Input, output Output, settings *PipelineSettings) error {
	p.logger = logger
	p.input = input
	p.output = output
	p.ticker = time.NewTicker(settings.Interval)
	p.batch = make([]*Message, 0, settings.BatchSize)
	p.settings = settings

	return nil
}

func (p *Pipeline) Run(ctx context.Context) error {
	defer p.logger.Info("leaving pipeline")
	defer p.process(ctx, true)

	p.tmb.Go(p.input.Run)
	p.tmb.Go(func() error {
		return p.read(ctx)
	})

	for {
		select {
		case <-ctx.Done():
			p.input.Stop()
			return p.tmb.Wait()

		case <-p.tmb.Dead():
			p.input.Stop()
			return p.tmb.Err()

		case <-p.ticker.C:
			p.process(ctx, true)
		}
	}
}

func (p *Pipeline) read(ctx context.Context) error {
	for {
		msg, ok := <-p.input.Data()

		if !ok {
			return nil
		}

		p.batch = append(p.batch, msg)
		p.process(ctx, false)
	}
}

func (p *Pipeline) process(ctx context.Context, force bool) {
	p.lck.Lock()
	defer p.lck.Unlock()

	batchSize := len(p.batch)

	if batchSize == 0 {
		return
	}

	if batchSize < p.settings.BatchSize && !force {
		return
	}

	defer func() {
		p.ticker = time.NewTicker(p.settings.Interval)
		p.batch = make([]*Message, 0, p.settings.BatchSize)
	}()

	p.ticker.Stop()
	messages, err := p.callback.Process(ctx, p.batch)

	if err != nil {
		p.logger.Error(err, "could not process the batch")
		return
	}

	err = p.output.Write(ctx, messages)

	if err != nil {
		p.logger.Error(err, "could not write messages to output")
	}
}
