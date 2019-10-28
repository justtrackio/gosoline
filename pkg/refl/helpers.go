package refl

import (
	"reflect"
)

func IsPointerToSlice(value interface{}) bool {
	t := reflect.TypeOf(value)

	if t.Kind() != reflect.Ptr {
		return false
	}

	t = t.Elem()

	return t.Kind() == reflect.Slice
}

func IsPointerToStruct(value interface{}) bool {
	t := reflect.TypeOf(value)

	if t.Kind() != reflect.Ptr {
		return false
	}

	t = t.Elem()

	return t.Kind() == reflect.Struct
}
