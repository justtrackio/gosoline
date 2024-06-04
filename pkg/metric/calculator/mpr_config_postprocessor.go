package calculator

import (
	"fmt"
	"strings"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.metrics.calculator_leader_elections", calculatorConfigPostprocessor)
}

func calculatorConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	settings := readCalculatorSettings(config)

	if !settings.Enabled {
		return false, nil
	}

	electionKey := ddb.GetLeaderElectionConfigKey(settings.LeaderElection)
	electionKeyType := ddb.GetLeaderElectionConfigKeyType(settings.LeaderElection)

	if config.IsSet(electionKey) {
		return true, nil
	}

	pattern := settings.DynamoDb.Naming.Pattern

	values := map[string]string{
		"modelId": "metric-calculator-leaders",
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		pattern = strings.ReplaceAll(pattern, templ, val)
	}

	leaderElectionSettings := &ddb.DdbLeaderElectionSettings{
		Naming: ddb.TableNamingSettings{
			Pattern: pattern,
		},
		GroupId:       fmt.Sprintf("%s-%s", config.GetString("app_group"), config.GetString("app_name")),
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
