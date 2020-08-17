package ddb

import (
	"context"
	"reflect"
	"strings"
)

func getTypeName(value interface{}) string {
	t := findBaseType(value)
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

	t := findBaseType(value)

	return t.Kind() == reflect.Struct
}

func findBaseType(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}

	return t
}

func isResultCallback(value interface{}) (func(ctx context.Context, items interface{}, progress Progress) (bool, error), bool) {
	if callback, ok := value.(func(ctx context.Context, items interface{}, progress Progress) (bool, error)); ok {
		return callback, true
	}

	return nil, false
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
