package log

import (
	"fmt"
	"reflect"
	"time"
)

func mergeFields(receiver map[string]interface{}, input map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{}, len(receiver)+len(input))

	for k, v := range receiver {
		newMap[k] = prepareForLog(v)
	}

	for k, v := range input {
		newMap[k] = prepareForLog(v)
	}

	return newMap
}

func prepareForLog(v interface{}) interface{} {
	switch t := v.(type) {
	case error:
		// Otherwise errors are ignored by `encoding/json`
		return t.Error()
	case time.Time:
		return v
	case map[string]interface{}:
		// perform a deep copy of any maps contained in this map element to ensure we own the object completely
		// also makes sure, that all nested values get prepared to be written as well
		return mergeFields(t, nil)

	default:
		// same as before, but handle the case of the map mapping to something
		// different than interface{}
		// should quite rarely get hit, otherwise you are using too complex objects for your logs
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Map:
			iter := rv.MapRange()
			newMap := make(map[string]interface{}, rv.Len())

			for iter.Next() {
				keyValue := iter.Key()
				elemValue := iter.Value()
				newMap[fmt.Sprint(keyValue.Interface())] = prepareForLog(elemValue.Interface())
			}

			return newMap

		case reflect.Ptr, reflect.Interface:
			if rv.IsNil() {
				return nil
			}

			return prepareForLog(rv.Elem().Interface())

		case reflect.Struct:
			rvt := rv.Type()
			newMap := make(map[string]interface{}, rv.NumField())

			for i := 0; i < rv.NumField(); i++ {
				field := rv.Field(i)
				if !field.CanInterface() {
					continue
				}
				newMap[rvt.Field(i).Name] = prepareForLog(field.Interface())
			}

			return newMap

		case reflect.Slice, reflect.Array:
			if rv.Kind() == reflect.Slice && rv.IsNil() {
				return nil
			}

			newArray := make([]interface{}, rv.Len())

			for i := range newArray {
				newArray[i] = prepareForLog(rv.Index(i).Interface())
			}

			return newArray

		default:
			return v
		}
	}
}
