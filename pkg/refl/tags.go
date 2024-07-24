package refl

import (
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
