package conc

import "context"

//go:generate mockery --name=LeaderElection
type LeaderElection interface {
	IsLeader(ctx context.Context, memberId string) (bool, error)
}

type StaticLeaderElection struct {
	result bool
}

func NewStaticLeaderElection(result bool) *StaticLeaderElection {
	return &StaticLeaderElection{
		result: result,
	}
}

func (e StaticLeaderElection) IsLeader(ctx context.Context) (bool, error) {
	return e.result, nil
}
