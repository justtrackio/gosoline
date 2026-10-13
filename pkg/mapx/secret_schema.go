package mapx

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// SecretSchema describes tagged fields and dynamic container elements, not values.
// It uses the same field tags and selector syntax as Struct and MapX.
type SecretSchema struct {
	Secret           bool
	Fields           map[string]*SecretSchema
	Element          *SecretSchema
	mapElements      bool
	sequenceElements bool
	typed            *SecretSchema
	concrete         *SecretSchema
	indexes          map[int]*SecretSchema
}

// SecretSchema builds tag metadata from static types and concrete interface
// values, including nil pointers and empty containers. Concrete annotations
// remain scoped to their map key or sequence index.
// forWrite selects Write's prefix mapping; Read preserves the tagged subtree.
func (s *Struct) SecretSchema(forWrite bool) *SecretSchema {
	active := make(map[secretValueVisit]bool)
	value, mapping := secretRootValue(reflect.ValueOf(s.target), active)

	return &SecretSchema{
		typed:    s.secretTypeSchema(value.Type(), forWrite),
		concrete: s.secretValue(value, forWrite, mapping, active, make(map[secretValueType]*SecretSchema)),
	}
}

func (s *Struct) secretTypeSchema(t reflect.Type, forWrite bool) *SecretSchema {
	occurrences := make(map[*SecretSchema]*SecretSchema)
	root := s.secretType(t, make(map[reflect.Type]*SecretSchema), occurrences, forWrite)
	// Recursive container types may not have their element assigned when a
	// tagged occurrence is discovered. Resolve occurrences after the type graph.
	for occurrence, source := range occurrences {
		for occurrences[source] != nil {
			source = occurrences[source]
		}
		occurrence.Element = source.Element
	}

	return root
}

func (s *Struct) secretType(t reflect.Type, seen map[reflect.Type]*SecretSchema, occurrences map[*SecretSchema]*SecretSchema, forWrite bool) *SecretSchema {
	if t == nil {
		return &SecretSchema{}
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if node := seen[t]; node != nil {
		return node
	}
	node := &SecretSchema{Fields: make(map[string]*SecretSchema)}
	seen[t] = node
	switch t.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		node.mapElements = t.Kind() == reflect.Map
		node.sequenceElements = t.Kind() != reflect.Map
		node.Element = s.secretType(t.Elem(), seen, occurrences, forWrite)
	case reflect.Struct:
		s.secretStructType(t, node, seen, occurrences, forWrite)
	}

	return node
}

func (s *Struct) secretStructType(t reflect.Type, node *SecretSchema, seen map[reflect.Type]*SecretSchema, occurrences map[*SecretSchema]*SecretSchema, forWrite bool) {
	if t == reflect.TypeOf(time.Time{}) {
		return
	}
	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		child := s.secretType(field.Type, seen, occurrences, forWrite)
		if field.Anonymous {
			node.Merge(child)

			continue
		}
		tag, ok := s.readTag(field.Tag, field.Name)
		if !ok {
			continue
		}
		s.secretFieldType(field.Type, tag, node, child, occurrences, forWrite)
	}
}

func (s *Struct) secretFieldType(t reflect.Type, tag *StructTag, node, child *SecretSchema, occurrences map[*SecretSchema]*SecretSchema, forWrite bool) {
	// A field annotation belongs to this occurrence, not all uses of its type.
	occurrence := &SecretSchema{Secret: tag.Secret, Fields: child.Fields, Element: child.Element, mapElements: child.mapElements, sequenceElements: child.sequenceElements}
	occurrences[occurrence] = child
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if forWrite && tag.Prefix != "" && t.Kind() == reflect.Struct && t != reflect.TypeOf(time.Time{}) {
		for key, sub := range child.Fields {
			prefixed := &SecretSchema{Secret: tag.Secret || sub.Secret, Fields: sub.Fields, Element: sub.Element, mapElements: sub.mapElements, sequenceElements: sub.sequenceElements}
			occurrences[prefixed] = sub
			node.attach(tag.Prefix+key, prefixed)
		}
	} else {
		node.attach(tag.Name, occurrence)
	}
}

func (s *SecretSchema) attach(selector string, child *SecretSchema) {
	parts := secretSelector(selector)
	if len(parts) == 0 {
		s.Merge(child)

		return
	}
	node := s
	for _, part := range parts[:len(parts)-1] {
		if part.index {
			node.sequenceElements = true
			if node.Element == nil {
				node.Element = &SecretSchema{Fields: make(map[string]*SecretSchema)}
			}
			node = node.Element

			continue
		}
		if node.Fields == nil {
			node.Fields = make(map[string]*SecretSchema)
		}
		if node.Fields[part.key] == nil {
			node.Fields[part.key] = &SecretSchema{Fields: make(map[string]*SecretSchema)}
		}
		node = node.Fields[part.key]
	}
	last := parts[len(parts)-1]
	if last.index {
		node.sequenceElements = true
		if node.Element == nil {
			node.Element = child
		} else {
			node.Element.Merge(child)
		}

		return
	}
	if node.Fields == nil {
		node.Fields = make(map[string]*SecretSchema)
	}
	if existing := node.Fields[last.key]; existing != nil {
		existing.Merge(child)
	} else {
		node.Fields[last.key] = child
	}
}

// Merge combines annotations without removing previously registered secrecy.
func (s *SecretSchema) Merge(other *SecretSchema) {
	s.merge(other, make(map[[2]*SecretSchema]bool), make(map[*SecretSchema]*SecretSchema))
}

func (s *SecretSchema) merge(other *SecretSchema, seen map[[2]*SecretSchema]bool, copies map[*SecretSchema]*SecretSchema) {
	if other == nil || s == other {
		return
	}
	pair := [2]*SecretSchema{s, other}
	if seen[pair] {
		return
	}
	seen[pair] = true
	if copies[other] == nil {
		copies[other] = s
		defer delete(copies, other)
	}
	s.Secret = s.Secret || other.Secret
	s.mapElements = s.mapElements || other.mapElements
	s.sequenceElements = s.sequenceElements || other.sequenceElements
	if s.Fields == nil {
		s.Fields = make(map[string]*SecretSchema)
	}
	for key, child := range other.Fields {
		current := s.Fields[key]
		mergeSecretChild(&current, child, seen, copies)
		s.Fields[key] = current
	}
	s.mergeSecretIndexes(other, seen, copies)
	mergeSecretChild(&s.Element, other.Element, seen, copies)
	mergeSecretChild(&s.typed, other.typed, seen, copies)
	mergeSecretChild(&s.concrete, other.concrete, seen, copies)
}

func mergeSecretChild(target **SecretSchema, source *SecretSchema, seen map[[2]*SecretSchema]bool, copies map[*SecretSchema]*SecretSchema) {
	if source == nil {
		return
	}
	if *target == nil {
		if cloned := copies[source]; cloned != nil {
			*target = cloned

			return
		}
		*target = &SecretSchema{}
	}
	(*target).merge(source, seen, copies)
}

func (s *SecretSchema) mergeSecretIndexes(other *SecretSchema, seen map[[2]*SecretSchema]bool, copies map[*SecretSchema]*SecretSchema) {
	if len(other.indexes) == 0 {
		return
	}
	if s.indexes == nil {
		s.indexes = make(map[int]*SecretSchema)
	}
	for index, child := range other.indexes {
		current := s.indexes[index]
		mergeSecretChild(&current, child, seen, copies)
		s.indexes[index] = current
	}
}

// Register attaches metadata at an explicit rooted selector.
func (s *SecretSchema) Register(selector string, other *SecretSchema) {
	node := s
	for _, part := range secretSelector(selector) {
		if part.index {
			node.sequenceElements = true
			if node.indexes == nil {
				node.indexes = make(map[int]*SecretSchema)
			}
			if node.indexes[part.number] == nil {
				node.indexes[part.number] = &SecretSchema{}
			}
			node = node.indexes[part.number]

			continue
		}
		if node.Fields == nil {
			node.Fields = make(map[string]*SecretSchema)
		}
		if node.Fields[part.key] == nil {
			node.Fields[part.key] = &SecretSchema{}
		}
		node = node.Fields[part.key]
	}
	node.Merge(other)
}

// At returns the schema at a selector, including secrecy inherited from parents.
func (s *SecretSchema) At(selector string) *SecretSchema {
	return s.atParts(secretSelector(selector))
}

func (s *SecretSchema) atParts(parts []secretPart) *SecretSchema {
	result := &SecretSchema{Secret: s.Secret}
	if len(parts) == 0 {
		result.Merge(s)

		return result
	}
	s.mergeAtChild(result, parts[0], parts[1:])
	mergeAtLayer(&result.typed, s.typed, parts)
	mergeAtLayer(&result.concrete, s.concrete, parts)

	return result
}

func mergeAtLayer(target **SecretSchema, source *SecretSchema, parts []secretPart) {
	if source == nil {
		return
	}
	subtree := source.atParts(parts)
	if *target == nil {
		*target = subtree
	} else {
		(*target).Merge(subtree)
	}
}

func (s *SecretSchema) mergeAtChild(result *SecretSchema, part secretPart, tail []secretPart) {
	if part.index {
		if !s.sequenceElements {
			return
		}
		if s.Element != nil {
			result.Merge(s.Element.atParts(tail))
		}
		if child := s.indexes[part.number]; child != nil {
			result.Merge(child.atParts(tail))
		}

		return
	}
	if child := s.Fields[part.key]; child != nil {
		result.Merge(child.atParts(tail))
	}
	if s.mapElements && s.Element != nil {
		result.Merge(s.Element.atParts(tail))
	}
}

// IsSecret includes annotations on every ancestor of the selector. Consecutive
// bracket indexes follow the declared sequence dimensions. At dynamic maps,
// otherwise unsupported bracket suffixes may instead belong to a literal key.
func (s *SecretSchema) IsSecret(selector string) bool {
	return s.isSecret(secretClassificationSelector(selector))
}

func (s *SecretSchema) isSecret(parts []secretPart) bool {
	if s == nil {
		return false
	}
	if s.Secret {
		return true
	}
	if s.typed.isSecret(parts) || s.concrete.isSecret(parts) {
		return true
	}
	if len(parts) == 0 {
		return false
	}
	part := parts[0]
	if part.index {
		if s.sequenceElements {
			return s.Element.isSecret(parts[1:]) || s.indexes[part.number].isSecret(parts[1:])
		}

		return part.root && s.literalMapSecret("", parts)
	}
	child := s.Fields[part.key]
	if child == nil && s.mapElements {
		child = s.Element
	}
	if child.isSecret(parts[1:]) {
		return true
	}

	return s.literalMapSecret(part.key, parts[1:])
}

func (s *SecretSchema) literalMapSecret(key string, parts []secretPart) bool {
	if s.mapElements {
		for i := 0; i < len(parts) && parts[i].index; i++ {
			key += parts[i].key
			if s.Fields[key].isSecret(parts[i+1:]) || s.Element.isSecret(parts[i+1:]) {
				return true
			}
		}
	}

	return false
}

type secretPart struct {
	key    string
	index  bool
	root   bool
	number int
}

func secretSelector(selector string) []secretPart {
	selector = strings.Trim(selector, ".")
	if selector == "" {
		return nil
	}
	var parts []secretPart
	for _, segment := range SplitUnescapedDotN(selector, -1) {
		index, key := getIndex(segment)
		if key != "" || index < 0 {
			parts = append(parts, secretPart{key: key})
		}
		if index >= 0 {
			parts = append(parts, secretPart{index: true, number: index})
		}
	}

	return parts
}

func secretClassificationSelector(selector string) []secretPart {
	selector = strings.Trim(selector, ".")
	if selector == "" {
		return nil
	}
	var parts []secretPart
	for segmentIndex, segment := range SplitUnescapedDotN(selector, -1) {
		key := segment
		var indexes []secretPart
		for {
			// Prefixing permits getIndex to parse a root sequence index too.
			index, prefix := getIndex("_" + key)
			if index < 0 {
				break
			}
			indexes = append(indexes, secretPart{index: true, number: index, key: key[len(prefix)-1:]})
			key = prefix[1:]
		}
		if key != "" || len(indexes) == 0 || segmentIndex > 0 {
			parts = append(parts, secretPart{key: key})
		}
		for i := len(indexes) - 1; i >= 0; i-- {
			part := indexes[i]
			part.root = segmentIndex == 0 && key == "" && i == len(indexes)-1
			parts = append(parts, part)
		}
	}

	return parts
}

type secretValueVisit struct {
	kind    reflect.Kind
	pointer uintptr
}

func secretRootValue(value reflect.Value, active map[secretValueVisit]bool) (reflect.Value, secretValueMapping) {
	// Remove Struct's argument box, but retain the caller's storage carrier
	// when deciding whether map keys are literal or Merge selectors.
	if value.Kind() == reflect.Pointer && !value.IsNil() && value.Type().Elem().Kind() == reflect.Interface {
		value = value.Elem()
		if !value.IsNil() {
			value = value.Elem()
		}
	}
	mapping := secretValueMappingFor(value.Type())
	value = secretRootCarrier(value, active)
	// Root structs are normalized by Struct.Read, including boxed structs.
	if resolved := secretValueMappingFor(value.Type()); resolved.readStructs {
		mapping = resolved
	}

	return value, mapping
}

func secretRootCarrier(value reflect.Value, active map[secretValueVisit]bool) reflect.Value {
	for {
		switch value.Kind() {
		case reflect.Interface:
			if value.IsNil() {
				return value
			}
			value = value.Elem()
		case reflect.Pointer:
			if value.IsNil() || value.Type().Elem().Kind() != reflect.Interface {
				return value
			}
			visit := secretValueVisit{kind: value.Kind(), pointer: value.Pointer()}
			if active[visit] {
				return value
			}
			active[visit] = true
			value = value.Elem()
		default:
			return value
		}
	}
}

type secretValueMapping struct {
	mapSelectors bool
	readStructs  bool
}

type secretValueType struct {
	typeOf   reflect.Type
	forWrite bool
}

func secretValueMappingFor(t reflect.Type) secretValueMapping {
	pointers := 0
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		pointers++
	}
	readStructs := t.Kind() == reflect.Struct && pointers <= 1

	return secretValueMapping{
		mapSelectors: readStructs || (pointers == 0 && t == reflect.TypeOf(map[string]any{})),
		readStructs:  readStructs,
	}
}

func (s *Struct) secretValue(value reflect.Value, forWrite bool, mapping secretValueMapping, active map[secretValueVisit]bool, types map[secretValueType]*SecretSchema) *SecretSchema {
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return nil
		}
		actual := value.Elem()
		t := actual.Type()
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Interface:
			// Pointer/interface carriers retain opaque storage semantics. Resolve
			// their payload through the same cycle-guarded value traversal.
			return s.secretValue(actual, forWrite, secretValueMapping{}, active, types)
		case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
			// Interface-held structs/pointers are opaque leaves unless Struct.Read
			// explicitly normalizes this container's concrete struct elements.
			if actual.Kind() == reflect.Pointer || (actual.Kind() == reflect.Struct && !mapping.readStructs) {
				mapping = secretValueMapping{}
			}
			key := secretValueType{typeOf: actual.Type(), forWrite: forWrite && mapping.readStructs}
			if types[key] == nil {
				types[key] = s.secretTypeSchema(actual.Type(), key.forWrite)
			}

			return &SecretSchema{typed: types[key], concrete: s.secretValue(actual, forWrite, mapping, active, types)}
		default:
			return nil
		}
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		visit := secretValueVisit{value.Kind(), value.Pointer()}
		if active[visit] {
			return nil
		}
		active[visit] = true
		defer delete(active, visit)

		return s.secretValue(value.Elem(), forWrite, mapping, active, types)
	case reflect.Map:
		return s.secretMapValue(value, forWrite, mapping, active, types)
	case reflect.Slice, reflect.Array:
		return s.secretSliceValue(value, forWrite, mapping, active, types)
	case reflect.Struct:
		return s.secretStructValue(value, forWrite, mapping, active, types)
	default:
		return nil
	}
}

func (s *Struct) secretMapValue(value reflect.Value, forWrite bool, mapping secretValueMapping, active map[secretValueVisit]bool, types map[secretValueType]*SecretSchema) *SecretSchema {
	visit := secretValueVisit{value.Kind(), uintptr(value.UnsafePointer())}
	if active[visit] {
		return nil
	}
	active[visit] = true
	defer delete(active, visit)
	var node *SecretSchema
	iter := value.MapRange()
	childMapping := mapping
	if value.Type() == reflect.TypeOf(map[string]any{}) {
		childMapping.readStructs = false
	}
	for iter.Next() {
		child := s.secretValue(iter.Value(), forWrite, childMapping, active, types)
		if child == nil {
			continue
		}
		if node == nil {
			node = &SecretSchema{mapElements: true, Fields: make(map[string]*SecretSchema)}
		}
		key := fmt.Sprint(iter.Key().Interface())
		if mapping.mapSelectors {
			node.attachConcrete(key, child)
		} else {
			key = strings.Join(SplitUnescapedDotN(key, -1), "\\.")
			node.attachConcreteField(key, child)
		}
	}

	return node
}

func (s *Struct) secretSliceValue(value reflect.Value, forWrite bool, mapping secretValueMapping, active map[secretValueVisit]bool, types map[secretValueType]*SecretSchema) *SecretSchema {
	if value.Kind() == reflect.Slice {
		visit := secretValueVisit{value.Kind(), value.Pointer()}
		if active[visit] {
			return nil
		}
		active[visit] = true
		defer delete(active, visit)
	}
	var node *SecretSchema
	for i := range value.Len() {
		childMapping := mapping
		if value.Index(i).Kind() == reflect.Interface {
			childMapping.readStructs = false
		}
		child := s.secretValue(value.Index(i), forWrite, childMapping, active, types)
		if child == nil {
			continue
		}
		if node == nil {
			node = &SecretSchema{sequenceElements: true, indexes: make(map[int]*SecretSchema)}
		}
		node.indexes[i] = child
	}

	return node
}

func (s *Struct) secretStructValue(value reflect.Value, forWrite bool, mapping secretValueMapping, active map[secretValueVisit]bool, types map[secretValueType]*SecretSchema) *SecretSchema {
	if value.Type() == reflect.TypeOf(time.Time{}) {
		return nil
	}
	var node *SecretSchema
	for i := range value.NumField() {
		field := value.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag, ok := s.readTag(field.Tag, field.Name)
		if !field.Anonymous && (!ok || tag.Secret) {
			continue
		}
		childMapping := mapping
		if !mapping.readStructs {
			childMapping.mapSelectors = false
		}
		if field.Type.Kind() == reflect.Interface {
			childMapping.readStructs = false
		}
		child := s.secretValue(value.Field(i), forWrite, childMapping, active, types)
		if child == nil {
			continue
		}
		if node == nil {
			node = &SecretSchema{Fields: make(map[string]*SecretSchema)}
		}
		if field.Anonymous {
			node.Merge(child)

			continue
		}
		s.attachConcreteStructField(node, field.Type, tag, child, forWrite)
	}

	return node
}

func (s *Struct) attachConcreteStructField(node *SecretSchema, t reflect.Type, tag *StructTag, child *SecretSchema, forWrite bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if forWrite && tag.Prefix != "" && t.Kind() == reflect.Struct && t != reflect.TypeOf(time.Time{}) {
		for key, sub := range child.Fields {
			node.attachConcrete(tag.Prefix+key, sub)
		}
	} else {
		node.attachConcrete(tag.Name, child)
	}
}

func (s *SecretSchema) attachConcrete(selector string, child *SecretSchema) {
	parts := secretSelector(selector)
	if len(parts) == 0 {
		s.Merge(child)

		return
	}
	node := s
	for _, part := range parts[:len(parts)-1] {
		node = node.concreteChild(part)
	}
	last := parts[len(parts)-1]
	if last.index {
		node.sequenceElements = true
		if node.indexes == nil {
			node.indexes = make(map[int]*SecretSchema)
		}
		combined := &SecretSchema{}
		combined.Merge(node.indexes[last.number])
		combined.Merge(child)
		node.indexes[last.number] = combined
	} else {
		node.attachConcreteField(last.key, child)
	}
}

func (s *SecretSchema) concreteChild(part secretPart) *SecretSchema {
	if part.index {
		s.sequenceElements = true
		if s.indexes == nil {
			s.indexes = make(map[int]*SecretSchema)
		}
		if s.indexes[part.number] == nil {
			s.indexes[part.number] = &SecretSchema{}
		}

		return s.indexes[part.number]
	}
	if s.Fields == nil {
		s.Fields = make(map[string]*SecretSchema)
	}
	if s.Fields[part.key] == nil {
		s.Fields[part.key] = &SecretSchema{}
	}

	return s.Fields[part.key]
}

func (s *SecretSchema) attachConcreteField(key string, child *SecretSchema) {
	if s.Fields == nil {
		s.Fields = make(map[string]*SecretSchema)
	}
	if s.Fields[key] == nil {
		s.Fields[key] = child

		return
	}
	combined := &SecretSchema{}
	combined.Merge(s.Fields[key])
	combined.Merge(child)
	s.Fields[key] = combined
}
