package sub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/tracing"
	"time"
)

const (
	MetricNameSuccess = "ModelEventConsumeSuccess"
	MetricNameFailure = "ModelEventConsumeFailure"
)

type Output interface {
	Boot(config cfg.Config, logger mon.Logger, settings Settings) error
	Persist(ctx context.Context, model Model, op string) error
}

type Settings struct {
	Type          string
	RunnerCount   int
	SourceModelId mdl.ModelId
	TargetModelId mdl.ModelId
}

type Subscriber interface {
	GetType() string
	Boot(config cfg.Config, logger mon.Logger) error
	Run(ctx context.Context) error
}

func NewSubscriber(logger mon.Logger, input stream.Input, output Output, transformerFactories TransformerMapVersionFactories, s Settings) Subscriber {
	consumerAck := stream.NewConsumerAcknowledgeWithInterfaces(logger, input)

	return &subscriber{
		ConsumerAcknowledge: consumerAck,
		logger:              logger,
		input:               input,
		output:              output,
		factories:           transformerFactories,
		settings:            s,
	}
}

type subscriber struct {
	kernel.EssentialModule
	stream.ConsumerAcknowledge

	logger mon.Logger
	tracer tracing.Tracer
	cfn    coffin.Coffin
	metric mon.MetricWriter

	settings Settings
	appId    cfg.AppId
	modelId  mdl.ModelId
	name     string

	input  stream.Input
	output Output

	factories TransformerMapVersionFactories
	transform ModelMsgTransformer
}

func (s *subscriber) Boot(config cfg.Config, logger mon.Logger) error {
	s.logger = logger
	s.tracer = tracing.ProviderTracer(config, logger)
	s.cfn = coffin.New()

	s.appId.PadFromConfig(config)
	s.settings.SourceModelId.PadFromConfig(config)

	mId := s.settings.SourceModelId
	outType := s.settings.Type

	s.modelId = mId
	s.name = fmt.Sprintf("%s-%s-%s-%s-%s_subscriber-%v-%v-%v", s.appId.Project, s.appId.Environment, s.appId.Family, s.appId.Application, outType, mId.Family, mId.Application, mId.Name)

	err := s.output.Boot(config, logger, s.settings)

	if err != nil {
		return err
	}

	defaultMetrics := s.getDefaultMetrics()
	s.metric = mon.NewMetricDaemonWriter(defaultMetrics...)

	versionedTransformers := make(TransformerMapVersion)
	for version, fac := range s.factories {
		versionedTransformers[version] = fac(config, logger)
	}
	s.transform = BuildTransformer(versionedTransformers)

	return nil
}

func (s *subscriber) Run(ctx context.Context) error {
	defer s.logger.Infof("leaving subscriber %s", s.name)

	for i := 0; i < s.settings.RunnerCount; i++ {
		s.cfn.Gof(s.consume, "panic during consuming the subscription")
	}

	s.cfn.GoWithContextf(ctx, s.input.Run, "panic during run of the subscription input")

	for {
		select {
		case <-ctx.Done():
			s.input.Stop()
			return s.cfn.Wait()

		case <-s.cfn.Dying():
			s.input.Stop()
			return s.cfn.Err()
		}
	}
}

func (s *subscriber) consume() error {
	for {
		msg, ok := <-s.input.Data()

		if !ok {
			return nil
		}

		s.handleMessage(msg)
	}
}

func (s *subscriber) handleMessage(msg *stream.Message) {
	ctx, model, spec, err := s.transformMessage(msg)

	if err != nil {
		s.logger.Errorf(err, "could not transform message")
		return
	}

	ctx, trans := s.tracer.StartSpanFromContext(ctx, s.name)

	defer s.recover(ctx, msg)
	defer trans.Finish()

	err = s.persist(ctx, model, spec)
	s.writeMetric(err)

	if err == nil {
		s.Acknowledge(ctx, msg)
	}
}

func (s *subscriber) transformMessage(msg *stream.Message) (context.Context, Model, *ModelSpecification, error) {
	ctx := context.Background()
	spec, err := getModelSpecification(msg)

	if err != nil {
		return ctx, nil, nil, fmt.Errorf("can not retrieve model specification from message: %w", err)
	}

	ctx, model, err := s.transform(ctx, spec, msg)

	return ctx, model, spec, err
}

func (s *subscriber) persist(ctx context.Context, model Model, spec *ModelSpecification) error {
	logger := s.logger.WithContext(ctx)

	if model == nil {
		logger.Infof("skipping %s op for subscription for modelId %s and version %d", spec.CrudType, spec.ModelId, spec.Version)
		return nil
	}

	err := s.output.Persist(ctx, model, spec.CrudType)

	if err != nil {
		logger.Errorf(err, "could not persist the model to db %s", spec.ModelId)
		return err
	}

	logger.Infof("persisted %s op for subscription for modelId %s and version %d with id %v", spec.CrudType, spec.ModelId, spec.Version, model.GetId())

	return nil
}

func (s *subscriber) recover(ctx context.Context, msg *stream.Message) {
	err := coffin.ResolveRecovery(recover())

	if err == nil {
		return
	}

	s.logger.WithContext(ctx).WithFields(mon.Fields{
		"body": msg.Body,
	}).Errorf(err, "can not persist model")
}

func (s *subscriber) writeMetric(err error) {
	metricName := MetricNameSuccess

	if err != nil {
		metricName = MetricNameFailure
	}

	s.metric.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"Application": s.appId.Application,
			"ModelId":     s.modelId.String(),
		},
		Unit:  mon.UnitCount,
		Value: 1.0,
	})
}

func (s *subscriber) getDefaultMetrics() []*mon.MetricDatum {
	return []*mon.MetricDatum{
		{
			Priority:   mon.PriorityHigh,
			MetricName: MetricNameSuccess,
			Dimensions: map[string]string{
				"Application": s.appId.Application,
				"ModelId":     s.modelId.String(),
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   mon.PriorityHigh,
			MetricName: MetricNameFailure,
			Dimensions: map[string]string{
				"Application": s.appId.Application,
				"ModelId":     s.modelId.String(),
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}
