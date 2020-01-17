package refl

import (
	"reflect"
)

func IsPointerToSlice(value interface{}) bool {
	t := reflect.TypeOf(value)

	if t == nil || t.Kind() != reflect.Ptr {
		return false
	}

	t = t.Elem()

	if t.Kind() == reflect.Interface {
		v := reflect.ValueOf(value).Elem().Interface()
		t = reflect.TypeOf(v)
	}

	return t.Kind() == reflect.Slice
}

func IsPointerToStruct(value interface{}) bool {
	t := reflect.TypeOf(value)

	if t == nil || t.Kind() != reflect.Ptr {
		return false
	}

	t = t.Elem()

	if t.Kind() == reflect.Interface {
		v := reflect.ValueOf(value).Elem().Interface()
		t = reflect.TypeOf(v)
	}

	return t.Kind() == reflect.Struct
}

func IsSlice(value interface{}) bool {
	t := reflect.TypeOf(value)

	return t.Kind() == reflect.Slice
}

func ResolveBaseTypeAndValue(value interface{}) (reflect.Type, reflect.Value) {
	return ResolveValueTo(value, reflect.Invalid)
}

func ResolveBaseType(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	if t == nil {
		return nil
	}

	if t.Kind() == reflect.Ptr {
		v := reflect.ValueOf(value).Elem().Interface()

		return ResolveBaseType(v)
	}

	if t.Kind() == reflect.Interface {
		v := reflect.ValueOf(value).Elem().Interface()
		t = reflect.TypeOf(v)
	}

	if t.Kind() != reflect.Slice {
		return t
	}

	t = t.Elem()

	if t.Kind() == reflect.Interface {
		v := reflect.ValueOf(value).Index(0).Interface()
		t = reflect.TypeOf(v)
	}

	ts := reflect.SliceOf(t)

	slice := reflect.MakeSlice(ts, 1, 1).Interface()
	v := reflect.ValueOf(slice)

	return ResolveBaseType(v.Index(0).Interface())
}

func ResolveValueTo(value interface{}, kind reflect.Kind) (reflect.Type, reflect.Value) {
	t := reflect.TypeOf(value)
	v := reflect.ValueOf(value)

	if t == nil {
		return nil, reflect.Value{}
	}

	if kind == t.Kind() {
		return t, v
	}

	if t.Kind() == reflect.Interface {
		v = v.Elem()
		t = v.Type()

		value = v.Interface()
		return ResolveBaseTypeAndValue(value)
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()

		value = v.Interface()
		return ResolveBaseTypeAndValue(value)
	}

	if t.Kind() == reflect.Slice {
		t = t.Elem()
		v = v.Index(0)

		value = v.Interface()
		return ResolveBaseTypeAndValue(value)
	}

	return t, v
}

func GetTypedValue(value interface{}) reflect.Value {
	t := reflect.TypeOf(value)

	if t.Kind() == reflect.Ptr {
		v := reflect.ValueOf(value).Elem().Interface()

		return GetTypedValue(v)
	}

	v := value

	if t.Kind() == reflect.Interface {
		v = reflect.ValueOf(value).Elem().Interface()
	}

	return reflect.ValueOf(v)
}

func CreatePointerToSliceOfTypeAndSize(value interface{}, size int) interface{} {
	baseType := ResolveBaseType(value)

	sliceType := reflect.SliceOf(baseType)
	slice := reflect.MakeSlice(sliceType, size, size)

	pt := reflect.PtrTo(slice.Type())
	pv := reflect.New(pt.Elem())
	pv.Elem().Set(slice)

	ptr := pv.Interface()

	return ptr
}

func CopyPointerSlice(ptrA interface{}, ptrB interface{}) {
	pv := reflect.ValueOf(ptrB)

	a := reflect.ValueOf(ptrA).Elem()
	b := reflect.Indirect(pv.Elem())

	a.Set(b)
}

func InitializeMapsAndSlices(value interface{}) {
	pv := reflect.ValueOf(value)

	if pv.Kind() == reflect.Ptr {
		pv = pv.Elem()
	}

	for i := 0; i < pv.NumField(); i++ {
		field := pv.Field(i)

		if field.Kind() == reflect.Map && field.IsNil() {
			mapType := field.Type()
			mapValue := reflect.MakeMap(mapType)

			field.Set(mapValue)
		}
	}
}
