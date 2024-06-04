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

func readCalculatorSettings(config cfg.Config) *CalculatorSettings {
	settings := &CalculatorSettings{}
	config.UnmarshalKey("metric.calculator", settings)

	settings.CloudWatchNamespace = metric.GetCloudWatchNamespace(config)

	return settings
}

func ReadHandlerSettings[T any](config cfg.Config, name string, settings T) T {
	key := fmt.Sprintf("metric.calculator.handlers.%s", name)
	config.UnmarshalKey(key, settings)

	return settings
}
