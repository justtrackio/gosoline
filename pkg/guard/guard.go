package guard

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/ory/ladon"
)

//go:generate mockery -name Guard
type Guard interface {
	IsAllowed(request *ladon.Request) error
	GetPolicesBySubject(subject string) (ladon.Policies, error)
	CreatePolicy(pol ladon.Policy) error
	UpdatePolicy(pol ladon.Policy) error
	DeletePolicy(pol ladon.Policy) error
}

type LadonGuard struct {
	logger mon.Logger
	warden *ladon.Ladon
}

func NewGuard(config cfg.Config, logger mon.Logger) *LadonGuard {
	m := NewSqlManager(config, logger)

	warden := &ladon.Ladon{
		Manager: m,
	}

	return &LadonGuard{
		logger: logger,
		warden: warden,
	}
}

func (g LadonGuard) IsAllowed(request *ladon.Request) error {
	return g.warden.IsAllowed(request)
}

func (g LadonGuard) GetPolicesBySubject(subject string) (ladon.Policies, error) {
	pol, err := g.warden.Manager.FindPoliciesForSubject(subject)

	if err != nil {
		g.logger.Error(err, "could not get policies by subject")
	}

	return pol, err
}

func (g LadonGuard) CreatePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Create(pol)

	if err != nil {
		g.logger.Error(err, "could not create policy")
	}

	return err
}

func (g LadonGuard) UpdatePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Update(pol)

	if err != nil {
		g.logger.Error(err, "could not update policy")
	}

	return err
}

func (g LadonGuard) DeletePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Delete(pol.GetID())

	if err != nil {
		g.logger.Error(err, "could not delete policy")
	}

	return err
}
