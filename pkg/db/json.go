package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

var (
	_                    driver.Valuer = JSON[struct{}, Nullable]{}
	_                    sql.Scanner   = &JSON[struct{}, Nullable]{}
	ErrJSONInvalidType                 = errors.New("incoming data type is not []byte for json")
	NullableBehaviour                  = Nullable{}
	NonNullableBehaviour               = NonNullable{}
)

type (
	Nullable    struct{}
	NonNullable struct{}

	IsNullable interface {
		IsNullable() bool
	}

	// JSON is a type wrapping other types for easy use of database json columns.
	// It is intended to be used without a pointer but having pointers to T.
	//
	JSON[T any, NullBehaviour Nullable | NonNullable] struct {
		val T
	}
)

func (n Nullable) IsNullable() bool {
	return true
}

func (n NonNullable) IsNullable() bool {
	return false
}

func behaviourIsNullable[T Nullable | NonNullable]() bool {
	var n T

	return IsNullable(n).IsNullable()
}

func NewJSON[T any, NullBehaviour Nullable | NonNullable](val T, _ NullBehaviour) JSON[T, NullBehaviour] {
	return JSON[T, NullBehaviour]{
		val: val,
	}
}

func (t JSON[T, NullBehaviour]) Get() T {
	return t.val
}

func (t JSON[T, NullBehaviour]) Value() (driver.Value, error) {
	if behaviourIsNullable[NullBehaviour]() && mdl.IsNil(t.val) {
		return nil, nil //nolint:nilnil // this is the expected behaviour by the driver package
	}

	return json.Marshal(t.val)
}

func (t *JSON[T, NullBehaviour]) Scan(src any) error {
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

func (t *JSON[T, NullBehaviour]) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.val)
}

func (t *JSON[T, NullBehaviour]) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, &t.val)
}
