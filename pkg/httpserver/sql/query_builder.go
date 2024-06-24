package sql

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/db-repo"
)

const (
	OpEq          = "="
	OpIs          = "is"
	OpIsNot       = "is not"
	OpLike        = "~"
	OPMemberOf    = "MEMBER OF"
	OpNeq         = "!="
	OPNotMemberOf = "NOT MEMBER OF"
)

type Order struct {
	Direction string `json:"direction"`
	Field     string `json:"field"`
}

type Page struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type Filter struct {
	Groups  []Filter      `json:"groups"`
	Matches []FilterMatch `json:"matches"`
	Bool    string        `json:"bool"`
}

type FilterMatch struct {
	Values    []interface{} `json:"values"`
	Dimension string        `json:"dimension"`
	Operator  string        `json:"operator"`
}

type Input struct {
	Filter  Filter   `json:"filter"`
	GroupBy []string `json:"groupBy"`
	Order   []Order  `json:"order"`
	Page    *Page    `json:"page"`
}

func NewInput() *Input {
	return &Input{
		Order: make([]Order, 0),
	}
}

type OrmQueryBuilder struct {
	*baseQueryBuilder
}

func NewOrmQueryBuilder(metadata db_repo.Metadata) *OrmQueryBuilder {
	return &OrmQueryBuilder{
		baseQueryBuilder: newBaseQueryBuilder(metadata),
	}
}

func (qb OrmQueryBuilder) Build(inp *Input) (*db_repo.QueryBuilder, error) {
	ormQb := db_repo.NewQueryBuilder()
	err := qb.build(inp, ormQb)

	return ormQb, err
}

type RawQueryBuilder struct {
	*baseQueryBuilder
}

func NewRawQueryBuilder(metadata db_repo.Metadata) *RawQueryBuilder {
	return &RawQueryBuilder{
		baseQueryBuilder: newBaseQueryBuilder(metadata),
	}
}

func (qb RawQueryBuilder) Build(inp *Input) (*db.RawQueryBuilder, error) {
	rawQb := db.NewQueryBuilder()
	err := qb.build(inp, rawQb)

	return rawQb, err
}

type baseQueryBuilder struct {
	mapping  db_repo.FieldMappings
	metadata db_repo.Metadata
}

func newBaseQueryBuilder(metadata db_repo.Metadata) *baseQueryBuilder {
	return &baseQueryBuilder{
		metadata: metadata,
		mapping:  metadata.Mappings,
	}
}

func (qb baseQueryBuilder) build(inp *Input, dbQb db.QueryBuilder) error {
	joins, err := qb.getJoins(inp)
	if err != nil {
		return err
	}

	query, args, err := qb.buildFilter(inp.Filter)
	if err != nil {
		return err
	}

	if qb.metadata.TableName == "" {
		return fmt.Errorf("no table name defined")
	}

	if qb.metadata.PrimaryKey == "" {
		return fmt.Errorf("no primary key defined")
	}

	groupBy := []string{qb.metadata.PrimaryKey}

	for _, g := range inp.GroupBy {
		if _, ok := qb.mapping[g]; !ok {
			return fmt.Errorf("no list mapping found for group by field %s", g)
		}

		field := qb.mapping[g]
		groupBy = append(groupBy, field.ColumnNames()...)
		joins = append(joins, field.Joins()...)
	}

	dbQb.Table(qb.metadata.TableName)
	dbQb.Joins(joins)
	dbQb.Where(query, args...)
	dbQb.GroupBy(groupBy...)

	for _, o := range inp.Order {
		if _, ok := qb.mapping[o.Field]; !ok {
			return fmt.Errorf("no list mapping found for order field %s", o.Field)
		}

		columns := strings.Join(qb.mapping[o.Field].ColumnNames(), ", ")
		dbQb.OrderBy(columns, o.Direction)
	}

	if inp.Page != nil {
		dbQb.Page(inp.Page.Offset, inp.Page.Limit)
	}

	return nil
}

func (qb baseQueryBuilder) getJoins(inp *Input) ([]string, error) {
	joins := make([]string, 0)

	err := qb.getJoinsFromOrder(&joins, inp.Order)
	if err != nil {
		return joins, err
	}

	err = qb.getJoinsFromFilter(&joins, inp.Filter)
	if err != nil {
		return joins, err
	}

	return joins, nil
}

func (qb baseQueryBuilder) getJoinsFromOrder(joins *[]string, order []Order) error {
	for _, o := range order {
		if _, ok := qb.mapping[o.Field]; !ok {
			return fmt.Errorf("no list mapping found for dimension %s", o.Field)
		}

		if len(qb.mapping[o.Field].Joins()) == 0 {
			continue
		}

		*joins = append(*joins, qb.mapping[o.Field].Joins()...)
	}

	return nil
}

func (qb baseQueryBuilder) getJoinsFromFilter(joins *[]string, filter Filter) error {
	for _, m := range filter.Matches {
		if _, ok := qb.mapping[m.Dimension]; !ok {
			return fmt.Errorf("no list mapping found for dimension %s", m.Dimension)
		}

		if len(qb.mapping[m.Dimension].Joins()) == 0 {
			continue
		}

		*joins = append(*joins, qb.mapping[m.Dimension].Joins()...)
	}

	for _, g := range filter.Groups {
		err := qb.getJoinsFromFilter(joins, g)
		if err != nil {
			return err
		}
	}

	return nil
}

func (qb baseQueryBuilder) buildFilter(filter Filter) (string, []interface{}, error) {
	where := ""
	args := make([]interface{}, 0)

	if len(filter.Matches) == 0 && len(filter.Groups) == 0 {
		return where, args, nil
	}

	matchesWhere, matchesArgs, err := qb.buildFilterMatches(filter.Matches)
	args = append(args, matchesArgs...)

	if err != nil {
		return where, args, err
	}

	for _, g := range filter.Groups {
		groupWhere, groupArgs, err := qb.buildFilter(g)
		if err != nil {
			return where, args, err
		}

		matchesWhere = append(matchesWhere, groupWhere)
		args = append(args, groupArgs...)
	}

	operator := fmt.Sprintf(" %s ", filter.Bool)
	where = strings.Join(matchesWhere, operator)
	where = fmt.Sprintf("(%s)", where)

	return where, args, nil
}

func (qb baseQueryBuilder) buildFilterMatches(filterMatches []FilterMatch) ([]string, []interface{}, error) {
	where := make([]string, 0, len(filterMatches))
	args := make([]interface{}, 0)

	for _, m := range filterMatches {
		if len(m.Dimension) == 0 {
			continue
		}

		values, valuesArgs, err := qb.buildFilterValues(m)
		if err != nil {
			return where, args, err
		}

		where = append(where, values)
		args = append(args, valuesArgs...)
	}

	return where, args, nil
}

func (qb baseQueryBuilder) buildFilterValues(match FilterMatch) (string, []interface{}, error) {
	if _, ok := qb.mapping[match.Dimension]; !ok {
		return "", []interface{}{}, fmt.Errorf("no list mapping found for dimension %s", match.Dimension)
	}

	if len(match.Values) == 0 {
		return "(1 = 2)", []interface{}{}, nil
	}

	stmts := make([]string, 0)
	args := make([]interface{}, 0)
	mapping := qb.mapping[match.Dimension]

	for _, column := range mapping.Columns() {
		w, a := qb.buildFilterColumn(match, column)

		stmts = append(stmts, w)
		args = append(args, a...)
	}

	b := fmt.Sprintf(" %s ", mapping.Bool())
	where := fmt.Sprintf("(%s)", strings.Join(stmts, b))

	return where, args, nil
}

func (qb baseQueryBuilder) buildFilterColumn(match FilterMatch, column db_repo.FieldMappingColumn) (string, []interface{}) {
	if (match.Operator == OpEq || match.Operator == OpNeq) && len(match.Values) > 1 {
		return qb.buildSetFilterColumn(match, column)
	}

	distinctNull := column.NullMode() == db_repo.NullModeDistinct

	stmts := make([]string, 0, len(match.Values))
	args := make([]interface{}, 0, len(match.Values))

	for _, v := range match.Values {
		switch true {
		case strings.EqualFold(OpIs, match.Operator):
			stmts = append(stmts, fmt.Sprintf("%s IS %s", column.Name(), v))

		case strings.EqualFold(OpIsNot, match.Operator):
			stmts = append(stmts, fmt.Sprintf("%s IS NOT %s", column.Name(), v))

		case match.Operator == OpLike:
			stmts = append(stmts, fmt.Sprintf("%s LIKE ?", column.Name()))
			args = append(args, fmt.Sprintf("%%%v%%", v))

		case match.Operator == OpEq && distinctNull && v == nil:
			stmts = append(stmts, fmt.Sprintf("%s IS NULL", column.Name()))

		case match.Operator == OpNeq && distinctNull && v == nil:
			stmts = append(stmts, fmt.Sprintf("%s IS NOT NULL", column.Name()))

		case match.Operator == OpNeq && distinctNull && v != nil:
			stmts = append(stmts, fmt.Sprintf("%s != ? OR %s IS NULL", column.Name(), column.Name()))
			args = append(args, v)

		case match.Operator == OPMemberOf && v != nil:
			stmts = append(stmts, fmt.Sprintf("? MEMBER OF (%s)", column.Name()))
			args = append(args, v)

		case match.Operator == OPNotMemberOf && v != nil:
			stmts = append(stmts, fmt.Sprintf("NOT ? MEMBER OF (%s)", column.Name()))
			args = append(args, v)

		default:
			stmts = append(stmts, fmt.Sprintf("%s %s ?", column.Name(), match.Operator))
			args = append(args, v)
		}
	}

	where := fmt.Sprintf("(%s)", strings.Join(stmts, " OR "))

	return where, args
}

func (qb baseQueryBuilder) buildSetFilterColumn(match FilterMatch, column db_repo.FieldMappingColumn) (string, []interface{}) {
	distinctNull := column.NullMode() == db_repo.NullModeDistinct
	eq := match.Operator == OpEq

	placeholders, filteredValues, hasNull := qb.buildSetPlaceholders(match, distinctNull)

	var filter string
	if distinctNull {
		if hasNull {
			if eq {
				filter = fmt.Sprintf("(%s IN (%s) OR %s IS NULL)", column.Name(), placeholders, column.Name())
			} else {
				filter = fmt.Sprintf("(%s NOT IN (%s) AND %s IS NOT NULL)", column.Name(), placeholders, column.Name())
			}
		} else if eq {
			filter = fmt.Sprintf("(%s IN (%s))", column.Name(), placeholders)
		} else {
			filter = fmt.Sprintf("(%s NOT IN (%s) OR %s IS NULL)", column.Name(), placeholders, column.Name())
		}
	} else if eq {
		filter = fmt.Sprintf("(%s IN (%s))", column.Name(), placeholders)
	} else {
		filter = fmt.Sprintf("(%s NOT IN (%s))", column.Name(), placeholders)
	}

	return filter, filteredValues
}

func (qb baseQueryBuilder) buildSetPlaceholders(match FilterMatch, distinctNull bool) (string, []interface{}, bool) {
	placeholders := make([]string, 0, len(match.Values))
	filteredValues := make([]interface{}, 0, len(match.Values))

	hasNull := false
	for _, value := range match.Values {
		if value == nil {
			hasNull = true

			if match.Operator == OpNeq && distinctNull {
				continue
			}
		}

		placeholders = append(placeholders, "?")
		filteredValues = append(filteredValues, value)
	}

	return strings.Join(placeholders, ","), filteredValues, hasNull
}
