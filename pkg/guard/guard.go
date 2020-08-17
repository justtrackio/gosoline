package guard

import (
	"fmt"
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
	warden *ladon.Ladon
}

func NewGuard(config cfg.Config, logger mon.Logger) *LadonGuard {
	m := NewSqlManager(config, logger)

	warden := &ladon.Ladon{
		Manager: m,
	}

	return &LadonGuard{
		warden: warden,
	}
}

func (g LadonGuard) IsAllowed(request *ladon.Request) error {
	return g.warden.IsAllowed(request)
}

func (g LadonGuard) GetPolicesBySubject(subject string) (ladon.Policies, error) {
	pol, err := g.warden.Manager.FindPoliciesForSubject(subject)

	if err != nil {
		return nil, fmt.Errorf("could not get policies by subject: %w", err)
	}

	return pol, nil
}

func (g LadonGuard) CreatePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Create(pol)

	if err != nil {
		return fmt.Errorf("could not create policy: %w", err)
	}

	return nil
}

func (g LadonGuard) UpdatePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Update(pol)

	if err != nil {
		return fmt.Errorf("could not update policy: %w", err)
	}

	return nil
}

func (g LadonGuard) DeletePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Delete(pol.GetID())

	if err != nil {
		return fmt.Errorf("could not delete policy: %w", err)
	}

	return nil
}
