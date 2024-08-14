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

//go:generate mockery --name=LeaderElection
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

	typ := config.GetString(key)

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
