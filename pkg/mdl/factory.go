package mdl

import (
	"reflect"
	"time"
)

func Bool(v bool) *bool {
	return &v
}

func Float32(v float32) *float32 {
	return &v
}

func Float64(v float64) *float64 {
	return &v
}

func Int(v int) *int {
	return &v
}

func Int32(v int32) *int32 {
	return &v
}

func Int64(v int64) *int64 {
	return &v
}

func String(v string) *string {
	return &v
}

func Uint(v uint) *uint {
	return &v
}

func EmptyBoolIfNil(b *bool) bool {
	if b == nil {
		return false
	}

	return *b
}

func EmptyFloat64IfNil(v *float64) float64 {
	if v == nil {
		return 0.0
	}

	return *v
}

func EmptyIntIfNil(v *int) int {
	if v == nil {
		return 0
	}

	return *v
}

func EmptyStringIfNil(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func EmptyTimeIfNil(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}

	return *t
}

func EmptyUintIfNil(i *uint) uint {
	if i == nil {
		return 0
	}

	return *i
}

func Time(t time.Time) *time.Time {
	return &t
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
