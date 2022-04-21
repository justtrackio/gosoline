package stream

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.stream.mpr_metric_leader_elections", mprConfigPostprocessor)
}

func mprConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	settings := readMessagesPerRunnerMetricSettings(config)

	if !settings.Enabled {
		return false, nil
	}

	key := ddb.GetLeaderElectionConfigKey(settings.LeaderElection)
	typKey := ddb.GetLeaderElectionConfigKeyType(settings.LeaderElection)

	if config.IsSet(key) {
		return true, nil
	}

	leaderElectionSettings := &ddb.DdbLeaderElectionSettings{
		TableName:     fmt.Sprintf("%s-%s-%s-stream-metric-writer-leaders", config.GetString("app_project"), config.GetString("env"), config.GetString("app_family")),
		GroupId:       config.GetString("app_name"),
		LeaseDuration: time.Minute,
	}

	configOptions := []cfg.Option{
		cfg.WithConfigSetting(typKey, ddb.LeaderElectionTypeDdb),
		cfg.WithConfigSetting(key, leaderElectionSettings),
	}

	if err := config.Option(configOptions...); err != nil {
		return false, fmt.Errorf("can not apply config settings for stream mpr metric: %w", err)
	}

	return true, nil
}
