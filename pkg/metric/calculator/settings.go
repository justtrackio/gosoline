package calculator

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/metric"
)

type EcsSettings struct {
	Cluster string `cfg:"cluster" default:"{app_project}-{env}-{app_family}"`
	Service string `cfg:"service" default:"{app_group}-{app_name}"`
}

type DynamoDbSettings struct {
	Naming DynamoDbNamingSettings `cfg:"naming"`
}

type DynamoDbNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{modelId}"`
}

type CloudWatchSettings struct {
	Client string `cfg:"client"`
}

type CloudWatchNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}/{env}/{family}/{group}-{app}"`
}

type CalculatorSettings struct {
	Cloudwatch          CloudWatchSettings `cfg:"cloudwatch"`
	DynamoDb            DynamoDbSettings   `cfg:"dynamodb"`
	Enabled             bool               `cfg:"enabled" default:"false"`
	Ecs                 EcsSettings        `cfg:"ecs"`
	LeaderElection      string             `cfg:"leader_election" default:"metric_calculator"`
	Period              time.Duration      `cfg:"period" default:"1m"`
	CloudWatchNamespace string
}

func readCalculatorSettings(config cfg.Config) (*CalculatorSettings, error) {
	var err error
	settings := &CalculatorSettings{}

	if err = config.UnmarshalKey("metric.calculator", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metric.calculator settings: %w", err)
	}

	if settings.CloudWatchNamespace, err = metric.GetCloudWatchNamespace(config); err != nil {
		return nil, fmt.Errorf("failed to get cloudwatch namespace: %w", err)
	}

	return settings, nil
}

func ReadHandlerSettings[T any](config cfg.Config, name string, settings T) (T, error) {
	key := fmt.Sprintf("metric.calculator.handlers.%s", name)
	if err := config.UnmarshalKey(key, settings); err != nil {
		return settings, fmt.Errorf("failed to unmarshal handler settings for '%s': %w", name, err)
	}

	return settings, nil
}
