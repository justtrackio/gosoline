package sql

import (
	"fmt"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/db-repo"
	"strings"
)

type Order struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type Page struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type Filter struct {
	Matches []FilterMatch `json:"matches"`
	Groups  []Filter      `json:"groups"`
	Bool    string        `json:"bool"`
}

type FilterMatch struct {
	Dimension string        `json:"dimension"`
	Operator  string        `json:"operator"`
	Values    []interface{} `json:"values"`
}

type Input struct {
	Filter Filter  `json:"filter"`
	Order  []Order `json:"order"`
	Page   *Page   `json:"page"`
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
	metadata db_repo.Metadata
	mapping  db_repo.FieldMappings
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

	dbQb.Table(qb.metadata.TableName)
	dbQb.Joins(joins)
	dbQb.Where(query, args...)
	dbQb.GroupBy(qb.metadata.PrimaryKey)

	for _, o := range inp.Order {
		if _, ok := qb.mapping[o.Field]; !ok {
			return fmt.Errorf("no list mapping found for order field %s", o.Field)
		}

		columns := strings.Join(qb.mapping[o.Field].Columns, ", ")
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

		if len(qb.mapping[o.Field].Joins) == 0 {
			continue
		}

		*joins = append(*joins, qb.mapping[o.Field].Joins...)
	}

	return nil
}

func (qb baseQueryBuilder) getJoinsFromFilter(joins *[]string, filter Filter) error {
	for _, m := range filter.Matches {
		if _, ok := qb.mapping[m.Dimension]; !ok {
			return fmt.Errorf("no list mapping found for dimension %s", m.Dimension)
		}

		if len(qb.mapping[m.Dimension].Joins) == 0 {
			continue
		}

		*joins = append(*joins, qb.mapping[m.Dimension].Joins...)
	}

	if filter.Groups == nil {
		return nil
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

	for _, column := range mapping.Columns {
		w, a := qb.buildFilterColumn(match, column)

		stmts = append(stmts, w)
		args = append(args, a...)
	}

	b := fmt.Sprintf(" %s ", mapping.Bool)
	where := fmt.Sprintf("(%s)", strings.Join(stmts, b))

	return where, args, nil
}

func (qb baseQueryBuilder) buildFilterColumn(match FilterMatch, column string) (string, []interface{}) {
	if match.Operator == "=" && len(match.Values) > 1 {
		placeholders := make([]string, 0, len(match.Values))
		for _ = range match.Values {
			placeholders = append(placeholders, "?")
		}
		return fmt.Sprintf("(%s IN (%s))", column, strings.Join(placeholders, ",")), match.Values
	}

	stmts := make([]string, len(match.Values))
	args := make([]interface{}, len(match.Values))

	for i, v := range match.Values {
		switch match.Operator {
		case "~":
			stmts[i] = fmt.Sprintf("%s LIKE ?", column)
			args[i] = fmt.Sprintf("%%%v%%", v)

		default:
			stmts[i] = fmt.Sprintf("%s %s ?", column, match.Operator)
			args[i] = v
		}
	}

	where := fmt.Sprintf("(%s)", strings.Join(stmts, " OR "))

	return where, args
}
