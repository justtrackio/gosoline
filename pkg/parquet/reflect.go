package parquet

import (
	"reflect"
)

func findBaseType(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	for {
		if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
			t = t.Elem()
			continue
		}

		break
	}

	return t
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
