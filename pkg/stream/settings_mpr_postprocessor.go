package stream

import (
	"fmt"
	"strings"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.stream.mpr_metric_leader_elections", mprConfigPostprocessor)
}

func mprConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	if !messagesPerRunnerIsEnabled(config) {
		return false, nil
	}

	settings := readMessagesPerRunnerMetricSettings(config)

	electionKey := ddb.GetLeaderElectionConfigKey(settings.LeaderElection)
	electionKeyType := ddb.GetLeaderElectionConfigKeyType(settings.LeaderElection)

	if config.IsSet(electionKey) {
		return true, nil
	}

	pattern := settings.DynamoDb.Naming.Pattern

	values := map[string]string{
		"modelId": "stream-metric-writer-leaders",
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
		return false, fmt.Errorf("can not apply config settings for stream mpr metric: %w", err)
	}

	return true, nil
}
