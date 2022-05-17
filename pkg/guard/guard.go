package guard

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/ory/ladon"
)

const fetchLimit = 100

//go:generate mockery --name Guard
type Guard interface {
	IsAllowed(request *ladon.Request) error
	GetPolicies() (ladon.Policies, error)
	GetPoliciesBySubject(subject string) (ladon.Policies, error)
	CreatePolicy(pol ladon.Policy) error
	UpdatePolicy(pol ladon.Policy) error
	DeletePolicy(pol ladon.Policy) error
}

//go:generate mockery --name Manager
type Manager interface {
	ladon.Manager
}

type ladonGuard struct {
	warden *ladon.Ladon
}

func NewGuard(config cfg.Config, logger log.Logger) (Guard, error) {
	sqlManager, err := NewSqlManager(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create sqlManager: %w", err)
	}

	return NewGuardWithInterfaces(sqlManager), nil
}

func NewGuardWithInterfaces(manager Manager) Guard {
	warden := &ladon.Ladon{
		Manager: manager,
	}

	return &ladonGuard{
		warden: warden,
	}
}

func (g ladonGuard) IsAllowed(request *ladon.Request) error {
	return g.warden.IsAllowed(request)
}

func (g ladonGuard) GetPolicies() (ladon.Policies, error) {
	policies := make(ladon.Policies, 0)

	offset := int64(0)
	var pols ladon.Policies
	var err error

	pols, err = g.warden.Manager.GetAll(fetchLimit, offset)
	if err != nil {
		return nil, fmt.Errorf("could not get all policies: %w", err)
	}

	offset += fetchLimit
	for ; len(pols) > 0; offset += fetchLimit {
		policies = append(policies, pols...)

		pols, err = g.warden.Manager.GetAll(fetchLimit, offset)
		if err != nil {
			return nil, fmt.Errorf("could not get all policies: %w", err)
		}
	}

	return policies, nil
}

func (g ladonGuard) GetPoliciesBySubject(subject string) (ladon.Policies, error) {
	pol, err := g.warden.Manager.FindPoliciesForSubject(subject)
	if err != nil {
		return nil, fmt.Errorf("could not get policies by subject: %w", err)
	}

	return pol, nil
}

func (g ladonGuard) CreatePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Create(pol)
	if err != nil {
		return fmt.Errorf("could not create policy: %w", err)
	}

	return nil
}

func (g ladonGuard) UpdatePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Update(pol)
	if err != nil {
		return fmt.Errorf("could not update policy: %w", err)
	}

	return nil
}

func (g ladonGuard) DeletePolicy(pol ladon.Policy) error {
	err := g.warden.Manager.Delete(pol.GetID())
	if err != nil {
		return fmt.Errorf("could not delete policy: %w", err)
	}

	return nil
}
