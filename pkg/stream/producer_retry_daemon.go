package stream

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	metricNameProducerRetryGetCount = "ProducerRetryGetCount"
	metricNameProducerRetryPutCount = "ProducerRetryPutCount"
)

type ProducerRetryDaemonSettings struct {
	DaemonWriterCount int `cfg:"daemon_writer_count" default:"1" min:"1"`
}

//go:generate mockery --name ProducerRetryDaemon
type ProducerRetryDaemon interface {
	kernel.Module
	RetryOne(ctx context.Context, msg WritableMessage) error
	RetryMany(ctx context.Context, msgs []WritableMessage) error
}

type producerRetryDaemonData struct {
	msg *Message
	src string
}

type producerRetryDaemon struct {
	name string

	logger       log.Logger
	metricWriter metric.Writer
	uuidGen      uuid.Uuid
	stopped      sync.Once
	wg           sync.WaitGroup
	cancel       context.CancelFunc
	data         chan *producerRetryDaemonData

	output            Output
	retryHandler      RetryHandler
	retryInput        Input
	daemonWriterCount int
}

func producerRetryDaemonName(name string) string {
	return fmt.Sprintf("producer-retry-daemon-%s", name)
}

func ProvideProducerRetryDaemon(
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	name string,
	metadata RetryMetadata,
) (ProducerRetryDaemon, error) {
	return appctx.Provide(ctx, producerDaemonKey(producerRetryDaemonName(name)), func() (ProducerRetryDaemon, error) {
		return NewProducerRetryDaemon(ctx, config, logger, name, metadata)
	})
}

func NewProducerRetryDaemon(ctx context.Context, config cfg.Config, logger log.Logger, name string, metadata RetryMetadata) (ProducerRetryDaemon, error) {
	settings := &ProducerRetryDaemonSettings{}
	err := config.UnmarshalKey(fmt.Sprintf("stream.producer.%s.retry", name), settings)
	if err != nil {
		return nil, fmt.Errorf("can not unmarshal producer retry daemon settings: %w", err)
	}

	retryInput, retryHandler, err := NewRetryHandler(ctx, config, logger, metadata)
	if err != nil {
		return nil, fmt.Errorf("can not create retry handler: %w", err)
	}

	confOutput, err := ProvideConfigurableOutput(ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not create retry handler: %w", err)
	}

	return NewProducerRetryDaemonWithInterfaces(
		name,
		logger,
		metric.NewWriter(getProducerDefaultMetrics(name)...),
		uuid.New(),
		retryInput,
		retryHandler,
		confOutput.Output,
		settings.DaemonWriterCount,
	), nil
}

func NewProducerRetryDaemonWithInterfaces(
	name string,
	logger log.Logger,
	metricWriter metric.Writer,
	uuidGen uuid.Uuid,
	input Input,
	retryHandler RetryHandler,
	output Output,
	daemonWriterCount int,
) ProducerRetryDaemon {
	return &producerRetryDaemon{
		name:              name,
		logger:            logger,
		metricWriter:      metricWriter,
		uuidGen:           uuidGen,
		stopped:           sync.Once{},
		data:              make(chan *producerRetryDaemonData),
		output:            output,
		retryHandler:      retryHandler,
		retryInput:        input,
		daemonWriterCount: daemonWriterCount,
	}
}

func (p *producerRetryDaemon) Run(kernelCtx context.Context) error {
	// create ctx whose done channel is closed on dying coffin
	cfn, dyingCtx := coffin.WithContext(context.Background())

	// create ctx whose done channel is closed on dying coffin and manual cancel
	manualCtx := cfn.Context(context.Background())
	manualCtx, p.cancel = context.WithCancel(manualCtx)

	cfn.Go(func() error {
		cfn.GoWithContextf(dyingCtx, p.retryInput.Run, "panic getting retry messages from queue")
		cfn.GoWithContextf(dyingCtx, p.ingestData, "panic digesting producer retry queue")

		// Start worker pool for parallel processing
		p.wg.Add(p.daemonWriterCount)
		for i := 0; i < p.daemonWriterCount; i++ {
			cfn.GoWithContextf(kernelCtx, p.processMessages, "panic during message processing")
		}

		cfn.GoWithContextf(manualCtx, p.stopConsuming, "panic during stopping the consuming")

		cfn.Go(func() error {
			// wait for kernel or coffin cancel...
			select {
			case <-manualCtx.Done():
			case <-kernelCtx.Done():
			}

			// and stop the input
			p.stopIncomingData(kernelCtx)

			return nil
		})

		return nil
	})

	return cfn.Wait()
}

func (p *producerRetryDaemon) RetryOne(ctx context.Context, msg WritableMessage) error {
	return p.retry(ctx, msg)
}

func (p *producerRetryDaemon) RetryMany(ctx context.Context, msgs []WritableMessage) error {
	return p.retry(ctx, msgs...)
}

func (p *producerRetryDaemon) retry(ctx context.Context, messages ...WritableMessage) error {
	for _, writableMessage := range messages {
		message, ok := writableMessage.(*Message)
		if !ok {
			return fmt.Errorf("can not cast messages to message struct")
		}

		retryMsg, retryId := p.buildRetryMessage(message)

		ctx = log.AppendGlobalContextFields(ctx, log.Fields{
			"retry_id": retryId,
		})
		p.logger.Warn(ctx, "putting message with id into retry")

		if err := p.retryHandler.Put(ctx, retryMsg); err != nil {
			return fmt.Errorf("can not write msg to output: %w", err)
		}
		p.writeMetricRetryCount(ctx, metricNameProducerRetryPutCount)
	}

	return nil
}

func (p *producerRetryDaemon) ingestData(ctx context.Context) error {
	defer p.logger.Debug(ctx, "ingestData is ending")
	defer close(p.data)

	cfn := coffin.New()
	cfn.Go(func() error {
		cfn.GoWithContextf(ctx, p.ingestDataFromSource(p.retryInput, dataSourceRetry), "panic during shoveling data from retry")

		return nil
	})

	return cfn.Wait()
}

func (p *producerRetryDaemon) ingestDataFromSource(input Input, src string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		defer p.logger.Debug(ctx, "ingestDataFromSource %s is ending", src)
		defer p.stopIncomingData(ctx)

		for {
			select {
			case <-ctx.Done():
				return nil

			case msg, ok := <-input.Data():
				if !ok {
					return nil
				}

				p.data <- &producerRetryDaemonData{
					msg: msg,
					src: src,
				}
			}
		}
	}
}

func (p *producerRetryDaemon) processMessages(ctx context.Context) error {
	defer p.logger.Debug(ctx, "processMessages is ending")
	defer p.wg.Done()

	ackInput, isAckInput := p.retryInput.(AcknowledgeableInput)
	ackFunc := func(ctx context.Context, msg *Message, ack bool) error {
		if !isAckInput {
			return nil
		}

		return ackInput.Ack(ctx, msg, ack)
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case data, ok := <-p.data:
			if !ok {
				return nil
			}

			p.writeMetricRetryCount(ctx, metricNameProducerRetryGetCount)
			err := p.output.WriteOne(ctx, data.msg)
			if err != nil {
				p.logger.WithFields(log.Fields{
					"error": err,
				}).Warn(ctx, "failed to retry message")
				continue
			}

			if ackErr := ackFunc(ctx, data.msg, true); ackErr != nil {
				p.logger.WithFields(log.Fields{
					"error": ackErr,
				}).Warn(ctx, "failed to ack message")
			}
		}
	}
}

func (p *producerRetryDaemon) stopIncomingData(ctx context.Context) {
	p.stopped.Do(func() {
		defer p.logger.Debug(ctx, "stopIncomingData is ending")

		p.retryInput.Stop(ctx)
	})
}

func (p *producerRetryDaemon) writeMetricRetryCount(ctx context.Context, metricName string) {
	p.metricWriter.Write(ctx, metric.Data{
		&metric.Datum{
			MetricName: metricName,
			Dimensions: map[string]string{
				"Producer": p.name,
			},
			Value: float64(1),
		},
	})
}

func (p *producerRetryDaemon) stopConsuming(ctx context.Context) error {
	p.wg.Wait()
	p.stopIncomingData(ctx)
	p.cancel()
	return nil
}

func (p *producerRetryDaemon) buildRetryMessage(msg *Message) (retryMsg *Message, retryId string) {
	if attrRetryId, ok := msg.Attributes[AttributeRetryId]; ok {
		return msg, attrRetryId
	}

	retryId = p.uuidGen.NewV4()
	retryMsg = &Message{
		Attributes: funk.MergeMaps(msg.Attributes, map[string]string{
			AttributeRetry:   strconv.FormatBool(true),
			AttributeRetryId: retryId,
		}),
		Body: msg.Body,
	}

	return retryMsg, retryId
}

func getProducerDefaultMetrics(name string) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameProducerRetryGetCount,
			Dimensions: map[string]string{
				"Producer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameProducerRetryPutCount,
			Dimensions: map[string]string{
				"Producer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
