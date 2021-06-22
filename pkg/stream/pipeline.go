package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/metric"
	"sync"
	"time"
)

const MetricNamePipelineReceivedCount = "PipelineReceivedCount"
const MetricNamePipelineProcessedCount = "PipelineProcessedCount"

//go:generate mockery --name PipelineCallback
type PipelineCallback interface {
	Process(ctx context.Context, messages []*Message) ([]*Message, error)
}

type PipelineCallbackFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (PipelineCallback, error)

type PipelineSettings struct {
	Interval  time.Duration
	BatchSize int
}

type Pipeline struct {
	kernel.EssentialModule
	kernel.ApplicationStage
	ConsumerAcknowledge

	metric   metric.Writer
	cfn      coffin.Coffin
	lck      sync.Mutex
	encoder  MessageEncoder
	output   Output
	ticker   *time.Ticker
	batch    []*Message
	stages   []PipelineCallback
	settings *PipelineSettings
}

func NewPipeline(stageFactories ...PipelineCallbackFactory) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var stages = make([]PipelineCallback, len(stageFactories))

		for i, factory := range stageFactories {
			if stages[i], err = factory(ctx, config, logger); err != nil {
				return nil, fmt.Errorf("can't build stage of type %T: %w", factory, err)
			}
		}

		defaults := getDefaultPipelineMetrics()
		metric := metric.NewDaemonWriter(defaults...)

		input, err := NewConfigurableInput(config, logger, "pipeline")
		if err != nil {
			return nil, fmt.Errorf("can not create configurable input: %w", err)
		}

		output, err := NewConfigurableOutput(config, logger, "pipeline")
		if err != nil {
			return nil, fmt.Errorf("can not create configurable output: %w", err)
		}

		settings := &PipelineSettings{
			Interval:  config.GetDuration("pipeline_interval") * time.Second,
			BatchSize: config.GetInt("pipeline_batch_size"),
		}

		return NewPipelineWithInterfaces(logger, metric, input, output, settings, stages...)
	}
}

func NewPipelineWithInterfaces(logger log.Logger, metric metric.Writer, input Input, output Output, settings *PipelineSettings, stages ...PipelineCallback) (*Pipeline, error) {
	pipeline := &Pipeline{
		cfn: coffin.New(),
		ConsumerAcknowledge: ConsumerAcknowledge{
			logger: logger,
			input:  input,
		},
		metric:   metric,
		output:   output,
		encoder:  NewMessageEncoder(&MessageEncoderSettings{}),
		ticker:   time.NewTicker(settings.Interval),
		batch:    make([]*Message, 0, settings.BatchSize),
		settings: settings,
		stages:   stages,
	}

	return pipeline, nil
}

func (p *Pipeline) Run(ctx context.Context) error {
	defer p.logger.Info("leaving pipeline")
	defer p.process(ctx, true)

	p.cfn.GoWithContextf(ctx, p.read, "panic during consuming")
	p.cfn.GoWithContextf(ctx, p.input.Run, "panic during run of the consumer input")

	for {
		select {
		case <-ctx.Done():
			p.input.Stop()
			return p.cfn.Wait()

		case <-p.cfn.Dead():
			p.input.Stop()
			return p.cfn.Err()
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

			disaggregated, err := p.disaggregateMessage(ctx, msg)

			if err != nil {
				p.logger.Error("can not disaggregate the message: %w", err)
				continue
			}

			p.batch = append(p.batch, disaggregated...)
			p.Acknowledge(ctx, msg)

		case <-p.ticker.C:
			force = true
		}

		if len(p.batch) >= p.settings.BatchSize || force {
			p.process(ctx, force)
		}
	}
}

func (p *Pipeline) disaggregateMessage(ctx context.Context, msg *Message) ([]*Message, error) {
	if _, ok := msg.Attributes[AttributeAggregate]; !ok {
		return []*Message{msg}, nil
	}

	batch := make([]*Message, 0)
	_, _, err := p.encoder.Decode(ctx, msg, &batch)

	if err != nil {
		return nil, fmt.Errorf("can not decode message: %w", err)
	}

	return batch, nil
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

	p.metric.WriteOne(&metric.Datum{
		MetricName: MetricNamePipelineReceivedCount,
		Value:      float64(batchSize),
	})

	defer func() {
		p.ticker = time.NewTicker(p.settings.Interval)
		p.batch = make([]*Message, 0, p.settings.BatchSize)

		err := coffin.ResolveRecovery(recover())

		if err == nil {
			return
		}

		p.logger.Error("panic when processing batch: %w", err)
	}()

	p.ticker.Stop()

	var err error

	for _, stage := range p.stages {
		p.batch, err = stage.Process(ctx, p.batch)

		if err != nil {
			p.logger.Error("could not process the batch: %w", err)
		}
	}

	err = p.output.Write(ctx, MessagesToWritableMessages(p.batch))

	if err != nil {
		p.logger.Error("could not write messages to output: %w", err)

		return
	}

	processedCount := len(p.batch)

	p.logger.Info("pipeline processed %d of %d messages", processedCount, batchSize)
	p.metric.WriteOne(&metric.Datum{
		MetricName: MetricNamePipelineProcessedCount,
		Value:      float64(processedCount),
	})
}

func getDefaultPipelineMetrics() metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricNamePipelineReceivedCount,
			Unit:       metric.UnitCount,
			Value:      0.0,
		}, {
			Priority:   metric.PriorityHigh,
			MetricName: MetricNamePipelineProcessedCount,
			Unit:       metric.UnitCount,
			Value:      0.0,
		},
	}
}
