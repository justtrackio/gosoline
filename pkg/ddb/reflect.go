package ddb

import (
	"context"
	"reflect"
	"strings"
)

func getTypeName(value any) string {
	t := findBaseType(value)
	name := t.Name()

	return strings.ToLower(string(name[0])) + name[1:]
}

func isPointer(value any) bool {
	return value != nil && reflect.TypeOf(value).Kind() == reflect.Ptr
}

func isStruct(value any) bool {
	if value == nil {
		return false
	}

	t := findBaseType(value)

	return t.Kind() == reflect.Struct
}

func findBaseType(value any) reflect.Type {
	t := reflect.TypeOf(value)

	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}

	return t
}

func isResultCallback(value any) (func(ctx context.Context, items any, progress Progress) (bool, error), bool) {
	if callback, ok := value.(func(ctx context.Context, items any, progress Progress) (bool, error)); ok {
		return callback, true
	}

	return nil, false
}

func chunk(batch []any, size int) [][]any {
	var chunks [][]any

	for i := 0; i < len(batch); i += size {
		end := i + size

		if end > len(batch) {
			end = len(batch)
		}

		chunks = append(chunks, batch[i:end])
	}

	return chunks
}
