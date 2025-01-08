package stream

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.kinsumer.autoscale.leader_elections", kinsumerAutoscaleConfigPostprocessor)
}

type KinsumerAutoscaleModuleEcsSettings struct {
	Client  string `cfg:"client" default:"default"`
	Cluster string `cfg:"cluster" default:"{env}"`
	Service string `cfg:"service" default:"{app_group}-{app_name}"`
}

type KinsumerAutoscaleModuleDynamoDbSettings struct {
	Naming KinsumerAutoscaleModuleDynamoDbNamingSettings `cfg:"naming"`
}

type KinsumerAutoscaleModuleDynamoDbNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{env}-kinsumer-autoscale-leaders"`
}

type KinsumerAutoscaleModuleSettings struct {
	Ecs            KinsumerAutoscaleModuleEcsSettings      `cfg:"ecs"`
	Enabled        bool                                    `cfg:"enabled" default:"true"`
	DynamoDb       KinsumerAutoscaleModuleDynamoDbSettings `cfg:"dynamodb"`
	LeaderElection string                                  `cfg:"leader_election" default:"kinsumer-autoscale"`
	Orchestrator   string                                  `cfg:"orchestrator" default:"ecs"`
	Period         time.Duration                           `cfg:"period" default:"1m"`
}

func readKinsumerAutoscaleSettings(config cfg.Config) KinsumerAutoscaleModuleSettings {
	settings := &KinsumerAutoscaleModuleSettings{}
	config.UnmarshalKey("kinsumer.autoscale", settings)

	return *settings
}

func readKinsumerInputSettings(config cfg.Config, kinsumerInputName string) KinesisInputConfiguration {
	kinsumerInputKey := ConfigurableInputKey(kinsumerInputName)

	settings := &KinesisInputConfiguration{}
	config.UnmarshalKey(kinsumerInputKey, settings)
	settings.Name = kinsumerInputName

	return *settings
}

func kinsumerAutoscaleConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	settings := readKinsumerAutoscaleSettings(config)

	if !settings.Enabled {
		return false, nil
	}

	electionKey := ddb.GetLeaderElectionConfigKey(settings.LeaderElection)
	electionKeyType := ddb.GetLeaderElectionConfigKeyType(settings.LeaderElection)

	if config.IsSet(electionKey) {
		return true, nil
	}

	leaderElectionSettings := &ddb.DdbLeaderElectionSettings{
		Naming: ddb.TableNamingSettings{
			Pattern: settings.DynamoDb.Naming.Pattern,
		},
		GroupId:       fmt.Sprintf("%s-%s", config.GetString("app_group"), config.GetString("app_name")),
		LeaseDuration: time.Minute,
	}

	configOptions := []cfg.Option{
		cfg.WithConfigSetting(electionKeyType, ddb.LeaderElectionTypeDdb),
		cfg.WithConfigSetting(electionKey, leaderElectionSettings),
	}

	if err := config.Option(configOptions...); err != nil {
		return false, fmt.Errorf("can not apply config settings for kinsumer autoscale leader election: %w", err)
	}

	return true, nil
}
