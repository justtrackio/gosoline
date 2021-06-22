package conc

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"time"
)

type DdbLeaderElectionItem struct {
	GroupId      string `json:"groupId" ddb:"key=hash"`
	MemberId     string `json:"memberId"`
	LeadingUntil int64  `json:"leadingUntil" ddb:"ttl=enabled"`
}

type DdbLeaderElectionSettings struct {
	TableName     string        `cfg:"table_name" default:"{app_project}-{env}-{app_family}-leader-elections"`
	GroupId       string        `cfg:"group_id" default:"{app_name}"`
	LeaseDuration time.Duration `cfg:"lease_duration" default:"1m"`
}

type DdbLeaderElection struct {
	logger     log.Logger
	clock      clock.Clock
	repository ddb.Repository
	settings   *DdbLeaderElectionSettings
}

func NewDdbLeaderElection(config cfg.Config, logger log.Logger, name string) (LeaderElection, error) {
	key := GetLeaderElectionConfigKey(name)
	settings := &DdbLeaderElectionSettings{}
	config.UnmarshalKey(key, settings)

	return NewDdbLeaderElectionWithSettings(config, logger, settings)
}

func NewDdbLeaderElectionWithSettings(config cfg.Config, logger log.Logger, settings *DdbLeaderElectionSettings) (LeaderElection, error) {
	namingFactory := func(_ mdl.ModelId) string {
		return settings.TableName
	}

	repository, err := ddb.NewRepository(config, logger, &ddb.Settings{
		ModelId:        mdl.ModelId{},
		NamingStrategy: namingFactory,
		Backoff: exec.BackoffSettings{
			Enabled:             true,
			Blocking:            false,
			CancelDelay:         0,
			InitialInterval:     time.Second,
			RandomizationFactor: 0.5,
			Multiplier:          1.5,
			MaxInterval:         time.Second * 10,
			MaxElapsedTime:      time.Minute,
		},
		DisableTracing: true,
		Main: ddb.MainSettings{
			Model:              DdbLeaderElectionItem{},
			ReadCapacityUnits:  3,
			WriteCapacityUnits: 3,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("can not create ddb repository: %w", err)
	}

	return NewDdbLeaderElectionWithInterfaces(logger, clock.Provider, repository, settings)
}

func NewDdbLeaderElectionWithInterfaces(logger log.Logger, clock clock.Clock, repository ddb.Repository, settings *DdbLeaderElectionSettings) (*DdbLeaderElection, error) {
	election := &DdbLeaderElection{
		logger:     logger,
		clock:      clock,
		repository: repository,
		settings:   settings,
	}

	return election, nil
}

func (e *DdbLeaderElection) IsLeader(ctx context.Context, memberId string) (bool, error) {
	now := e.clock.Now()
	leadingUntil := now.Add(e.settings.LeaseDuration).Unix()

	item := &DdbLeaderElectionItem{
		GroupId:      e.settings.GroupId,
		MemberId:     memberId,
		LeadingUntil: leadingUntil,
	}

	// leader election is successful if we're the current leader already or if the current leader election is older than x minutes
	conditionNoCurrentLeader := ddb.AttributeNotExists("memberId")
	conditionIsLeaderAlready := ddb.Eq("memberId", memberId)
	conditionIsNotLeader := ddb.And(ddb.NotEq("memberId", memberId), ddb.Lt("leadingUntil", now.Unix()))
	condition := ddb.Or(conditionNoCurrentLeader, conditionIsLeaderAlready, conditionIsNotLeader)

	qb := e.repository.PutItemBuilder().WithCondition(condition)
	res, err := e.repository.PutItem(ctx, qb, item)

	if err == nil {
		return !res.ConditionalCheckFailed, nil
	}

	if ddb.IsTableNotFoundError(err) {
		return false, NewLeaderElectionFatalError(err)
	}

	if err != nil {
		return false, NewLeaderElectionTransientError(err)
	}

	return !res.ConditionalCheckFailed, nil
}

func (e *DdbLeaderElection) Resign(ctx context.Context, memberId string) error {
	conditionCurrentLeader := ddb.Eq("memberId", memberId)

	qb := e.repository.DeleteItemBuilder().WithCondition(conditionCurrentLeader)
	res, err := e.repository.DeleteItem(ctx, qb, DdbLeaderElectionItem{
		GroupId: e.settings.GroupId,
	})

	if err != nil {
		return fmt.Errorf("can not resign as current leader: %w", err)
	}

	if res.ConditionalCheckFailed {
		e.logger.Warn("not not resign as leader as we're not the current one")
	}

	return nil
}
