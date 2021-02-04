package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/conc"
	"time"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.stream.mpr_metric_leader_elections", mprConfigPostprocessor)
}

func mprConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	mprSettings := readAllMessagesPerRunnerMetricSettings(config)

	if len(mprSettings) == 0 {
		return false, nil
	}

	for name, settings := range mprSettings {
		key := conc.GetLeaderElectionConfigKey(settings.LeaderElection)
		typKey := conc.GetLeaderElectionConfigKeyType(settings.LeaderElection)

		if config.IsSet(key) {
			continue
		}

		leaderElectionSettings := &conc.DdbLeaderElectionSettings{
			TableName:     fmt.Sprintf("%s-%s-%s-stream-metric-writer-leaders", config.GetString("app_project"), config.GetString("env"), config.GetString("app_family")),
			GroupId:       fmt.Sprintf("%s-%s", config.GetString("app_name"), name),
			LeaseDuration: time.Minute,
		}

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(typKey, conc.LeaderElectionTypeDdb),
			cfg.WithConfigSetting(key, leaderElectionSettings),
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply config settings for stream mpr metrics %s: %w", name, err)
		}
	}

	return true, nil
}
