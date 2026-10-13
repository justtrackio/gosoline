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

// DebugConfig logs settings and a fingerprint of exactly the redacted entries.
// Native configs retain rooted cfg:",secret" annotations for this logging-only
// projection; current retained structs also enforce their current tags. Imported
// SDK owners use RegisterConfigDebugType to protect settings even before decoding.
// Secret parents hide their contents, including keys. Getters, AllSettings and
// decoded values are unchanged. Unknown Config implementations and unsupported
// public values fail before any logging; raw getters do not register annotations.
func DebugConfig(ctx context.Context, config Config, logger Logger) error {
	metadata, ok := config.(interface {
		loggingSecrets(map[string]any) (map[string]bool, error)
	})
	if !ok {
		return fmt.Errorf("config %T does not provide safe logging metadata", config)
	}
	settings := config.AllSettings()
	secrets, err := metadata.loggingSecrets(settings)
	if err != nil {
		return err
	}
	entries := make(map[string]debugEntry)
	if err := debugFlatten(secrets, reflect.ValueOf(settings), "", "", entries, make(map[debugVisit]bool)); err != nil {
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
		typ     reflect.Type
	}
	debugEntry struct {
		path  string
		value any
	}
)

func debugFlatten(secrets map[string]bool, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if secrets[selector] {
		entries[selector] = debugEntry{path, "[REDACTED]"}

		return nil
	}
	if !value.IsValid() {
		entries[selector] = debugEntry{path, nil}

		return nil
	}

	return debugFlattenValue(secrets, value, path, selector, entries, active)
}

func debugFlattenValue(secrets map[string]bool, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		return debugFlattenPointer(secrets, value, path, selector, entries, active)
	case reflect.Map:
		return debugFlattenMap(secrets, value, path, selector, entries, active)
	case reflect.Slice, reflect.Array:
		return debugFlattenSequence(secrets, value, path, selector, entries, active)
	case reflect.Struct:
		return debugFlattenStruct(secrets, value, path, selector, entries, active)
	default:
		scalar, err := debugScalar(value, path)
		if err != nil {
			return err
		}
		entries[selector] = debugEntry{path, scalar}

		return nil
	}
}

func debugFlattenPointer(secrets map[string]bool, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if value.IsNil() {
		entries[selector] = debugEntry{path, nil}

		return nil
	}
	visit := loggingVisit(value)
	if visit.pointer != 0 {
		if active[visit] {
			return fmt.Errorf("cyclic config settings at %s (type %v)", path, value.Type())
		}
		active[visit] = true
		defer delete(active, visit)
	}

	return debugFlattenValue(secrets, value.Elem(), path, selector, entries, active)
}

func debugFlattenMap(secrets map[string]bool, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if value.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("unsupported config map at %s (type %v)", path, value.Type())
	}
	visit := loggingVisit(value)
	if active[visit] {
		return fmt.Errorf("cyclic config settings at %s (type %v)", path, value.Type())
	}
	active[visit] = true
	defer delete(active, visit)
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		if err := debugFlatten(secrets, iter.Value(), debugJoin(path, key), debugJoin(selector, loggingKey(key)), entries, active); err != nil {
			return err
		}
	}

	return nil
}

func debugFlattenSequence(secrets map[string]bool, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if value.Kind() == reflect.Slice {
		visit := loggingVisit(value)
		if active[visit] {
			return fmt.Errorf("cyclic config settings at %s (type %v)", path, value.Type())
		}
		active[visit] = true
		defer delete(active, visit)
	}
	for i := range value.Len() {
		if err := debugFlatten(secrets, value.Index(i), debugJoin(path, fmt.Sprint(i)), fmt.Sprintf("%s[%d]", selector, i), entries, active); err != nil {
			return err
		}
	}

	return nil
}

func debugScalar(value reflect.Value, path string) (any, error) {
	switch value.Kind() {
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.String:
		return value.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Type() == reflect.TypeOf(time.Duration(0)) {
			return time.Duration(value.Int()), nil
		}

		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint(), nil
	case reflect.Float32:
		return float32(value.Float()), nil
	case reflect.Float64:
		return value.Float(), nil
	default:
		return nil, fmt.Errorf("unsupported config value at %s (type %v)", path, value.Type())
	}
}

func debugFlattenStruct(secrets map[string]bool, value reflect.Value, path, selector string, entries map[string]debugEntry, active map[debugVisit]bool) error {
	if value.Type() == reflect.TypeOf(time.Time{}) {
		entries[selector] = debugEntry{path, value.Interface()}

		return nil
	}
	for i := range value.NumField() {
		field := value.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			if err := debugFlatten(secrets, value.Field(i), path, selector, entries, active); err != nil {
				return err
			}

			continue
		}
		tag, ok := mapx.ReadStructTag(field.Tag, "cfg")
		if !ok {
			continue
		}
		fieldPath := debugJoin(path, tag.Name)
		fieldSelector := debugJoin(selector, loggingKey(tag.Name))
		if tag.Secret {
			entries[fieldSelector] = debugEntry{fieldPath, "[REDACTED]"}

			continue
		}
		if err := debugFlatten(secrets, value.Field(i), fieldPath, fieldSelector, entries, active); err != nil {
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

var loggingKeyEscaper = strings.NewReplacer("\\", "\\\\", ".", "\\.", "[", "\\[", "]", "\\]")

// loggingKey quotes a literal key only for private logging identities. Display
// paths and the public MapX selector grammar remain unchanged.
func loggingKey(key string) string {
	return loggingKeyEscaper.Replace(key)
}

func loggingSelectorPath(selector string) string {
	path := ""
	for _, segment := range mapx.SplitUnescapedDotN(strings.Trim(selector, "."), -1) {
		name, indexes := segment, ""
		if index := strings.IndexByte(segment, '['); index >= 0 {
			name, indexes = segment[:index], segment[index:]
		}
		path = debugJoin(path, loggingKey(name)+indexes)
	}

	return path
}
