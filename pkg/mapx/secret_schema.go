package mapx

import (
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
}

// SecretSchema builds type metadata, including nil pointers and empty containers.
// forWrite selects Write's prefix mapping; Read preserves the tagged subtree.
func (s *Struct) SecretSchema(forWrite bool) *SecretSchema {
	occurrences := make(map[*SecretSchema]*SecretSchema)
	root := s.secretType(reflect.TypeOf(s.target), make(map[reflect.Type]*SecretSchema), occurrences, forWrite)
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
		if s.Fields[key] == nil {
			if cloned := copies[child]; cloned != nil {
				s.Fields[key] = cloned

				continue
			}
			s.Fields[key] = &SecretSchema{}
		}
		s.Fields[key].merge(child, seen, copies)
	}
	if other.Element != nil {
		if s.Element == nil {
			if cloned := copies[other.Element]; cloned != nil {
				s.Element = cloned

				return
			}
			s.Element = &SecretSchema{}
		}
		s.Element.merge(other.Element, seen, copies)
	}
}

// Register attaches metadata at an explicit rooted selector.
func (s *SecretSchema) Register(selector string, other *SecretSchema) {
	node := s
	for _, part := range secretSelector(selector) {
		if part.index {
			node.sequenceElements = true
			if node.Element == nil {
				node.Element = &SecretSchema{}
			}
			node = node.Element

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
	node, secret := s.lookup(selector)
	if node == nil {
		return &SecretSchema{Secret: secret}
	}

	return &SecretSchema{Secret: secret, Fields: node.Fields, Element: node.Element, mapElements: node.mapElements, sequenceElements: node.sequenceElements}
}

func (s *SecretSchema) lookup(selector string) (*SecretSchema, bool) {
	node := s
	secret := node.Secret
	for _, part := range secretSelector(selector) {
		if part.index {
			if !node.sequenceElements {
				return nil, secret
			}
			node = node.Element
		} else {
			child := node.Fields[part.key]
			if child == nil && node.mapElements {
				child = node.Element
			}
			node = child
		}
		if node == nil {
			return nil, secret
		}
		secret = secret || node.Secret
	}

	return node, secret
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
	if len(parts) == 0 {
		return false
	}
	part := parts[0]
	if part.index {
		if s.sequenceElements {
			return s.Element.isSecret(parts[1:])
		}

		return part.root && s.literalMapSecret(parts)
	}
	child := s.Fields[part.key]
	if child == nil && s.mapElements {
		child = s.Element
	}
	if child.isSecret(parts[1:]) {
		return true
	}

	return s.literalMapSecret(parts[1:])
}

func (s *SecretSchema) literalMapSecret(parts []secretPart) bool {
	if s.mapElements {
		for i := 0; i < len(parts) && parts[i].index; i++ {
			if s.Element.isSecret(parts[i+1:]) {
				return true
			}
		}
	}

	return false
}

type secretPart struct {
	key   string
	index bool
	root  bool
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
			parts = append(parts, secretPart{index: true})
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
		indexes := 0
		for {
			// Prefixing permits getIndex to parse a root sequence index too.
			index, prefix := getIndex("_" + key)
			if index < 0 {
				break
			}
			key = prefix[1:]
			indexes++
		}
		if key != "" || indexes == 0 || segmentIndex > 0 {
			parts = append(parts, secretPart{key: key})
		}
		for i := range indexes {
			parts = append(parts, secretPart{index: true, root: segmentIndex == 0 && key == "" && i == 0})
		}
	}

	return parts
}
