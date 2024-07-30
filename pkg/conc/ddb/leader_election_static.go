package ddb

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type StaticLeaderElection struct {
	result bool
}

func NewStaticLeaderElection(_ context.Context, config cfg.Config, _ log.Logger, name string) (LeaderElection, error) {
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

func (e StaticLeaderElection) IsLeader(context.Context, string) (bool, error) {
	return e.result, nil
}

func (e StaticLeaderElection) Resign(context.Context, string) error {
	return nil
}
