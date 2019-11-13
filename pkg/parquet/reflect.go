package parquet

import (
	"reflect"
)

func findBaseType(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}

	if t.Kind() != reflect.Slice {
		return t
	}

	v, ok := value.([]interface{})

	if !ok {
		return t.Elem()
	}

	return findBaseType(v[0])
}

func createPointerToSliceOfTypeAndSize(value interface{}, size int) interface{} {
	baseType := findBaseType(value)
	sliceType := reflect.SliceOf(baseType)

	slice := reflect.MakeSlice(sliceType, size, size)

	pt := reflect.PtrTo(slice.Type())
	pv := reflect.New(pt.Elem())
	pv.Elem().Set(slice)

	ptr := pv.Interface()

	return ptr
}

func copyPointerSlice(ptrA interface{}, ptrB interface{}) {
	pv := reflect.ValueOf(ptrB)

	a := reflect.ValueOf(ptrA).Elem()
	b := reflect.Indirect(pv.Elem())

	a.Set(b)
}
