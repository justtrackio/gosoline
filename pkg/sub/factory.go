package sub

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"time"
)

type Subscription struct {
	Input       string                `cfg:"input"`
	Output      string                `cfg:"output"`
	RunnerCount int                   `cfg:"runner_count" default:"10" validate:"min=1"`
	SourceModel SubscriptionModel     `cfg:"source"`
	TargetModel SubscriptionModel     `cfg:"target"`
	Backoff     cloud.BackoffSettings `cfg:"backoff"`
}

type SubscriptionModel struct {
	Family      string `cfg:"family"`
	Application string `cfg:"application"`
	Name        string `cfg:"name"`
}

func NewGenericTransformer(transformer ModelTransformer) func(cfg.Config, mon.Logger) ModelTransformer {
	return func(_ cfg.Config, _ mon.Logger) ModelTransformer {
		return transformer
	}
}

func NewSubscriberFactory(transformer TransformerMapTypeVersionFactories) kernel.ModuleFactory {
	return func(config cfg.Config, logger mon.Logger) (map[string]kernel.Module, error) {
		return SubscriberFactory(config, logger, transformer)
	}
}

func SubscriberFactory(config cfg.Config, logger mon.Logger, transformerMapType TransformerMapTypeVersionFactories) (map[string]kernel.Module, error) {
	modules := make(map[string]kernel.Module)
	subscriptions := make([]Subscription, 0)

	config.UnmarshalKey("subscriptions", &subscriptions)

	for _, s := range subscriptions {
		sourceModelId := mdl.ModelId{
			Family:      s.SourceModel.Family,
			Application: s.SourceModel.Application,
			Name:        s.SourceModel.Name,
		}
		sourceModelId.PadFromConfig(config)

		targetModelId := mdl.ModelId{
			Family:      s.TargetModel.Family,
			Application: s.TargetModel.Application,
			Name:        s.TargetModel.Name,
		}

		if targetModelId.Name == "" {
			targetModelId.Name = s.SourceModel.Name
		}

		targetModelId.PadFromConfig(config)

		settings := Settings{
			Type:          s.Output,
			RunnerCount:   s.RunnerCount,
			SourceModelId: sourceModelId,
			TargetModelId: targetModelId,
			Backoff:       s.Backoff,
		}

		input, err := getInputByType(config, logger, s, sourceModelId)
		if err != nil {
			logger.Error(err, "could not build subscribers")
			return modules, err
		}

		output, err := getOutputByType(s.Output)
		if err != nil {
			logger.Error(err, "could not build subscribers")
			return modules, err
		}

		modelId := sourceModelId.String()
		if _, ok := transformerMapType[modelId]; !ok {
			err := fmt.Errorf("there is no transformer for modelId %s", modelId)
			logger.Errorf(err, "missing transformer for SubscriberFactory of type %s", s.Output)

			return modules, err
		}
		transformerMapVersion := transformerMapType[modelId]

		subscriber := NewSubscriber(logger, input, output, transformerMapVersion, settings)

		name := fmt.Sprintf("sub_%s_%s_%s_%s", s.Output, sourceModelId.Family, sourceModelId.Application, sourceModelId.Name)
		modules[name] = subscriber
	}

	return modules, nil
}

func getInputByType(config cfg.Config, logger mon.Logger, sub Subscription, mId mdl.ModelId) (stream.Input, error) {
	switch sub.Input {
	case "sns":
		inputSettings := stream.SnsInputSettings{
			QueueId:     mId.Name,
			WaitTime:    5,
			RunnerCount: sub.RunnerCount,
			Backoff: cloud.BackoffSettings{
				Enabled:     true,
				Blocking:    true,
				CancelDelay: time.Second * 6,
			},
		}
		inputTargets := []stream.SnsInputTarget{
			{
				AppId: cfg.AppId{
					Project:     mId.Project,
					Environment: mId.Environment,
					Family:      mId.Family,
					Application: mId.Application,
				},
				TopicId: mId.Name,
			},
		}

		return stream.NewSnsInput(config, logger, inputSettings, inputTargets), nil
	}

	return nil, fmt.Errorf("there is no input defined of type %s", sub.Input)
}

func getOutputByType(outType string) (Output, error) {
	switch outType {
	case "blob":
		return &subOutBlob{}, nil
	case "db":
		return &subOutDb{}, nil
	case "ddb":
		return &subOutDdb{}, nil
	case "kvstore":
		return &subOutKvstore{}, nil
	}

	return nil, fmt.Errorf("there is no output defined of type %s", outType)
}
