package ddb

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	LeaderElectionTypeDdb    = "ddb"
	LeaderElectionTypeStatic = "static"
)

//go:generate go run github.com/vektra/mockery/v2 --name=LeaderElection
type LeaderElection interface {
	IsLeader(ctx context.Context, memberId string) (bool, error)
	Resign(ctx context.Context, memberId string) error
}

type LeaderElectionFactory func(ctx context.Context, config cfg.Config, logger log.Logger, name string) (LeaderElection, error)

var leaderElectionFactories = map[string]LeaderElectionFactory{
	LeaderElectionTypeDdb:    NewDdbLeaderElection,
	LeaderElectionTypeStatic: NewStaticLeaderElection,
}

func NewLeaderElection(ctx context.Context, config cfg.Config, logger log.Logger, name string) (LeaderElection, error) {
	key := GetLeaderElectionConfigKeyType(name)

	if !config.IsSet(key) {
		return nil, fmt.Errorf("no leader election with name %s configured", name)
	}

	typ, err := config.GetString(key)
	if err != nil {
		return nil, fmt.Errorf("could not get leader election type: %w", err)
	}

	if _, ok := leaderElectionFactories[typ]; !ok {
		return nil, fmt.Errorf("leader election with name %s has an unknown type %s", name, typ)
	}

	return leaderElectionFactories[typ](ctx, config, logger, name)
}

func GetLeaderElectionConfigKeyType(name string) string {
	return fmt.Sprintf("%s.type", GetLeaderElectionConfigKey(name))
}

func GetLeaderElectionConfigKey(name string) string {
	return fmt.Sprintf("conc.leader_election.%s", name)
}
