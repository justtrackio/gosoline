package sub

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

type inputSettings map[string]interface{}

type Subscription struct {
	Input         string            `cfg:"input"`
	InputSettings inputSettings     `cfg:"input_settings"`
	Output        string            `cfg:"output"`
	Redis         string            `cfg:"redis"`
	SourceModel   SubscriptionModel `cfg:"source"`
	TargetModel   SubscriptionModel `cfg:"target"`
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

		targetModelId := sourceModelId
		if s.TargetModel.Family != "" {
			targetModelId.Family = s.TargetModel.Family
		}
		if s.TargetModel.Application != "" {
			targetModelId.Application = s.TargetModel.Application
		}
		if s.TargetModel.Name != "" {
			targetModelId.Name = s.TargetModel.Name
		}

		settings := Settings{
			Type:          s.Output,
			SourceModelId: sourceModelId,
			TargetModelId: targetModelId,
		}

		input, err := getInputByType(config, logger, s.Input, s.InputSettings, sourceModelId)
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

func getInputByType(config cfg.Config, logger mon.Logger, inType string, inputSettings inputSettings, mId mdl.ModelId) (stream.Input, error) {
	switch inType {
	case "sns":
		waitTime := int64(5)

		if wt, ok := inputSettings["wait_time"].(int); ok {
			waitTime = int64(wt)
		}

		snsInputSettings := stream.SnsInputSettings{
			QueueId:  mId.Name,
			WaitTime: waitTime,
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

		return stream.NewSnsInput(config, logger, snsInputSettings, inputTargets), nil
	}

	return nil, fmt.Errorf("there is no input defined of type %s", inType)
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
