package conc

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
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

type LeaderElectionFactory func(config cfg.Config, logger log.Logger, name string) (LeaderElection, error)

var leaderElectionFactories = map[string]LeaderElectionFactory{
	LeaderElectionTypeDdb:    NewDdbLeaderElection,
	LeaderElectionTypeStatic: NewStaticLeaderElection,
}

func NewLeaderElection(config cfg.Config, logger log.Logger, name string) (LeaderElection, error) {
	key := GetLeaderElectionConfigKeyType(name)

	if !config.IsSet(key) {
		return nil, fmt.Errorf("no leader election with name %s configured", name)
	}

	typ := config.GetString(key)

	if _, ok := leaderElectionFactories[typ]; !ok {
		return nil, fmt.Errorf("leader election with name %s has an unknown type %s", name, typ)
	}

	return leaderElectionFactories[typ](config, logger, name)
}

func GetLeaderElectionConfigKeyType(name string) string {
	return fmt.Sprintf("conc.leader_election.%s.type", name)
}

func GetLeaderElectionConfigKey(name string) string {
	return fmt.Sprintf("conc.leader_election.%s", name)
}
