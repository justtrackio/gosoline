package conc

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
)

type StaticLeaderElectionSettings struct {
	Result bool `cfg:"result"`
}

type StaticLeaderElection struct {
	result bool
}

func NewStaticLeaderElection(config cfg.Config, logger log.Logger, name string) (LeaderElection, error) {
	key := GetLeaderElectionConfigKey(name)
	settings := &StaticLeaderElectionSettings{}
	config.UnmarshalKey(key, settings)

	return NewStaticLeaderElectionWithSettings(settings)
}

func NewStaticLeaderElectionWithSettings(settings *StaticLeaderElectionSettings) (*StaticLeaderElection, error) {
	return &StaticLeaderElection{
		result: settings.Result,
	}, nil
}

func (e StaticLeaderElection) IsLeader(ctx context.Context, memberId string) (bool, error) {
	return e.result, nil
}

func (e StaticLeaderElection) Resign(ctx context.Context, memberId string) error {
	return nil
}
