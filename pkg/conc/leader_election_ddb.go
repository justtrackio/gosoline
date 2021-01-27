package conc

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

type DdbLeaderElectionItem struct {
	GroupId      string `json:"groupId" ddb:"key=hash"`
	MemberId     string `json:"memberId"`
	LeadingUntil int64  `json:"leadingUntil" ddb:"ttl=enabled"`
}

type DdbLeaderElectionSettings struct {
	TableName     string
	GroupId       string
	LeaseDuration time.Duration
}

type DdbLeaderElection struct {
	repository ddb.Repository
	settings   *DdbLeaderElectionSettings
}

func NewDdbLeaderElection(config cfg.Config, logger mon.Logger, settings *DdbLeaderElectionSettings) (*DdbLeaderElection, error) {
	namingFactory := func(_ mdl.ModelId) string {
		return settings.TableName
	}

	repository := ddb.NewRepository(config, logger, &ddb.Settings{
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
		Main: ddb.MainSettings{
			Model:              DdbLeaderElectionItem{},
			ReadCapacityUnits:  3,
			WriteCapacityUnits: 3,
		},
	})

	return NewDdbLeaderElectionWithInterfaces(repository, settings)
}

func NewDdbLeaderElectionWithInterfaces(repository ddb.Repository, settings *DdbLeaderElectionSettings) (*DdbLeaderElection, error) {
	election := &DdbLeaderElection{
		repository: repository,
		settings:   settings,
	}

	return election, nil
}

func (e *DdbLeaderElection) IsLeader(ctx context.Context, memberId string) (bool, error) {
	now := time.Now()
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

	if err != nil {
		return false, fmt.Errorf("can not determine current leader: %w", err)
	}

	return !res.ConditionalCheckFailed, nil
}
