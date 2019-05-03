package sub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/tracing"
	"gopkg.in/tomb.v2"
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
	Type    string
	ModelId mdl.ModelId
}

type Subscriber interface {
	GetType() string
	Boot(config cfg.Config, logger mon.Logger) error
	Run(ctx context.Context) error
}

func NewSubscriber(input stream.Input, output Output, transformerFactories TransformerMapVersionFactories, s Settings) Subscriber {
	return &subscriber{
		input:     input,
		output:    output,
		factories: transformerFactories,
		settings:  s,
	}
}

type subscriber struct {
	kernel.ForegroundModule

	logger mon.Logger
	tracer tracing.Tracer
	tmb    tomb.Tomb
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
	s.tracer = tracing.NewAwsTracer(config)

	s.appId.PadFromConfig(config)
	s.settings.ModelId.PadFromConfig(config)

	mId := s.settings.ModelId
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

	s.tmb.Go(s.input.Run)
	s.tmb.Go(s.consume)

	for {
		select {
		case <-ctx.Done():
			s.input.Stop()
			return s.tmb.Wait()

		case <-s.tmb.Dead():
			s.input.Stop()
			return s.tmb.Err()
		}
	}
}

func (s *subscriber) consume() error {
	for {
		msg, ok := <-s.input.Data()

		if !ok {
			return nil
		}

		err := s.persist(msg)
		s.writeMetric(err)
	}
}

func (s *subscriber) persist(msg *stream.Message) error {
	ctx, trans := s.tracer.StartSpanFromTraceAble(msg, s.name)
	defer trans.Finish()

	logger := s.logger.WithContext(ctx)
	modelMsg, err := stream.CreateModelMsg(msg)

	if err != nil {
		logger.Error(err, "the msg has invalid model information")
		return err
	}

	model, err := s.transform(ctx, modelMsg)

	if err != nil {
		logger.Errorf(err, "could not transform the msg to a model %s", modelMsg.ModelId)
		return err
	}

	err = s.output.Persist(ctx, model, modelMsg.CrudType)

	if err != nil {
		logger.Errorf(err, "could not persist the model to ddb %s", modelMsg.ModelId)
		return err
	}

	logger.Infof("persisted %s op for subscription for modelId %s and version %d with id %v", modelMsg.CrudType, modelMsg.ModelId, modelMsg.Version, model.GetId())

	return nil
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
