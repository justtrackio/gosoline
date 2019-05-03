package sub

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

type Subscription struct {
	Input  string            `mapstructure:"input"`
	Output string            `mapstructure:"output"`
	Redis  string            `mapstructure:"redis"`
	Model  SubscriptionModel `mapstructure:"model"`
}

type SubscriptionModel struct {
	Family      string `mapstructure:"family"`
	Application string `mapstructure:"application"`
	Name        string `mapstructure:"name"`
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

	config.Unmarshal("subscriptions", &subscriptions)

	for _, s := range subscriptions {
		mId := mdl.ModelId{
			Family:      s.Model.Family,
			Application: s.Model.Application,
			Name:        s.Model.Name,
		}
		mId.PadFromConfig(config)

		settings := Settings{
			Type:    s.Output,
			ModelId: mId,
		}

		input, err := getInputByType(config, logger, s.Input, mId)
		if err != nil {
			logger.Error(err, "could not build subscribers")
			return modules, err
		}

		output, err := getOutputByType(s.Output)
		if err != nil {
			logger.Error(err, "could not build subscribers")
			return modules, err
		}

		modelId := mId.String()
		if _, ok := transformerMapType[modelId]; !ok {
			err := fmt.Errorf("there is no transformer for modelId %s", modelId)
			logger.Errorf(err, "missing transformer for SubscriberFactory of type %s", s.Output)

			return modules, err
		}
		transformerMapVersion := transformerMapType[modelId]

		subscriber := NewSubscriber(input, output, transformerMapVersion, settings)

		name := fmt.Sprintf("sub_%s_%s_%s_%s", s.Output, mId.Family, mId.Application, mId.Name)
		modules[name] = subscriber
	}

	return modules, nil
}

func getInputByType(config cfg.Config, logger mon.Logger, inType string, mId mdl.ModelId) (stream.Input, error) {
	switch inType {
	case "sns":
		inputSettings := stream.SnsInputSettings{
			QueueId:  mId.Name,
			WaitTime: 5,
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
	}

	return nil, fmt.Errorf("there is no output defined of type %s", outType)
}
