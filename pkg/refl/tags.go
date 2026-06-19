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
	if typ == nil || typ.Kind() != reflect.Struct {
		return []string{}
	}

	return getTagsFromType(typ, tagName, make(map[reflect.Type]bool))
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
	addTagNamesFromType(set, typ, make(map[reflect.Type]bool))

	names := set.ToSlice()
	sort.Strings(names)

	return names
}

func getTagsFromType(typ reflect.Type, tagName string, seen map[reflect.Type]bool) []string {
	if seen[typ] {
		return []string{}
	}

	seen[typ] = true
	defer delete(seen, typ)

	var availableFields []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag, ok := field.Tag.Lookup(tagName)

		if !ok {
			if embeddedType, isEmbeddedStruct := resolveEmbeddedStructType(field); isEmbeddedStruct {
				availableFields = append(availableFields, getTagsFromType(embeddedType, tagName, seen)...)
			}

			continue
		}

		if tag == "" || tag == "-" {
			continue
		}

		parts := strings.Split(tag, ",")
		availableFields = append(availableFields, parts[0])
	}

	return availableFields
}

func resolveEmbeddedStructType(field reflect.StructField) (reflect.Type, bool) {
	if !field.Anonymous {
		return nil, false
	}

	typ := field.Type
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return typ, typ.Kind() == reflect.Struct
}

func addTagNamesFromType(set funk.Set[string], typ reflect.Type, seen map[reflect.Type]bool) {
	if seen[typ] {
		return
	}

	seen[typ] = true
	defer delete(seen, typ)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		raw := string(field.Tag)
		if raw == "" {
			if embeddedType, isEmbeddedStruct := resolveEmbeddedStructType(field); isEmbeddedStruct {
				addTagNamesFromType(set, embeddedType, seen)
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
}
