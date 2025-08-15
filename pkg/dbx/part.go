package dbx

import (
	"fmt"
	"io"
)

type part struct {
	pred any
	args []any
}

func newPart(pred any, args ...any) Sqlizer {
	return &part{pred, args}
}

func (p part) ToSql() (sql string, args []any, err error) {
	switch pred := p.pred.(type) {
	case nil:
		// no-op
	case Sqlizer:
		sql, args, err = pred.ToSql()
	case string:
		sql = pred
		args = p.args
	default:
		err = fmt.Errorf("expected string or Sqlizer, not %T", pred)
	}

	return
}

func nestedToSql(s Sqlizer) (sql string, args []any, err error) {
	if raw, ok := s.(rawSqlizer); ok {
		return raw.toSqlRaw()
	}

	return s.ToSql()
}

func appendToSql(parts []Sqlizer, w io.Writer, sep string, args []any) ([]any, error) {
	for i, p := range parts {
		partSql, partArgs, err := nestedToSql(p)
		if err != nil {
			return nil, err
		} else if partSql == "" {
			continue
		}

		if i > 0 {
			_, err := io.WriteString(w, sep)
			if err != nil {
				return nil, err
			}
		}

		_, err = io.WriteString(w, partSql)
		if err != nil {
			return nil, err
		}
		args = append(args, partArgs...)
	}

	return args, nil
}
