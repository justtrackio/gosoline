package refl

import (
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

var tagKeyRegex = regexp.MustCompile(`(\w+):"[^"]*"`)

// GetTags reads the values of tags with the name tagName.
// Tag values get extracted until the first comma.
// Nested structs without or with an empty tag are embedded into the result.
// Best suited to read json and db tags.
func GetTags(v any, tagName string) []string {
	typ := ResolveBaseType(v)

	var ok bool
	var tag string
	var availableFields, parts []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag, ok = field.Tag.Lookup(tagName); !ok || tag == "" {
			if field.Type.Kind() == reflect.Struct {
				availableFields = append(availableFields, GetTags(reflect.New(field.Type).Elem().Interface(), tagName)...)
			}

			continue
		}

		if tag == "-" {
			continue
		}

		parts = strings.Split(tag, ",")
		availableFields = append(availableFields, parts[0])
	}

	return availableFields
}

// GetTagNames returns a sorted list of distinct struct tag keys present on the fields of the given value.
// Nested structs without any tags are handled as if they were embedded.
// Works with structs, pointers to structs, and slices of structs. Returns an empty slice for nil or non-structs.
func GetTagNames(v any) []string {
	if v == nil {
		return []string{}
	}

	// handle typed nil pointer
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Ptr && rv.IsNil() {
		return []string{}
	}

	typ := ResolveBaseType(v)
	if typ == nil || typ.Kind() != reflect.Struct {
		return []string{}
	}

	set := make(funk.Set[string])
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		raw := string(field.Tag)
		if raw == "" {
			if field.Type.Kind() == reflect.Struct {
				set.Add(GetTagNames(reflect.New(field.Type).Elem().Interface())...)
			}

			continue
		}

		allMatches := tagKeyRegex.FindAllStringSubmatch(raw, -1)
		for _, match := range allMatches {
			if len(match) > 1 {
				set.Add(match[1])
			}
		}
	}

	names := set.ToSlice()
	sort.Strings(names)

	return names
}
