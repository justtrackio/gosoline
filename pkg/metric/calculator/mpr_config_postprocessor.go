package calculator

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/metric"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.metrics.calculator_leader_elections", calculatorConfigPostprocessor)
}

func calculatorConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	var err error
	var metricSettings *metric.Settings
	var calculatorSettings *CalculatorSettings
	var identity cfg.Identity
	var namespace string

	if metricSettings, err = metric.GetMetricSettings(config); err != nil {
		return false, fmt.Errorf("can not read metric settings: %w", err)
	}

	if !metricSettings.Enabled {
		return false, nil
	}

	if calculatorSettings, err = readCalculatorSettings(config); err != nil {
		return false, fmt.Errorf("can not read calculator settings: %w", err)
	}

	if !calculatorSettings.Enabled {
		return false, nil
	}

	electionKey := ddb.GetLeaderElectionConfigKey(calculatorSettings.LeaderElection)
	electionKeyType := ddb.GetLeaderElectionConfigKeyType(calculatorSettings.LeaderElection)

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
			TablePattern:   calculatorSettings.DynamoDb.Naming.TablePattern,
			TableDelimiter: calculatorSettings.DynamoDb.Naming.TableDelimiter,
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
