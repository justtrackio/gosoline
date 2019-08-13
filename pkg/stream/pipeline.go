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

const MetricNamePipelineReceivedCount = "PipelineReceivedCount"
const MetricNamePipelineProcessedCount = "PipelineProcessedCount"

//go:generate mockery -name PipelineCallback
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
	metric   mon.MetricWriter
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

	defaults := getDefaultPipelineMetrics()
	metric := mon.NewMetricDaemonWriter(defaults...)

	input := NewConfigurableInput(config, logger, "pipeline")
	output := NewConfigurableOutput(config, logger, "pipeline")

	settings := &PipelineSettings{
		Interval:  config.GetDuration("pipeline_interval") * time.Second,
		BatchSize: config.GetInt("pipeline_batch_size"),
	}

	return p.BootWithInterfaces(logger, metric, input, output, settings)
}

func (p *Pipeline) BootWithInterfaces(logger mon.Logger, metric mon.MetricWriter, input Input, output Output, settings *PipelineSettings) error {
	p.logger = logger
	p.metric = metric
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
		}
	}
}

func (p *Pipeline) read(ctx context.Context) error {
	for {
		force := false

		select {
		case msg, ok := <-p.input.Data():
			if !ok {
				return nil
			}

			p.batch = append(p.batch, msg)

		case <-p.ticker.C:
			force = true
		}

		if len(p.batch) >= p.settings.BatchSize || force {
			p.process(ctx, force)
		}
	}
}

func (p *Pipeline) process(ctx context.Context, force bool) {
	p.lck.Lock()
	defer p.lck.Unlock()

	batchSize := len(p.batch)

	if batchSize == 0 {
		p.logger.Info("pipeline has nothing to do")
		return
	}

	if batchSize < p.settings.BatchSize && !force {
		return
	}

	p.metric.WriteOne(&mon.MetricDatum{
		MetricName: MetricNamePipelineReceivedCount,
		Value:      float64(batchSize),
	})

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
		return
	}

	messageCount := len(messages)

	p.logger.Infof("pipeline processed %d of %d messages", messageCount, batchSize)
	p.metric.WriteOne(&mon.MetricDatum{
		MetricName: MetricNamePipelineProcessedCount,
		Value:      float64(messageCount),
	})
}

func getDefaultPipelineMetrics() mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: MetricNamePipelineReceivedCount,
			Unit:       mon.UnitCount,
			Value:      0.0,
		}, {
			Priority:   mon.PriorityHigh,
			MetricName: MetricNamePipelineProcessedCount,
			Unit:       mon.UnitCount,
			Value:      0.0,
		},
	}
}
