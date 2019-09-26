package refl

import (
	"reflect"
)

func IsSlice(value interface{}) bool {
	t := reflect.TypeOf(value)

	for {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
			continue
		}

		break
	}

	return t.Kind() == reflect.Slice
}
