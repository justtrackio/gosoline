package log

import (
	"fmt"
	"reflect"
	"time"
)

func mergeFields(receiver map[string]any, input map[string]any) map[string]any {
	newMap := make(map[string]any, len(receiver)+len(input))

	for k, v := range receiver {
		if k == "" {
			continue
		}

		newMap[k] = prepareForLog(v)
	}

	for k, v := range input {
		if k == "" {
			continue
		}

		newMap[k] = prepareForLog(v)
	}

	return newMap
}

func prepareForLog(v any) any {
	switch t := v.(type) {
	case error:
		// Otherwise errors are ignored by `encoding/json`
		return t.Error()
	case time.Time:
		return v
	case map[string]any:
		// perform a deep copy of any maps contained in this map element to ensure we own the object completely
		// also makes sure, that all nested values get prepared to be written as well
		return mergeFields(t, nil)

	default:
		// same as before, but handle the case of the map mapping to something
		// different than any
		// should quite rarely get hit, otherwise you are using too complex objects for your logs
		return prepareReflectValue(reflect.ValueOf(v))
	}
}

func prepareReflectValue(rv reflect.Value) any {
	switch rv.Kind() {
	case reflect.Invalid:
		return nil

	case reflect.Map:
		return prepareMapForLog(rv)

	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return nil
		}

		return prepareForLog(rv.Elem().Interface())

	case reflect.Struct:
		return prepareStructForLog(rv)

	case reflect.Slice, reflect.Array:
		return prepareSliceForLog(rv)

	default:
		return rv.Interface()
	}
}

func prepareMapForLog(rv reflect.Value) any {
	iter := rv.MapRange()
	newMap := make(map[string]any, rv.Len())

	for iter.Next() {
		keyValue := iter.Key()
		key := fmt.Sprint(keyValue.Interface())
		if key == "" {
			continue
		}
		elemValue := iter.Value()
		newMap[key] = prepareForLog(elemValue.Interface())
	}

	return newMap
}

func prepareStructForLog(rv reflect.Value) any {
	rvt := rv.Type()
	newMap := make(map[string]any, rv.NumField())

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		if !field.CanInterface() {
			continue
		}
		fieldName := rvt.Field(i).Name
		if fieldName == "" {
			continue
		}
		newMap[fieldName] = prepareForLog(field.Interface())
	}

	return newMap
}

func prepareSliceForLog(rv reflect.Value) any {
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return nil
	}

	newArray := make([]any, rv.Len())

	for i := range newArray {
		newArray[i] = prepareForLog(rv.Index(i).Interface())
	}

	return newArray
}
