package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

var (
	_                        driver.Valuer = JSONType[struct{}]{}
	_                        sql.Scanner   = &JSONType[struct{}]{}
	ErrJSONInvalidDriverType               = errors.New("invalid incoming driver type for json type")
)

type (
	JSONType[T any] struct {
		val        T
		asJsonNull bool
	}
)

func NewJSONType[T any](val T) JSONType[T] {
	return JSONType[T]{
		val: val,
	}
}

// AsJSONNull causes the JSONType to produce a "null" string whet the underlying value is nil instead
// of the normal SQL Null.
func (t *JSONType[T]) AsJSONNull() JSONType[T] {
	t.asJsonNull = true

	return *t
}

func (t JSONType[T]) Get() T {
	return t.val
}

func (t JSONType[T]) Value() (driver.Value, error) {
	if mdl.IsNil(t.val) && !t.asJsonNull {
		return nil, nil //nolint:nilnil // this is the expected behaviour by the driver package
	}

	return json.Marshal(t.val)
}

func (t *JSONType[T]) Scan(src any) error {
	if src == nil {
		return nil
	}

	switch src := src.(type) {
	case []byte:
		return json.Unmarshal(src, &t.val)
	default:
		return ErrJSONInvalidDriverType
	}
}
