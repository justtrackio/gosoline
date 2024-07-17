package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

var (
	_                  driver.Valuer = JSON[struct{}]{}
	_                  sql.Scanner   = &JSON[struct{}]{}
	ErrJSONInvalidType               = errors.New("incoming data type is not []byte for json")
)

type (
	JSON[T any] struct {
		val T
	}
)

func NewJSON[T any](val T) JSON[T] {
	return JSON[T]{
		val: val,
	}
}

func (t JSON[T]) Get() T {
	return t.val
}

func (t JSON[T]) Value() (driver.Value, error) {
	if mdl.IsNil(t.val) {
		return nil, nil //nolint:nilnil // this is the expected behaviour by the driver package
	}

	return json.Marshal(t.val)
}

func (t *JSON[T]) Scan(src any) error {
	if src == nil {
		return nil
	}

	switch src := src.(type) {
	case []byte:
		return json.Unmarshal(src, &t.val)
	default:
		return ErrJSONInvalidType
	}
}
