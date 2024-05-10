package ddb

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type StaticLeaderElectionSettings struct {
	Result bool `cfg:"result"`
}

type StaticLeaderElection struct {
	result bool
}

func NewStaticLeaderElection(_ context.Context, config cfg.Config, _ log.Logger, name string) (LeaderElection, error) {
	key := GetLeaderElectionConfigKey(name)
	settings := &StaticLeaderElectionSettings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal static leader election settings: %w", err)
	}

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
