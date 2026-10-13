package cfg

import (
	"context"
	"crypto/md5"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/justtrackio/gosoline/pkg/mapx"
)

// DebugConfig logs flattened settings and a fingerprint of the same redacted
// entries. Annotate secret fields with cfg:"name,secret" and register their
// types through rooted merges or UnmarshalKey before calling DebugConfig.
// Secret parents hide their entire contents, including map keys. Raw getters
// do not register annotations. Redaction changes neither storage nor decoded
// values; other consumers of AllSettings still receive the original values.
func DebugConfig(ctx context.Context, config Config, logger Logger) error {
	entries := make(map[string]debugEntry)
	if err := debugFlatten(config, reflect.ValueOf(config.AllSettings()), "", "", entries, make(map[debugVisit]bool)); err != nil {
		return err
	}
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if entries[keys[i]].path == entries[keys[j]].path {
			return keys[i] < keys[j]
		}

		return entries[keys[i]].path < entries[keys[j]].path
	})
	hashValues := make([]string, len(keys))
	for i, key := range keys {
		entry := entries[key]
		hashValues[i] = fmt.Sprintf("%v=%v", entry.path, entry.value)
		logger.Info(ctx, "cfg %s", hashValues[i])
	}
	hashBytes := md5.Sum([]byte(strings.Join(hashValues, ";")))
	logger.Info(ctx, "cfg fingerprint: %x", hashBytes)

	return nil
}

type (
	debugVisit struct {
		kind    reflect.Kind
		pointer uintptr
	}
	debugEntry struct {
		path  string
		value any
	}
)

func debugFlatten(config Config, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if config.IsSecret(selector) {
		entries[selector] = debugEntry{path, "[REDACTED]"}

		return nil
	}
	if !value.IsValid() {
		entries[selector] = debugEntry{path, nil}

		return nil
	}

	return debugFlattenValue(config, value, path, selector, entries, active)
}

func debugFlattenValue(config Config, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		return debugFlattenPointer(config, value, path, selector, entries, active)
	case reflect.Map:
		return debugFlattenMap(config, value, path, selector, entries, active)
	case reflect.Slice, reflect.Array:
		return debugFlattenSlice(config, value, path, selector, entries, active)
	case reflect.Struct:
		return debugFlattenStruct(config, value, path, selector, entries, active)
	default:
		entries[selector] = debugEntry{path, value.Interface()}

		return nil
	}
}

func debugFlattenPointer(config Config, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if value.IsNil() {
		entries[selector] = debugEntry{path, nil}

		return nil
	}
	if value.Kind() == reflect.Pointer {
		visit := debugVisit{value.Kind(), value.Pointer()}
		if active[visit] {
			return fmt.Errorf("can not flatten cyclic config settings")
		}
		active[visit] = true
		defer delete(active, visit)
	}

	return debugFlattenValue(config, value.Elem(), path, selector, entries, active)
}

func debugFlattenMap(config Config, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	visit := debugVisit{value.Kind(), uintptr(value.UnsafePointer())}
	if active[visit] {
		return fmt.Errorf("can not flatten cyclic config settings")
	}
	active[visit] = true
	defer delete(active, visit)
	iter := value.MapRange()
	for iter.Next() {
		key := fmt.Sprint(iter.Key().Interface())
		escapedKey := strings.Join(mapx.SplitUnescapedDotN(key, -1), "\\.")
		if err := debugFlatten(config, iter.Value(), debugJoin(path, key), debugJoin(selector, escapedKey), entries, active); err != nil {
			return err
		}
	}

	return nil
}

func debugFlattenSlice(config Config, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if value.Kind() == reflect.Slice {
		visit := debugVisit{value.Kind(), value.Pointer()}
		if active[visit] {
			return fmt.Errorf("can not flatten cyclic config settings")
		}
		active[visit] = true
		defer delete(active, visit)
	}
	for i := range value.Len() {
		if err := debugFlatten(config, value.Index(i), debugJoin(path, fmt.Sprint(i)), fmt.Sprintf("%s[%d]", selector, i), entries, active); err != nil {
			return err
		}
	}

	return nil
}

func debugFlattenStruct(config Config, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if value.Type() == reflect.TypeOf(time.Time{}) {
		entries[selector] = debugEntry{path, value.Interface()}

		return nil
	}
	for i := range value.NumField() {
		field := value.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, ok := field.Tag.Lookup("cfg")
		if field.Anonymous {
			if err := debugFlatten(config, value.Field(i), path, selector, entries, active); err != nil {
				return err
			}

			continue
		}
		if !ok {
			continue
		}
		name = strings.TrimSpace(strings.Split(name, ",")[0])
		if err := debugFlatten(config, value.Field(i), debugJoin(path, name), debugJoin(selector, name), entries, active); err != nil {
			return err
		}
	}

	return nil
}

func debugJoin(parent, key string) string {
	if parent == "" {
		return key
	}

	return parent + "." + key
}
