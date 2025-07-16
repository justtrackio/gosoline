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

func readKinsumerAutoscaleSettings(config cfg.Config) (KinsumerAutoscaleModuleSettings, error) {
	settings := &KinsumerAutoscaleModuleSettings{}
	if err := config.UnmarshalKey("kinsumer.autoscale", settings); err != nil {
		return KinsumerAutoscaleModuleSettings{}, fmt.Errorf("failed to unmarshal kinsumer autoscale settings in readKinsumerAutoscaleSettings: %w", err)
	}

	return *settings, nil
}

func readKinsumerInputSettings(config cfg.Config, kinsumerInputName string) (KinesisInputConfiguration, error) {
	kinsumerInputKey := ConfigurableInputKey(kinsumerInputName)

	settings := &KinesisInputConfiguration{}
	if err := config.UnmarshalKey(kinsumerInputKey, settings); err != nil {
		return KinesisInputConfiguration{}, fmt.Errorf("failed to unmarshal kinsumer input settings for key %q in readKinsumerInputSettings: %w", kinsumerInputKey, err)
	}
	settings.Name = kinsumerInputName

	return *settings, nil
}

func kinsumerAutoscaleConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	settings, err := readKinsumerAutoscaleSettings(config)
	if err != nil {
		return false, fmt.Errorf("failed to read kinsumer autoscale settings in kinsumerAutoscaleConfigPostprocessor: %w", err)
	}

	if !settings.Enabled {
		return false, nil
	}

	electionKey := ddb.GetLeaderElectionConfigKey(settings.LeaderElection)
	electionKeyType := ddb.GetLeaderElectionConfigKeyType(settings.LeaderElection)

	if config.IsSet(electionKey) {
		return true, nil
	}

	appGroup, err := config.GetString("app_group")
	if err != nil {
		return false, fmt.Errorf("could not get app_group: %w", err)
	}

	appName, err := config.GetString("app_name")
	if err != nil {
		return false, fmt.Errorf("could not get app_name: %w", err)
	}

	leaderElectionSettings := &ddb.DdbLeaderElectionSettings{
		Naming: ddb.TableNamingSettings{
			Pattern: settings.DynamoDb.Naming.Pattern,
		},
		GroupId:       fmt.Sprintf("%s-%s", appGroup, appName),
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
