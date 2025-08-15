package dbx

import (
	"fmt"
	"reflect"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/lann/builder"
)

type wherePart part

func newWherePart(pred interface{}, args ...interface{}) Sqlizer {
	return &wherePart{pred: pred, args: args}
}

func (p wherePart) ToSql() (sql string, args []interface{}, err error) {
	switch pred := p.pred.(type) {
	case nil:
		// no-op
	case rawSqlizer:
		return pred.toSqlRaw()
	case Sqlizer:
		return pred.ToSql()
	case map[string]interface{}:
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

func (p whereStruct[T]) ToSql() (sql string, args []interface{}, err error) {
	var st *mapx.Struct
	var mpx *mapx.MapX

	if st, err = mapx.NewStruct(&p.val, &mapx.StructSettings{FieldTag: "db"}); err != nil {
		return
	}

	if mpx, err = st.Read(); err != nil {
		return
	}

	values := funk.MapFilter(mpx.Msi(), func(key string, value any) bool {
		vt := reflect.TypeOf(value)
		zeroValue := reflect.Zero(vt).Interface()

		return value != zeroValue
	})

	return Eq(values).ToSql()
}

func applyWhere[T any](b any, pred interface{}, args ...interface{}) any {
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
