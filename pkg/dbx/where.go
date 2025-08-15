package dbx

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/lann/builder"
)

type wherePart part

func newWherePart(pred any, args ...any) Sqlizer {
	return &wherePart{pred: pred, args: args}
}

func (p wherePart) ToSql() (sql string, args []any, err error) {
	switch pred := p.pred.(type) {
	case nil:
		// no-op
	case rawSqlizer:
		return pred.toSqlRaw()
	case Sqlizer:
		return pred.ToSql()
	case map[string]any:
		return Eq(pred).ToSql()
	case string:
		return pred, p.args, nil
	}

	err = fmt.Errorf("expected string-keyed map or string, not %T", p.pred)

	return
}

type whereStruct[T any] struct {
	val T
}

func (p whereStruct[T]) ToSql() (sql string, args []any, err error) {
	var msi map[string]any

	if msi, err = toNonZeroMap(p.val); err != nil {
		return "", nil, fmt.Errorf("unable to convert struct to map: %w", err)
	}

	return Eq(msi).ToSql()
}

func toNonZeroMap[T any](val T) (map[string]any, error) {
	var st *mapx.Struct
	var mpx *mapx.MapX
	var err error

	if st, err = mapx.NewStruct(&val, &mapx.StructSettings{FieldTag: "db"}); err != nil {
		return nil, err
	}

	if mpx, err = st.ReadNonZero(); err != nil {
		return nil, err
	}

	return mpx.Msi(), nil
}

func applyWhere[T any](b any, pred any, args ...any) any {
	if pred == nil || pred == "" {
		return b
	}

	switch val := pred.(type) {
	case T:
		return builder.Append(b, "WhereParts", whereStruct[T]{val: val})
	default:
		return builder.Append(b, "WhereParts", newWherePart(pred, args...))
	}
}
