package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
)

const (
	tableName = "guard_policies"
)

type SqlManager struct {
	logger   log.Logger
	dbClient db.Client
}

func NewSqlManager(ctx context.Context, config cfg.Config, logger log.Logger) (*SqlManager, error) {
	dbClient, err := db.ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create dbClient: %w", err)
	}

	return NewSqlManagerWithInterfaces(logger, dbClient), nil
}

func NewSqlManagerWithInterfaces(logger log.Logger, dbClient db.Client) *SqlManager {
	return &SqlManager{
		logger:   logger,
		dbClient: dbClient,
	}
}

func (m SqlManager) Create(ctx context.Context, pol ladon.Policy) error {
	var err error
	var policy []byte

	if policy, err = buildPolicy(pol); err != nil {
		return fmt.Errorf("can not marshal the policy: %w", err)
	}

	ins := squirrel.Insert(tableName).Options("IGNORE").SetMap(squirrel.Eq{
		"id":         pol.GetID(),
		"policy":     string(policy),
		"updated_at": time.Now().Format(db.FormatDateTime),
		"created_at": time.Now().Format(db.FormatDateTime),
	})

	sql, args, err := ins.ToSql()
	if err != nil {
		return err
	}

	_, err = m.dbClient.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func (m SqlManager) Update(ctx context.Context, pol ladon.Policy) error {
	var err error
	var policy []byte

	if policy, err = buildPolicy(pol); err != nil {
		return fmt.Errorf("can not marshal the policy: %w", err)
	}

	up := squirrel.Update(tableName).Where("id = ?", pol.GetID()).SetMap(squirrel.Eq{
		"policy":     string(policy),
		"updated_at": time.Now().Format(db.FormatDateTime),
	})

	sql, args, err := up.ToSql()
	if err != nil {
		return err
	}

	if _, err = m.dbClient.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("can not execute sql statement: %w", err)
	}

	return nil
}

func (m SqlManager) Get(ctx context.Context, id string) (ladon.Policy, error) {
	sel := buildSelectBuilder(squirrel.Eq{"p.id": id})

	policies, err := m.queryPolicies(ctx, sel)
	if err != nil {
		return nil, err
	}

	if len(policies) != 1 {
		return nil, fmt.Errorf("invalid amount of policies for id %s", id)
	}

	return policies[0], nil
}

func (m SqlManager) Delete(ctx context.Context, id string) error {
	del := squirrel.Delete(tableName).Where(squirrel.Eq{"id": id})
	sql, args, err := del.ToSql()
	if err != nil {
		m.logger.Error(ctx, "can not delete from %s: %w", tableName, err)

		return err
	}

	_, err = m.dbClient.Exec(ctx, sql, args...)
	if err != nil {
		m.logger.Error(ctx, "can not delete from %s: %w", tableName, err)

		return err
	}

	return nil
}

func (m SqlManager) GetAll(ctx context.Context, limit, offset int64) (ladon.Policies, error) {
	sel := buildSelectBuilder(squirrel.Eq{"1": "1"})
	sel = sel.Limit(uint64(limit))
	sel = sel.Offset(uint64(offset))

	return m.queryPolicies(ctx, sel)
}

func (m SqlManager) FindRequestCandidates(ctx context.Context, r *ladon.Request) (ladon.Policies, error) {
	return m.FindPoliciesForSubject(ctx, r.Subject)
}

func (m SqlManager) FindPoliciesForSubject(ctx context.Context, subject string) (ladon.Policies, error) {
	sel := buildSelectBuilder(squirrel.Expr("JSON_CONTAINS(p.policy, JSON_QUOTE(?), '$.subjects')", subject))

	return m.queryPolicies(ctx, sel)
}

func (m SqlManager) FindPoliciesForResource(ctx context.Context, resource string) (ladon.Policies, error) {
	sel := buildSelectBuilder(squirrel.Expr("JSON_CONTAINS(p.policy, JSON_QUOTE(?), '$.resources')", resource))

	return m.queryPolicies(ctx, sel)
}

func (m SqlManager) queryPolicies(ctx context.Context, sel squirrel.SelectBuilder) (ladon.Policies, error) {
	sql, args, err := sel.ToSql()
	if err != nil {
		return nil, fmt.Errorf("can not build sql string to query the policies: %w", err)
	}

	res, err := m.dbClient.GetResult(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("can not get result to query the policies: %w", err)
	}

	policies := make(ladon.Policies, 0)
	for _, row := range *res {
		var policy ladon.DefaultPolicy

		if err = json.Unmarshal([]byte(row["policy"]), &policy); err != nil {
			return nil, fmt.Errorf("can not unmarshal the policy: %w", err)
		}

		policy.ID = row["id"]

		policies = append(policies, &policy)
	}

	return policies, nil
}

func buildSelectBuilder(where any) squirrel.SelectBuilder {
	sel := squirrel.Select("p.id", "p.policy")
	sel = sel.From(fmt.Sprintf("%s AS p", tableName))
	sel = sel.Where(where)
	sel = sel.OrderBy("p.id")

	return sel
}

func buildPolicy(policy ladon.Policy) ([]byte, error) {
	// removes ID field in the policy
	return json.Marshal(&struct {
		Description string           `json:"description"`
		Effect      string           `json:"effect"`
		Conditions  ladon.Conditions `json:"conditions"`
		Subjects    []string         `json:"subjects"`
		Resources   []string         `json:"resources"`
		Actions     []string         `json:"actions"`
	}{
		Description: policy.GetDescription(),
		Effect:      policy.GetEffect(),
		Conditions:  policy.GetConditions(),
		Subjects:    policy.GetSubjects(),
		Resources:   policy.GetResources(),
		Actions:     policy.GetActions(),
	})
}
