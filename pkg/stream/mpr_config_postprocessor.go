package stream

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.stream.mpr_metric_leader_elections", mprConfigPostprocessor)
}

func mprConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	settings := readMessagesPerRunnerMetricSettings(config)

	if !settings.Enabled {
		return false, nil
	}

	key := conc.GetLeaderElectionConfigKey(settings.LeaderElection)
	typKey := conc.GetLeaderElectionConfigKeyType(settings.LeaderElection)

	if config.IsSet(key) {
		return true, nil
	}

	leaderElectionSettings := &conc.DdbLeaderElectionSettings{
		TableName:     fmt.Sprintf("%s-%s-%s-stream-metric-writer-leaders", config.GetString("app_project"), config.GetString("env"), config.GetString("app_family")),
		GroupId:       config.GetString("app_name"),
		LeaseDuration: time.Minute,
	}

	configOptions := []cfg.Option{
		cfg.WithConfigSetting(typKey, conc.LeaderElectionTypeDdb),
		cfg.WithConfigSetting(key, leaderElectionSettings),
	}

	if err := config.Option(configOptions...); err != nil {
		return false, fmt.Errorf("can not apply config settings for stream mpr metric: %w", err)
	}

	return true, nil
}
