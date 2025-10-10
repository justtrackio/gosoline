package refl

import (
	"reflect"
	"sort"
	"strings"
)

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

	set := make(map[string]struct{})
	names := make([]string, 0)

	for i := 0; i < typ.NumField(); i++ {
		raw := string(typ.Field(i).Tag)
		if raw == "" {
			continue
		}

		// parse raw struct tag: key:"value" pairs separated by spaces
		for j := 0; j < len(raw); {
			// skip spaces
			for j < len(raw) && raw[j] == ' ' {
				j++
			}
			if j >= len(raw) {
				break
			}

			start := j
			for j < len(raw) && raw[j] > ' ' && raw[j] != ':' && raw[j] != '"' {
				j++
			}
			if j >= len(raw) || raw[j] != ':' {
				break
			}
			key := raw[start:j]
			j++ // skip ':'
			if j >= len(raw) || raw[j] != '"' {
				break
			}
			j++ // skip opening quote
			for j < len(raw) {
				if raw[j] == '"' && (j == start || raw[j-1] != '\\') {
					j++
					break
				}
				j++
			}

			if key != "" {
				if _, ok := set[key]; !ok {
					set[key] = struct{}{}
					names = append(names, key)
				}
			}
		}
	}

	sort.Strings(names)

	return names
}
