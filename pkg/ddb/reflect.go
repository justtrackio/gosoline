package ddb

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func getTypeName(value interface{}) string {
	t := reflect.TypeOf(value)
	name := t.Name()

	return strings.ToLower(string(name[0])) + name[1:]
}

func isPointer(value interface{}) bool {
	return value != nil && reflect.TypeOf(value).Kind() == reflect.Ptr
}

func isStruct(value interface{}) bool {
	if value == nil {
		return false
	}

	t := reflect.TypeOf(value)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Kind() == reflect.Struct
}

func isResultCallback(value interface{}) (func(ctx context.Context, result interface{}) (bool, error), bool) {
	t := reflect.TypeOf(value)

	if t.Kind() != reflect.Func {
		return nil, false
	}

	if callback, ok := value.(func(ctx context.Context, result interface{}) (bool, error)); ok {
		return callback, true
	}

	return nil, false
}

func interfaceToSliceOfInterfaces(sliceOfItems interface{}) ([]interface{}, error) {
	s := reflect.ValueOf(sliceOfItems)

	if s.Kind() != reflect.Slice {
		return nil, fmt.Errorf("value is not of type slice")
	}

	items := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		items[i] = s.Index(i).Interface()
	}

	return items, nil
}

func chunk(batch []interface{}, size int) [][]interface{} {
	var chunks [][]interface{}

	for i := 0; i < len(batch); i += size {
		end := i + size

		if end > len(batch) {
			end = len(batch)
		}

		chunks = append(chunks, batch[i:end])
	}

	return chunks
}
