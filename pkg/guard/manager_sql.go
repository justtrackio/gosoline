package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
)

const (
	tablePolicies  = "guard_policies"
	tableSubjects  = "guard_subjects"
	tableResources = "guard_resources"
	tableActions   = "guard_actions"
)

type SqlManager struct {
	logger   log.Logger
	dbClient db.Client
}

func NewSqlManager(config cfg.Config, logger log.Logger) (*SqlManager, error) {
	dbClient, err := db.NewClient(config, logger, "default")
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
	var conditions []byte

	if conditions, err = json.Marshal(pol.GetConditions()); err != nil {
		return fmt.Errorf("can not marshal the conditions: %w", err)
	}

	ins := squirrel.Insert(tablePolicies).Options("IGNORE").SetMap(squirrel.Eq{
		"id":          pol.GetID(),
		"description": pol.GetDescription(),
		"effect":      pol.GetEffect(),
		"conditions":  string(conditions),
		"updated_at":  time.Now().Format(db.FormatDateTime),
		"created_at":  time.Now().Format(db.FormatDateTime),
	})

	sql, args, err := ins.ToSql()
	if err != nil {
		return err
	}

	_, err = m.dbClient.Exec(ctx, sql, args...)

	if err != nil {
		return err
	}

	if err = m.createAssociations(ctx, pol, tableSubjects, pol.GetSubjects()); err != nil {
		return err
	}

	if err = m.createAssociations(ctx, pol, tableResources, pol.GetResources()); err != nil {
		return err
	}

	if err = m.createAssociations(ctx, pol, tableActions, pol.GetActions()); err != nil {
		return err
	}

	return nil
}

func (m SqlManager) createAssociations(ctx context.Context, pol ladon.Policy, table string, values []string) error {
	ins := squirrel.Insert(table).Options("IGNORE").Columns("id", "name")
	for _, s := range values {
		ins = ins.Values(pol.GetID(), s)
	}

	sql, args, err := ins.ToSql()
	if err != nil {
		return err
	}

	if _, err = m.dbClient.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("can not execute sql statement: %w", err)
	}

	return nil
}

func (m SqlManager) Update(ctx context.Context, pol ladon.Policy) error {
	var err error
	var conditions []byte

	if conditions, err = json.Marshal(pol.GetConditions()); err != nil {
		return fmt.Errorf("can not marshal the conditions: %w", err)
	}

	up := squirrel.Update(tablePolicies).Where("id = ?", pol.GetID()).SetMap(squirrel.Eq{
		"description": pol.GetDescription(),
		"effect":      pol.GetEffect(),
		"conditions":  string(conditions),
		"updated_at":  time.Now().Format(db.FormatDateTime),
	})

	sql, args, err := up.ToSql()
	if err != nil {
		return err
	}

	if _, err = m.dbClient.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("can not execute sql statement: %w", err)
	}

	if err = m.updateAssociations(ctx, pol, tableResources, pol.GetResources()); err != nil {
		return err
	}

	if err = m.updateAssociations(ctx, pol, tableActions, pol.GetActions()); err != nil {
		return err
	}

	return nil
}

func (m SqlManager) updateAssociations(ctx context.Context, pol ladon.Policy, table string, values []string) error {
	err := m.deleteByIdAndTable(ctx, pol.GetID(), table)
	if err != nil {
		return err
	}

	if err = m.createAssociations(ctx, pol, table, values); err != nil {
		return err
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
	tables := []string{tablePolicies, tableSubjects, tableResources, tableActions}

	for _, table := range tables {
		err := m.deleteByIdAndTable(ctx, id, table)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m SqlManager) deleteByIdAndTable(ctx context.Context, id string, table string) error {
	del := squirrel.Delete(table).Where(squirrel.Eq{"id": id})
	sql, args, err := del.ToSql()
	if err != nil {
		m.logger.Error("can not delete from %s: %w", table, err)
		return err
	}

	_, err = m.dbClient.Exec(ctx, sql, args...)

	if err != nil {
		m.logger.Error("can not delete from %s: %w", table, err)
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
	sel := buildSelectBuilder(squirrel.Eq{"s.name": subject})

	return m.queryPolicies(ctx, sel)
}

func (m SqlManager) FindPoliciesForResource(ctx context.Context, resource string) (ladon.Policies, error) {
	sel := buildSelectBuilder(squirrel.Eq{"r.name": resource})

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

	policies := map[string]*ladon.DefaultPolicy{}
	for _, row := range *res {
		if _, ok := policies[row["id"]]; !ok {
			conditions := make(ladon.Conditions)

			if err = json.Unmarshal([]byte(row["conditions"]), &conditions); err != nil {
				return nil, fmt.Errorf("can not unmarshal the conditions of policy %s: %w", row["id"], err)
			}

			policies[row["id"]] = &ladon.DefaultPolicy{
				ID:          row["id"],
				Description: row["description"],
				Effect:      row["effect"],
				Subjects:    make([]string, 0),
				Resources:   make([]string, 0),
				Actions:     make([]string, 0),
				Conditions:  conditions,
			}
		}

		pol := policies[row["id"]]
		pol.Subjects = append(pol.Subjects, row["subject"])
		pol.Resources = append(pol.Resources, row["resource"])
		pol.Actions = append(pol.Actions, row["action"])
	}

	unique := make(ladon.Policies, 0)
	for _, pol := range policies {
		pol.Subjects = funk.Uniq(pol.Subjects)
		pol.Resources = funk.Uniq(pol.Resources)
		pol.Actions = funk.Uniq(pol.Actions)

		unique = append(unique, pol)
	}

	return unique, nil
}

func buildSelectBuilder(where squirrel.Eq) squirrel.SelectBuilder {
	sel := squirrel.Select("p.id", "p.description", "p.effect", "p.conditions", "s.name AS subject", "r.name AS resource", "a.name AS action")
	sel = sel.From(fmt.Sprintf("%s AS p", tablePolicies))
	sel = sel.Join(fmt.Sprintf("%s AS s ON s.id = p.id", tableSubjects))
	sel = sel.Join(fmt.Sprintf("%s AS r ON r.id = p.id", tableResources))
	sel = sel.Join(fmt.Sprintf("%s AS a ON a.id = p.id", tableActions))
	sel = sel.Where(where)
	sel = sel.OrderBy("p.id")

	return sel
}
