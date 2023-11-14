package guard

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
)

const fetchLimit = 100

//go:generate mockery --name Guard
type Guard interface {
	CreatePolicy(ctx context.Context, pol ladon.Policy) error
	DeletePolicy(ctx context.Context, pol ladon.Policy) error
	GetPolicy(ctx context.Context, id string) (ladon.Policy, error)
	GetPolicies(ctx context.Context) (ladon.Policies, error)
	GetPoliciesBySubject(ctx context.Context, subject string) (ladon.Policies, error)
	IsAllowed(ctx context.Context, request *ladon.Request) error
	UpdatePolicy(ctx context.Context, pol ladon.Policy) error
}

//go:generate mockery --name Manager
type Manager interface {
	ladon.Manager
}

type LadonGuard struct {
	warden *ladon.Ladon
}

func NewGuard(config cfg.Config, logger log.Logger) (*LadonGuard, error) {
	sqlManager, err := NewSqlManager(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create sqlManager: %w", err)
	}

	auditLogger := NewAuditLogger(config, logger)

	return NewGuardWithInterfaces(sqlManager, auditLogger), nil
}

func NewGuardWithInterfaces(manager Manager, logger AuditLogger) *LadonGuard {
	warden := &ladon.Ladon{
		Manager:     manager,
		AuditLogger: logger,
	}

	return &LadonGuard{
		warden: warden,
	}
}

func (g LadonGuard) IsAllowed(ctx context.Context, request *ladon.Request) error {
	return g.warden.IsAllowed(ctx, request)
}

func (g LadonGuard) GetPolicies(ctx context.Context) (ladon.Policies, error) {
	policies := make(ladon.Policies, 0)

	offset := int64(0)
	var pols ladon.Policies
	var err error

	pols, err = g.warden.Manager.GetAll(ctx, fetchLimit, offset)
	if err != nil {
		return nil, fmt.Errorf("could not get all policies: %w", err)
	}

	offset += fetchLimit
	for ; len(pols) > 0; offset += fetchLimit {
		policies = append(policies, pols...)

		pols, err = g.warden.Manager.GetAll(ctx, fetchLimit, offset)
		if err != nil {
			return nil, fmt.Errorf("could not get all policies: %w", err)
		}
	}

	return policies, nil
}

func (g LadonGuard) GetPoliciesBySubject(ctx context.Context, subject string) (ladon.Policies, error) {
	pol, err := g.warden.Manager.FindPoliciesForSubject(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("could not get policies by subject: %w", err)
	}

	return pol, nil
}

func (g LadonGuard) CreatePolicy(ctx context.Context, pol ladon.Policy) error {
	err := g.warden.Manager.Create(ctx, pol)
	if err != nil {
		return fmt.Errorf("could not create policy: %w", err)
	}

	return nil
}

func (g LadonGuard) UpdatePolicy(ctx context.Context, pol ladon.Policy) error {
	err := g.warden.Manager.Update(ctx, pol)
	if err != nil {
		return fmt.Errorf("could not update policy: %w", err)
	}

	return nil
}

func (g LadonGuard) DeletePolicy(ctx context.Context, pol ladon.Policy) error {
	err := g.warden.Manager.Delete(ctx, pol.GetID())
	if err != nil {
		return fmt.Errorf("could not delete policy: %w", err)
	}

	return nil
}

func (g LadonGuard) GetPolicy(ctx context.Context, id string) (ladon.Policy, error) {
	return g.warden.Manager.Get(ctx, id)
}
