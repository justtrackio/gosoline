package mdl

import (
	"reflect"
	"time"

	"golang.org/x/exp/constraints"
)

type Basic interface {
	~bool | constraints.Float | constraints.Integer | time.Time | ~string
}

func EmptyIfNil[T comparable](v *T) (out T) {
	if v != nil {
		return *v
	}

	return
}

func NilIfEmpty[T comparable](in T) *T {
	if *new(T) == in {
		return nil
	}

	return &in
}

func Box[T Basic](v T) (out *T) {
	return &v
}

func IsNil(m interface{}) bool {
	if m == nil {
		return true
	}

	v := reflect.ValueOf(m)

	switch v.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Map,
		reflect.Ptr,
		reflect.UnsafePointer,
		reflect.Interface,
		reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
