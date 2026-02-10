package calculator

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.metrics.calculator_leader_elections", calculatorConfigPostprocessor)
}

func calculatorConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	var err error
	var settings *CalculatorSettings
	var identity cfg.AppIdentity
	var namespace string

	if settings, err = readCalculatorSettings(config); err != nil {
		return false, fmt.Errorf("can not read calculator settings: %w", err)
	}

	if !settings.Enabled {
		return false, nil
	}

	electionKey := ddb.GetLeaderElectionConfigKey(settings.LeaderElection)
	electionKeyType := ddb.GetLeaderElectionConfigKeyType(settings.LeaderElection)

	if config.IsSet(electionKey) {
		return true, nil
	}

	if identity, err = cfg.GetAppIdentity(config); err != nil {
		return false, fmt.Errorf("could not get app identity: %w", err)
	}

	if namespace, err = identity.FormatNamespace("-"); err != nil {
		return false, fmt.Errorf("could not format app namespace: %w", err)
	}

	leaderElectionSettings := &ddb.DdbLeaderElectionSettings{
		Naming: ddb.TableNamingSettings{
			TablePattern:   settings.DynamoDb.Naming.TablePattern,
			TableDelimiter: settings.DynamoDb.Naming.TableDelimiter,
		},
		GroupId:       fmt.Sprintf("%s-%s", namespace, identity.Name),
		LeaseDuration: time.Minute,
	}

	configOptions := []cfg.Option{
		cfg.WithConfigSetting(electionKeyType, ddb.LeaderElectionTypeDdb),
		cfg.WithConfigSetting(electionKey, leaderElectionSettings),
	}

	if err := config.Option(configOptions...); err != nil {
		return false, fmt.Errorf("can not apply config settings for metrics calculator leader election: %w", err)
	}

	return true, nil
}
