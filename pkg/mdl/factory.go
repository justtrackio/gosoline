package mdl

import "time"

func Bool(v bool) *bool {
	return &v
}

func Float64(v float64) *float64 {
	return &v
}

func Int(v int) *int {
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

func EmptyUintIfNil(i *uint) uint {
	if i == nil {
		return 0
	}

	return *i
}

func Time(t time.Time) *time.Time {
	return &t
}
