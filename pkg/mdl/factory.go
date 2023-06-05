package mdl

import (
	"reflect"
)

func Empty[T any]() (out T) {
	return
}

func EmptyIfNil[T any](v *T) T {
	return Unbox(v, Empty[T]())
}

func NilIfEmpty[T comparable](in T) *T {
	if Empty[T]() == in {
		return nil
	}

	return &in
}

func IsNilOrEmpty[T comparable](in *T) bool {
	return in == nil || *in == Empty[T]()
}

func Box[T any](v T) *T {
	return &v
}

func Unbox[T any](v *T, def T) T {
	if v == nil {
		return def
	}

	return *v
}

func UnboxWith[T any](v *T, mkDef func() T) T {
	if v == nil {
		return mkDef()
	}

	return *v
}

func IsNil(m any) bool {
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
