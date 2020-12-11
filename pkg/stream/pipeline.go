package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"sync"
	"time"
)

const MetricNamePipelineReceivedCount = "PipelineReceivedCount"
const MetricNamePipelineProcessedCount = "PipelineProcessedCount"

//go:generate mockery -name PipelineCallback
type PipelineCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	Process(ctx context.Context, messages []PipelineInput) ([]PipelineOutput, error)
}

type PipelineInput struct {
	attributes map[string]interface{}
	Messages   []*Message
}

// Use this function to transform each of your input batches to an output batch.
// This function ensures we can trace each message through the pipeline and finally
// acknowledge the original message should it produce any (even an empty) set of messages.
// To retry a message later, simply do not include it as a PipelineOutput in your result.
func (i *PipelineInput) CreateOutput(messages []*Message) PipelineOutput {
	return PipelineOutput{
		attributes: i.attributes,
		messages:   messages,
	}
}

type PipelineOutput struct {
	attributes map[string]interface{}
	messages   []*Message
}

type PipelineSettings struct {
	Interval  time.Duration
	BatchSize int
}

type Pipeline struct {
	kernel.EssentialModule
	kernel.ApplicationStage
	ConsumerAcknowledge

	metric   mon.MetricWriter
	cfn      coffin.Coffin
	lck      sync.Mutex
	encoder  MessageEncoder
	output   Output
	ticker   *time.Ticker
	batch    []PipelineInput
	stages   []PipelineCallback
	settings *PipelineSettings
}

func NewPipeline(stages ...PipelineCallback) *Pipeline {
	return &Pipeline{
		cfn:    coffin.New(),
		stages: stages,
	}
}

func (p *Pipeline) Boot(config cfg.Config, logger mon.Logger) error {
	for _, stage := range p.stages {
		err := stage.Boot(config, logger)

		if err != nil {
			return err
		}
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
	p.encoder = NewMessageEncoder(&MessageEncoderSettings{})
	p.ticker = time.NewTicker(settings.Interval)
	p.batch = make([]PipelineInput, 0, settings.BatchSize)
	p.settings = settings

	return nil
}

func (p *Pipeline) Run(ctx context.Context) error {
	defer p.logger.Info("leaving pipeline")
	defer p.process(ctx, true)

	p.cfn.GoWithContextf(ctx, p.input.Run, "panic during run of the consumer input")
	p.cfn.GoWithContextf(ctx, p.read, "panic during consuming")

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
				p.logger.Errorf(err, "can not disaggregate the message")
				continue
			}

			p.batch = append(p.batch, PipelineInput{
				attributes: msg.Attributes,
				Messages:   disaggregated,
			})

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

	p.metric.WriteOne(&mon.MetricDatum{
		MetricName: MetricNamePipelineReceivedCount,
		Value:      float64(batchSize),
	})

	defer func() {
		p.ticker = time.NewTicker(p.settings.Interval)
		p.batch = make([]PipelineInput, 0, p.settings.BatchSize)

		err := coffin.ResolveRecovery(recover())

		if err == nil {
			return
		}

		p.logger.Error(err, "panic when processing batch")
	}()

	p.ticker.Stop()

	input := p.batch

	var err error
	var output []PipelineOutput

	for i, stage := range p.stages {
		output, err = stage.Process(ctx, input)

		if err != nil {
			p.logger.Error(err, "could not process the batch")
		}

		if i != len(p.stages)-1 {
			input = make([]PipelineInput, len(output))

			for j, outputBatch := range output {
				input[j] = PipelineInput{
					attributes: outputBatch.attributes,
					Messages:   outputBatch.messages,
				}
			}
		}
	}

	writableBatch := make([]WritableMessage, 0, len(output))
	acknowledgeableBatch := make([]*Message, len(output))

	for i, records := range output {
		for _, record := range records.messages {
			writableBatch = append(writableBatch, record)
		}
		acknowledgeableBatch[i] = &Message{
			Attributes: records.attributes,
		}
	}

	err = p.output.Write(ctx, writableBatch)

	if err != nil {
		p.logger.Error(err, "could not write messages to output")

		return
	}

	p.AcknowledgeBatch(ctx, acknowledgeableBatch)

	processedCount := len(output)

	p.logger.Infof("pipeline processed %d of %d messages", processedCount, batchSize)
	p.metric.WriteOne(&mon.MetricDatum{
		MetricName: MetricNamePipelineProcessedCount,
		Value:      float64(processedCount),
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
