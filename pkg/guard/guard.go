package guard

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
)

const fetchLimit = 100

//go:generate go run github.com/vektra/mockery/v2 --name Guard
type Guard interface {
	CreatePolicy(ctx context.Context, pol ladon.Policy) error
	DeletePolicy(ctx context.Context, pol ladon.Policy) error
	DeletePolicyIfUnchanged(ctx context.Context, before PolicySnapshot) (bool, error)
	GetPolicy(ctx context.Context, id string) (ladon.Policy, error)
	GetPolicySnapshot(ctx context.Context, id string) (*PolicySnapshot, error)
	GetPolicies(ctx context.Context) (ladon.Policies, error)
	GetPoliciesBySubject(ctx context.Context, subject string) (ladon.Policies, error)
	IsAllowed(ctx context.Context, request *ladon.Request) error
	UpdatePolicy(ctx context.Context, pol ladon.Policy) error
	UpdatePolicyIfUnchanged(ctx context.Context, before PolicySnapshot, after ladon.Policy) (bool, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name Manager
type Manager interface {
	ladon.Manager
	GetSnapshot(ctx context.Context, id string) (*PolicySnapshot, error)
	UpdateIfUnchanged(ctx context.Context, before PolicySnapshot, after ladon.Policy) (bool, error)
	DeleteIfUnchanged(ctx context.Context, before PolicySnapshot) (bool, error)
}

type LadonGuard struct {
	warden  *ladon.Ladon
	manager Manager
}

func NewGuard(ctx context.Context, config cfg.Config, logger log.Logger) (*LadonGuard, error) {
	sqlManager, err := NewSqlManager(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create sqlManager: %w", err)
	}

	auditLogger, err := NewAuditLogger(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create auditLogger: %w", err)
	}

	return NewGuardWithInterfaces(NewCachedManagerWithInterfaces(sqlManager), auditLogger), nil
}

func NewGuardWithInterfaces(manager Manager, logger AuditLogger) *LadonGuard {
	warden := &ladon.Ladon{
		Manager:     manager,
		AuditLogger: logger,
	}

	return &LadonGuard{
		warden:  warden,
		manager: manager,
	}
}

// GetPolicySnapshot reads an uncached policy and its original stored value. Missing policies return nil.
func (g LadonGuard) GetPolicySnapshot(ctx context.Context, id string) (*PolicySnapshot, error) {
	return g.manager.GetSnapshot(ctx, id)
}

// UpdatePolicyIfUnchanged updates a policy only when its stored value still matches before.
func (g LadonGuard) UpdatePolicyIfUnchanged(ctx context.Context, before PolicySnapshot, after ladon.Policy) (bool, error) {
	return g.manager.UpdateIfUnchanged(ctx, before, after)
}

// DeletePolicyIfUnchanged deletes a policy only when its stored value still matches before.
func (g LadonGuard) DeletePolicyIfUnchanged(ctx context.Context, before PolicySnapshot) (bool, error) {
	return g.manager.DeleteIfUnchanged(ctx, before)
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
