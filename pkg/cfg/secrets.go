package cfg

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/mapx"
)

// logProjection retains only the annotated shape of a rooted registration.
// It is applied to present settings when logging, not queried as a schema.
type logProjection struct {
	key         string
	typ         reflect.Type
	write       bool
	privatePath bool
	secret      bool
	err         error
}

var configDebugTypes struct {
	sync.RWMutex
	projections []logProjection
}

// RegisterConfigDebugType registers a rooted configuration type for DebugConfig
// to redact its cfg:",secret" fields, including settings that were never decoded.
// Registration supplies logging metadata only; it does not load or validate
// settings. SDK owners register their types when imported, so types from packages
// not imported by the application are not automatically available.
// Named collections support literal keys and selector-normalized namespaces.
// Logging fails before emitting settings if a subtree admits both a model and
// a nested named model with secret fields, rather than guessing ownership.
// Repeated registrations of the same key and type are ignored, as are nil types.
func RegisterConfigDebugType(key string, typ reflect.Type) {
	if typ == nil {
		return
	}
	key = strings.Trim(key, ".")
	configDebugTypes.Lock()
	defer configDebugTypes.Unlock()
	for _, projection := range configDebugTypes.projections {
		if projection.key == key && projection.typ == typ {
			return
		}
	}
	configDebugTypes.projections = append(configDebugTypes.projections, logProjection{key: key, typ: typ, write: true})
}

func (c *config) registerSecrets(key string, value any, forWrite bool) {
	c.registerLogProjection(logProjection{key: strings.Trim(key, "."), typ: reflect.TypeOf(value), write: forWrite})
	v := loggingIndirect(reflect.ValueOf(value), make(map[uintptr]bool))
	// Only struct reads normalize annotated nested container elements. Ordinary
	// map/value merges retain their structs and are handled live by DebugConfig.
	if v.IsValid() && v.Kind() == reflect.Struct {
		c.captureLogTypes(loggingSelectorPath(key), v, forWrite, make(map[debugVisit]bool))
	}
}

func (c *config) registerLogProjection(projection logProjection) {
	c.secretMu.Lock()
	defer c.secretMu.Unlock()
	c.logProjections = append(c.logProjections, projection)
}

func loggingVisit(value reflect.Value) debugVisit {
	switch value.Kind() {
	case reflect.Pointer, reflect.Slice:
		return debugVisit{kind: value.Kind(), pointer: value.Pointer()}
	case reflect.Map:
		return debugVisit{kind: value.Kind(), pointer: uintptr(value.UnsafePointer())}
	default:
		return debugVisit{}
	}
}

func (c *config) captureLogTypes(path string, value reflect.Value, write bool, active map[debugVisit]bool) {
	if !value.IsValid() {
		return
	}
	visit := loggingVisit(value)
	if visit.pointer != 0 {
		if active[visit] {
			return
		}
		active[visit] = true
		defer delete(active, visit)
	}
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !value.IsNil() {
			c.captureLogTypes(path, value.Elem(), write, active)
		}
	case reflect.Struct:
		c.captureLogStruct(path, value, write, active)
	case reflect.Map:
		c.captureLogMap(path, value, write, active)
	case reflect.Slice, reflect.Array:
		for i := range value.Len() {
			c.captureLogElement(fmt.Sprintf("%s[%d]", path, i), value.Index(i), write, active)
		}
	}
}

func (c *config) captureLogStruct(path string, value reflect.Value, write bool, active map[debugVisit]bool) {
	if value.Type() == reflect.TypeOf(time.Time{}) {
		return
	}
	for i := range value.NumField() {
		field := value.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			c.captureLogTypes(path, value.Field(i), write, active)

			continue
		}
		tag, ok := mapx.ReadStructTag(field.Tag, "cfg")
		if !ok || tag.Secret {
			continue
		}
		c.captureLogTypes(debugJoin(path, loggingSelectorPath(tag.Name)), value.Field(i), write, active)
	}
}

func (c *config) captureLogMap(path string, value reflect.Value, write bool, active map[debugVisit]bool) {
	if value.Type().Key().Kind() != reflect.String {
		return
	}
	iter := value.MapRange()
	for iter.Next() {
		key := loggingKey(iter.Key().String())
		if !write {
			// Struct.Read builds selector paths from map keys, unlike raw
			// retained maps and existing source data used in write mode.
			key = loggingSelectorPath(iter.Key().String())
		}
		c.captureLogElement(debugJoin(path, key), iter.Value(), write, active)
	}
}

func (c *config) captureLogElement(path string, value reflect.Value, write bool, active map[debugVisit]bool) {
	for value.Kind() == reflect.Interface && !value.IsNil() {
		value = value.Elem()
	}
	c.registerLogProjection(logProjection{key: path, typ: value.Type(), write: write, privatePath: true})
	c.captureLogTypes(path, value, write, active)
}

type defaultsConfig struct {
	GosoConf
	owner *config
	typ   reflect.Type
	value reflect.Value
}

func (c *config) defaultsContext(output any) Config {
	return &defaultsConfig{GosoConf: c, owner: c, typ: reflect.TypeOf(output), value: reflect.ValueOf(output)}
}

// Metadata must not change default loading. An absent destination field has no
// annotations to transfer; an unresolved interface is rejected only by logging.
func (c *defaultsConfig) registerSource(source, target string) {
	typ, value, secret := c.typ, c.value, false
	for _, name := range mapx.SplitUnescapedDotN(strings.Trim(target, "."), -1) {
		if name == "" {
			continue
		}
		var found bool
		typ, value, secret, found = projectionTarget(typ, value, name)
		if !found {
			if typ != nil && typ.Kind() == reflect.Interface {
				c.unsupportedSource(source, target, typ)
			}

			return
		}
		if secret {
			break
		}
	}
	if !secret {
		typ, value = projectionModel(typ, value)
		if typ != nil && typ.Kind() == reflect.Interface {
			c.unsupportedSource(source, target, typ)

			return
		}
		c.owner.captureLogTypes(loggingSelectorPath(source), value, true, make(map[debugVisit]bool))
	}
	c.owner.registerLogProjection(logProjection{key: strings.Trim(source, "."), typ: typ, write: true, secret: secret})
}

func (c *defaultsConfig) unsupportedSource(source, target string, typ reflect.Type) {
	err := fmt.Errorf("unsupported config logging projection at %s (type %v)", target, typ)
	c.owner.registerLogProjection(logProjection{key: strings.Trim(source, "."), err: err})
}

func projectionField(typ reflect.Type, name string) (reflect.Type, bool, bool) {
	child, _, secret, found := projectionTarget(typ, reflect.Value{}, name)

	return child, secret, found
}

func (c *config) loggingSecrets(settings map[string]any) (map[string]bool, error) {
	c.secretMu.RLock()
	roots := append([]logProjection(nil), c.logProjections...)
	c.secretMu.RUnlock()
	configDebugTypes.RLock()
	roots = append(roots, configDebugTypes.projections...)
	configDebugTypes.RUnlock()
	sort.SliceStable(roots, func(i, j int) bool { return len(roots[i].key) < len(roots[j].key) })
	secrets := make(map[string]bool)
	for _, root := range roots {
		path := root.key
		if !root.privatePath {
			path = loggingSelectorPath(root.key)
		}
		if loggingMasked(secrets, path) {
			continue
		}
		value := reflect.ValueOf(settings)
		if root.key != "" {
			if root.privatePath {
				value = loggingSelectPath(value, root.key)
			} else {
				value = loggingSelect(value, root.key)
			}
		}
		if !value.IsValid() {
			continue
		}
		if root.err != nil {
			return nil, root.err
		}
		if root.secret {
			secrets[path] = true

			continue
		}
		if err := projectLoggingSecrets(root.typ, value, path, root.write, secrets, make(map[debugVisit]bool)); err != nil {
			return nil, err
		}
	}

	return secrets, nil
}

func loggingMasked(secrets map[string]bool, path string) bool {
	for parent := range secrets {
		if path == parent || strings.HasPrefix(path, parent+".") || strings.HasPrefix(path, parent+"[") {
			return true
		}
	}

	return false
}

func loggingProjectionVisit(typ reflect.Type, value reflect.Value) debugVisit {
	// Different anonymous types can inspect the same current map. Repeating
	// the same type/map pair is recursion, not another meaningful projection.
	if typ.Kind() == reflect.Struct && value.Kind() == reflect.Map {
		return debugVisit{kind: value.Kind(), pointer: uintptr(value.UnsafePointer()), typ: typ}
	}

	return loggingVisit(value)
}

func projectLoggingSecrets(typ reflect.Type, value reflect.Value, path string, write bool, secrets map[string]bool, active map[debugVisit]bool) error {
	if typ == nil || !value.IsValid() || secrets[path] {
		return nil
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == reflect.TypeOf(time.Time{}) {
		return nil
	}
	visit := loggingProjectionVisit(typ, value)
	if visit.pointer != 0 {
		if active[visit] {
			return fmt.Errorf("cyclic config logging projection at %s (type %v)", path, value.Type())
		}
		active[visit] = true
		defer delete(active, visit)
	}
	if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}

		return projectLoggingSecrets(typ, value.Elem(), path, write, secrets, active)
	}
	// Retained structs enforce their current tags during the logging traversal.
	if value.Kind() == reflect.Struct || typ.Kind() == reflect.Interface {
		return nil
	}
	switch typ.Kind() {
	case reflect.Struct:
		return projectLogStruct(typ, value, path, write, secrets, active)
	case reflect.Map:
		return projectLogMap(typ, value, path, write, secrets, active)
	case reflect.Array, reflect.Slice:
		return projectLogSequence(typ, value, path, write, secrets, active)
	default:
		return nil
	}
}

func projectLogStruct(typ reflect.Type, value reflect.Value, path string, write bool, secrets map[string]bool, active map[debugVisit]bool) error {
	if value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
		return unsupportedLogProjection(path, value.Type())
	}
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			if err := projectLoggingSecrets(field.Type, value, path, write, secrets, active); err != nil {
				return err
			}

			continue
		}
		tag, ok := mapx.ReadStructTag(field.Tag, "cfg")
		if !ok {
			continue
		}
		if err := projectLogChild(field.Type, loggingSelect(value, tag.Name), debugJoin(path, loggingSelectorPath(tag.Name)), write, tag.Secret, secrets, active); err != nil {
			return err
		}
		if err := projectLogPrefix(field.Type, value, path, write, tag, secrets, active); err != nil {
			return err
		}
	}

	return nil
}

func projectLogChild(typ reflect.Type, value reflect.Value, path string, write, secret bool, secrets map[string]bool, active map[debugVisit]bool) error {
	if !value.IsValid() {
		return nil
	}
	if secret {
		secrets[path] = true

		return nil
	}

	return projectLoggingSecrets(typ, value, path, write, secrets, active)
}

func projectLogPrefix(typ reflect.Type, value reflect.Value, path string, write bool, tag *mapx.StructTag, secrets map[string]bool, active map[debugVisit]bool) error {
	fieldType := typ
	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	if !write || tag.Prefix == "" || fieldType.Kind() != reflect.Struct {
		return nil
	}
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		if !strings.HasPrefix(key, tag.Prefix) {
			continue
		}
		childPath := debugJoin(path, loggingKey(key))
		if tag.Secret {
			secrets[childPath] = true

			continue
		}
		childType, childSecret, found := projectionField(typ, strings.TrimPrefix(key, tag.Prefix))
		if !found {
			return unsupportedLogProjection(childPath, typ)
		}
		if err := projectLogChild(childType, iter.Value(), childPath, write, childSecret, secrets, active); err != nil {
			return err
		}
	}

	return nil
}

func projectLogMap(typ reflect.Type, value reflect.Value, path string, write bool, secrets map[string]bool, active map[debugVisit]bool) error {
	if typ.Elem().Kind() == reflect.Interface {
		return nil
	}
	if value.Kind() != reflect.Map || typ.Key().Kind() != reflect.String || value.Type().Key().Kind() != reflect.String {
		return unsupportedLogProjection(path, value.Type())
	}
	iter := value.MapRange()
	for iter.Next() {
		if err := projectLogCollectionMember(typ, iter.Value(), debugJoin(path, loggingKey(iter.Key().String())), write, secrets, active); err != nil {
			return err
		}
	}

	return nil
}

func projectLogCollectionMember(typ reflect.Type, value reflect.Value, path string, write bool, secrets map[string]bool, active map[debugVisit]bool) error {
	childType := typ.Elem()
	modelType := childType
	for modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem()
	}
	current := loggingIndirect(value, make(map[uintptr]bool))
	if modelType.Kind() != reflect.Struct || !current.IsValid() || current.Kind() != reflect.Map || current.Type().Key().Kind() != reflect.String {
		return projectLoggingSecrets(childType, value, path, write, secrets, active)
	}
	// Merges expand selector names into maps. Descend only through maps with
	// no model fields; actual public fields also establish a model boundary.
	if !loggingModelBoundary(modelType, current, write, active) {
		return projectLogNamespace(typ, current, path, write, secrets, active)
	}
	if err := projectLoggingSecrets(childType, value, path, write, secrets, active); err != nil {
		return err
	}
	// A model-field segment can also occur in an accepted selector-based name.
	// Never guess between that namespace and a current model's raw metadata.
	if loggingNestedModelSecrets(modelType, current, path, write, false, secrets, active) {
		return fmt.Errorf("ambiguous config logging collection at %s (type %v)", path, modelType)
	}

	return nil
}

func projectLogNamespace(typ reflect.Type, value reflect.Value, path string, write bool, secrets map[string]bool, active map[debugVisit]bool) error {
	visit := loggingVisit(value)
	if active[visit] {
		return fmt.Errorf("cyclic config logging namespace at %s (type %v)", path, value.Type())
	}
	active[visit] = true
	defer delete(active, visit)
	iter := value.MapRange()
	for iter.Next() {
		child := loggingIndirect(iter.Value(), make(map[uintptr]bool))
		// No model boundary owns namespace scalars. The normal logging
		// traversal keeps them public and rejects unsupported public values.
		if !child.IsValid() || child.Kind() != reflect.Map {
			continue
		}
		if err := projectLogCollectionMember(typ, iter.Value(), debugJoin(path, loggingKey(iter.Key().String())), write, secrets, active); err != nil {
			return err
		}
	}

	return nil
}

func loggingNestedModelSecrets(typ reflect.Type, value reflect.Value, path string, write, probe bool, secrets map[string]bool, active map[debugVisit]bool) bool {
	if loggingMasked(secrets, path) {
		return false
	}
	value = loggingIndirect(value, make(map[uintptr]bool))
	if !value.IsValid() || value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
		return false
	}
	visit := loggingVisit(value)
	if active[visit] {
		return true
	}
	active[visit] = true
	defer delete(active, visit)
	if probe && loggingModelBoundary(typ, value, write, active) {
		// Reuse current masks and inspect only potential model boundaries.
		before := len(secrets)
		err := projectLoggingSecrets(typ, value, path, write, secrets, active)
		if len(secrets) != before {
			return true
		}
		if err != nil {
			// Scalar-only public metadata can resemble model field names
			// without accepting the model. Structured rejected candidates
			// or present direct annotations remain ambiguous and fail safely.
			return loggingRejectedModelAmbiguous(typ, value, path, write, secrets, active)
		}
	}
	iter := value.MapRange()
	for iter.Next() {
		if loggingNestedModelSecrets(typ, iter.Value(), debugJoin(path, loggingKey(iter.Key().String())), write, true, secrets, active) {
			return true
		}
	}

	return false
}

func loggingRejectedModelAmbiguous(typ reflect.Type, value reflect.Value, path string, write bool, secrets map[string]bool, active map[debugVisit]bool) bool {
	if loggingDirectModelSecret(typ, value, write, active) {
		return true
	}
	iter := value.MapRange()
	for iter.Next() {
		if loggingMasked(secrets, debugJoin(path, loggingKey(iter.Key().String()))) {
			continue
		}
		child := loggingIndirect(iter.Value(), make(map[uintptr]bool))
		if child.IsValid() && (child.Kind() == reflect.Map || child.Kind() == reflect.Struct || child.Kind() == reflect.Slice || child.Kind() == reflect.Array) {
			return true
		}
	}

	return false
}

func loggingDirectModelSecret(typ reflect.Type, value reflect.Value, write bool, active map[debugVisit]bool) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	visit := debugVisit{kind: reflect.Struct, typ: typ}
	if typ.Kind() != reflect.Struct || active[visit] {
		return false
	}
	active[visit] = true
	defer delete(active, visit)
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			if loggingDirectModelSecret(field.Type, value, write, active) {
				return true
			}

			continue
		}
		tag, ok := mapx.ReadStructTag(field.Tag, "cfg")
		if ok && tag.Secret && (loggingSelect(value, tag.Name).IsValid() || loggingModelPrefix(value, tag, write)) {
			return true
		}
	}

	return false
}

func loggingModelBoundary(typ reflect.Type, value reflect.Value, write bool, active map[debugVisit]bool) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	visit := debugVisit{typ: typ}
	if active[visit] {
		return false
	}
	active[visit] = true
	defer delete(active, visit)
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			if loggingModelBoundary(field.Type, value, write, active) {
				return true
			}

			continue
		}
		tag, ok := mapx.ReadStructTag(field.Tag, "cfg")
		if !ok {
			continue
		}
		if loggingSelect(value, tag.Name).IsValid() {
			return true
		}
		if loggingModelPrefix(value, tag, write) {
			return true
		}
	}

	return false
}

func loggingModelPrefix(value reflect.Value, tag *mapx.StructTag, write bool) bool {
	if !write || tag.Prefix == "" {
		return false
	}
	iter := value.MapRange()
	for iter.Next() {
		if strings.HasPrefix(iter.Key().String(), tag.Prefix) {
			return true
		}
	}

	return false
}

func projectLogSequence(typ reflect.Type, value reflect.Value, path string, write bool, secrets map[string]bool, active map[debugVisit]bool) error {
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		// Decoding may accept a raw scalar (for example a comma-separated string).
		// Scalar element types have no child annotations to project. Composite
		// elements must still fail safely rather than expose encoded credentials.
		if loggingScalarElement(typ.Elem()) {
			return nil
		}

		return unsupportedLogProjection(path, value.Type())
	}
	for i := range value.Len() {
		if err := projectLoggingSecrets(typ.Elem(), value.Index(i), fmt.Sprintf("%s[%d]", path, i), write, secrets, active); err != nil {
			return err
		}
	}

	return nil
}

func loggingScalarElement(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return typ == reflect.TypeOf(time.Time{})
	}
}

func unsupportedLogProjection(path string, typ reflect.Type) error {
	return fmt.Errorf("unsupported config logging projection at %s (type %v)", path, typ)
}

func loggingMapValue(value reflect.Value, key string) reflect.Value {
	mapKey := reflect.New(value.Type().Key()).Elem()
	mapKey.SetString(key)

	return value.MapIndex(mapKey)
}

// Follow MapX's escaped-dot and bracket-index spelling without copying private
// containers into another MapX. Public cycles are rejected by debug traversal.
func loggingSelect(value reflect.Value, selector string) reflect.Value {
	seen := make(map[uintptr]bool)
	for _, segment := range mapx.SplitUnescapedDotN(strings.Trim(selector, "."), -1) {
		value = loggingIndirect(value, seen)
		if !value.IsValid() || value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
			return reflect.Value{}
		}
		name, indexes := segment, ""
		if index := strings.IndexByte(segment, '['); index >= 0 {
			name, indexes = segment[:index], segment[index:]
		}
		value = loggingIndexes(loggingMapValue(value, name), indexes, seen)
	}

	return value
}

// Captured literal-map paths use private quoting, never public selectors.
func loggingSelectPath(value reflect.Value, path string) reflect.Value {
	seen := make(map[uintptr]bool)
	for _, segment := range mapx.SplitUnescapedDotN(path, -1) {
		value = loggingIndirect(value, seen)
		if !value.IsValid() || value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
			return reflect.Value{}
		}
		name, indexes := loggingPathSegment(segment)
		value = loggingIndexes(loggingMapValue(value, name), indexes, seen)
	}

	return value
}

func loggingPathSegment(segment string) (name, indexes string) {
	var key strings.Builder
	for i := 0; i < len(segment); i++ {
		switch segment[i] {
		case '\\':
			if i+1 < len(segment) {
				i++
			}
			key.WriteByte(segment[i])
		case '[':
			return key.String(), segment[i:]
		default:
			key.WriteByte(segment[i])
		}
	}

	return key.String(), ""
}

func loggingIndirect(value reflect.Value, seen map[uintptr]bool) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return reflect.Value{}
		}
		if value.Kind() == reflect.Pointer {
			if seen[value.Pointer()] {
				return reflect.Value{}
			}
			seen[value.Pointer()] = true
		}
		value = value.Elem()
	}

	return value
}

func loggingIndexes(value reflect.Value, indexes string, seen map[uintptr]bool) reflect.Value {
	for indexes != "" {
		end := strings.IndexByte(indexes, ']')
		if indexes[0] != '[' || end < 0 {
			return reflect.Value{}
		}
		index, err := strconv.Atoi(indexes[1:end])
		if err != nil || index < 0 {
			return reflect.Value{}
		}
		value = loggingIndirect(value, seen)
		if !value.IsValid() || (value.Kind() != reflect.Array && value.Kind() != reflect.Slice) || index >= value.Len() {
			return reflect.Value{}
		}
		value = value.Index(index)
		indexes = indexes[end+1:]
	}

	return value
}

func projectionModel(typ reflect.Type, value reflect.Value) (reflect.Type, reflect.Value) {
	for typ != nil && (typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Interface) {
		switch {
		case value.IsValid() && !value.IsNil():
			value = value.Elem()
			typ = value.Type()
		case typ.Kind() == reflect.Pointer:
			typ, value = typ.Elem(), reflect.Value{}
		default:
			return typ, reflect.Value{}
		}
	}

	return typ, value
}

func projectionTarget(typ reflect.Type, value reflect.Value, name string) (reflect.Type, reflect.Value, bool, bool) {
	typ, value = projectionModel(typ, value)
	if typ != nil && typ.Kind() == reflect.Interface {
		return typ, reflect.Value{}, false, false
	}
	if typ == nil || typ.Kind() != reflect.Struct {
		return nil, reflect.Value{}, false, false
	}
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		child := reflect.Value{}
		if value.IsValid() {
			child = value.Field(i)
		}
		if field.Anonymous {
			if childType, childValue, secret, ok := projectionTarget(field.Type, child, name); ok {
				return childType, childValue, secret, true
			}
		}
		tag, ok := mapx.ReadStructTag(field.Tag, "cfg")
		if ok && tag.Name == name {
			return field.Type, child, tag.Secret, true
		}
	}

	return nil, reflect.Value{}, false, false
}
