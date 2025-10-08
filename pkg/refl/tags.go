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
// Best suited to read json and db tags.
func GetTags(v any, tagName string) []string {
	typ := ResolveBaseType(v)

	var ok bool
	var tag string
	var availableFields, parts []string

	for i := 0; i < typ.NumField(); i++ {
		if tag, ok = typ.Field(i).Tag.Lookup(tagName); !ok {
			continue
		}

		if tag == "" {
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
		raw := string(typ.Field(i).Tag)
		if raw == "" {
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
