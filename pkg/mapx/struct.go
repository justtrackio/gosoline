package mapx

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/spf13/cast"
)

const (
	optionNoCast   = "nocast"
	optionNoDecode = "nodecode"
)

type StructKey struct {
	Parent  string
	Key     string
	Kind    reflect.Kind
	SubKeys []StructKey
}

type StructTag struct {
	Name     string
	NoCast   bool
	NoDecode bool
}

func (k StructKey) String() string {
	return k.Key
}

type StructSettings struct {
	FieldTag   string
	DefaultTag string
	Casters    []MapStructCaster
	Decoders   []MapStructDecoder
}

type Struct struct {
	target   any
	casters  []MapStructCaster
	decoders []MapStructDecoder
	settings *StructSettings
}

func NewStruct(source any, settings *StructSettings) (*Struct, error) {
	st := reflect.TypeOf(source)

	if st.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("the target value has to be a pointer")
	}

	return &Struct{
		target:   source,
		casters:  append([]MapStructCaster{}, settings.Casters...),
		decoders: append([]MapStructDecoder{}, settings.Decoders...),
		settings: settings,
	}, nil
}

func (s *Struct) Keys() []StructKey {
	sv := reflect.ValueOf(s.target).Elem().Interface()

	return s.doKeys("", sv)
}

func (s *Struct) doKeys(parent string, target any) []StructKey {
	st := reflect.TypeOf(target)
	sv := reflect.ValueOf(target)

	var ok bool
	var tag *StructTag
	var keys []StructKey

	for i := 0; i < st.NumField(); i++ {
		targetField := st.Field(i)
		targetValue := sv.Field(i)

		// skip unexported fields
		if targetField.PkgPath != "" {
			continue
		}

		if targetField.Type.Kind() == reflect.Struct && targetField.Anonymous {
			embeddedValue := targetValue.Interface()
			embeddedKeys := s.doKeys("", embeddedValue)

			keys = append(keys, embeddedKeys...)

			continue
		}

		if tag, ok = s.readTag(targetField.Tag); !ok {
			continue
		}

		if targetField.Type.Kind() == reflect.Struct && targetField.Type != reflect.TypeOf(time.Time{}) {
			continue
		}

		if targetField.Type.Kind() == reflect.Slice {
			slValue := reflect.MakeSlice(targetField.Type, 1, 1).Index(0)

			if slValue.Kind() == reflect.Struct && slValue.Type() != reflect.TypeOf(time.Time{}) {
				slInterface := slValue.Interface()
				slKeys := s.doKeys(tag.Name, slInterface)

				keys = append(keys, StructKey{
					Parent:  parent,
					Key:     tag.Name,
					Kind:    targetField.Type.Kind(),
					SubKeys: slKeys,
				})

				continue
			}
		}

		keys = append(keys, StructKey{
			Parent: parent,
			Key:    tag.Name,
			Kind:   targetField.Type.Kind(),
		})
	}

	return keys
}

func (s *Struct) ReadZeroAndDefaultValues() (zeros *MapX, defaults *MapX, err error) {
	sv := reflect.ValueOf(s.target).Elem().Interface()

	return s.doReadZeroAndDefaultValues(sv)
}

//nolint:gocognit // trying to split it up made it even harder to read
func (s *Struct) doReadZeroAndDefaultValues(target any) (zeros *MapX, defaults *MapX, err error) {
	st := reflect.TypeOf(target)
	sv := reflect.ValueOf(target)

	var val string
	var tag *StructTag
	var ok bool
	var zeroValue, defValue any
	zeros, defaults = NewMapX(), NewMapX()

	for i := 0; i < st.NumField(); i++ {
		targetField := st.Field(i)
		targetValue := sv.Field(i)

		// skip unexported fields
		if targetField.PkgPath != "" {
			continue
		}

		if targetField.Type.Kind() == reflect.Struct && targetField.Anonymous {
			embeddedZeros, embeddedDefaults, err := s.doReadZeroAndDefaultValues(targetValue.Interface())
			if err != nil {
				return nil, nil, fmt.Errorf("can not read from embedded field %s", targetField.Name)
			}

			zeros.Merge(".", embeddedZeros.Msi())
			defaults.Merge(".", embeddedDefaults.Msi())

			continue
		}

		if tag, ok = s.readTag(targetField.Tag); !ok {
			continue
		}

		if targetField.Type.Kind() == reflect.Struct && targetField.Type != reflect.TypeOf(time.Time{}) {
			v, d, err := s.doReadZeroAndDefaultValues(targetValue.Interface())
			if err != nil {
				return nil, nil, fmt.Errorf("can not read from nested field %s", targetField.Name)
			}

			zeros.Set(tag.Name, v.Msi())
			defaults.Set(tag.Name, d.Msi())

			continue
		}

		if targetField.Type.Kind() == reflect.Slice {
			zeroValue = reflect.MakeSlice(targetField.Type, 0, 4).Interface()
			zeros.Set(tag.Name, zeroValue)
		}

		if targetField.Type.Kind() == reflect.Map {
			zeroValue = reflect.MakeMap(targetField.Type).Interface()
			zeros.Set(tag.Name, zeroValue)

			continue
		}

		zeroValue = reflect.Zero(targetField.Type).Interface()
		zeros.Set(tag.Name, zeroValue)

		if val, ok = targetField.Tag.Lookup(s.settings.DefaultTag); !ok {
			continue
		}

		if defValue, err = s.cast(targetField.Type, val); err != nil {
			return nil, nil, fmt.Errorf("can not read default from field %s: %w", tag.Name, err)
		}

		defaults.Set(tag.Name, defValue)
	}

	return zeros, defaults, nil
}

func (s *Struct) Read() (*MapX, error) {
	mapValues := NewMapX()

	if err := s.doReadStruct("", mapValues, s.target); err != nil {
		return nil, err
	}

	return mapValues, nil
}

func (s *Struct) doReadMap(path string, mapValues *MapX, mp any) error {
	if _, ok := mp.(map[string]any); ok {
		return s.doReadMsi(path, mapValues, mp.(map[string]any))
	}

	if _, ok := mp.(map[string]string); ok {
		return s.doReadMss(path, mapValues, mp.(map[string]string))
	}

	mapValue := reflect.ValueOf(mp)
	valueType := reflect.TypeOf(mp).Elem()

	switch valueType.Kind() {
	case reflect.Map:
		for _, key := range mapValue.MapKeys() {
			if key.Kind() != reflect.String {
				return fmt.Errorf("only string values are allowed as map keys for path %s", path)
			}

			element := mapValue.MapIndex(key).Interface()
			elementPath := fmt.Sprintf("%s.%s", path, key.String())

			if err := s.doReadMap(elementPath, mapValues, element); err != nil {
				return fmt.Errorf("can not read path value %s: %w", elementPath, err)
			}
		}
	case reflect.Struct:
		for _, key := range mapValue.MapKeys() {
			if key.Kind() != reflect.String {
				return fmt.Errorf("only string values are allowed as map keys for path %s", path)
			}

			element := mapValue.MapIndex(key).Interface()
			elementPath := fmt.Sprintf("%s.%s", path, key.String())

			if err := s.doReadStruct(elementPath, mapValues, element); err != nil {
				return fmt.Errorf("can not read path value %s: %w", elementPath, err)
			}
		}
	default:
		return fmt.Errorf("MSI fields or a map of structs are allowed only for path %s", path)
	}

	return nil
}

func (s *Struct) doReadMsi(path string, mapValues *MapX, msi map[string]any) error {
	for k, v := range msi {
		elementPath := fmt.Sprintf("%s.%s", path, k)
		mapValues.Set(elementPath, v)
	}

	return nil
}

func (s *Struct) doReadMss(path string, mapValues *MapX, mss map[string]string) error {
	for k, v := range mss {
		elementPath := fmt.Sprintf("%s.%s", path, k)
		mapValues.Set(elementPath, v)
	}

	return nil
}

func (s *Struct) doReadSlice(path string, mapValues *MapX, slice reflect.Value) error {
	sl := make([]any, 0, slice.Len())
	mapValues.Set(path, sl)

	for i := 0; i < slice.Len(); i++ {
		elementValue := slice.Index(i)
		elementPath := fmt.Sprintf("%s[%d]", path, i)
		element := elementValue.Interface()

		if elementValue.Kind() == reflect.Map {
			element = elementValue.Interface()

			if _, ok := element.(map[string]any); !ok {
				return fmt.Errorf("MSI fields are allowed only for path %s", elementPath)
			}

			if err := s.doReadMsi(elementPath, mapValues, element.(map[string]any)); err != nil {
				return err
			}

			continue
		}

		if elementValue.Kind() == reflect.Struct {
			if err := s.doReadStruct(elementPath, mapValues, element); err != nil {
				return fmt.Errorf("error on reading slice element on path %s: %w", elementPath, err)
			}

			continue
		}

		mapValues.Set(elementPath, element)
	}

	return nil
}

func (s *Struct) doReadStruct(path string, mapValues *MapX, target any) error {
	targetType := reflect.TypeOf(target)
	targetValue := reflect.ValueOf(target)

	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
		targetValue = targetValue.Elem()
	}

	var ok bool
	var tag *StructTag

	for i := 0; i < targetValue.NumField(); i++ {
		fieldType := targetType.Field(i)
		fieldValue := targetValue.Field(i)

		// skip unexported fields
		if fieldType.PkgPath != "" {
			continue
		}

		if fieldType.Anonymous {
			target = fieldValue.Interface()

			if err := s.doReadStruct(path, mapValues, target); err != nil {
				return err
			}

			continue
		}

		// skip fields without tag
		if tag, ok = s.readTag(fieldType.Tag); !ok {
			continue
		}

		if err := s.doReadValue(path, mapValues, tag, fieldType, fieldValue); err != nil {
			return err
		}
	}

	return nil
}

func (s *Struct) doReadValue(path string, mapValues *MapX, tag *StructTag, fieldType reflect.StructField, fieldValue reflect.Value) error {
	fieldPath := fmt.Sprintf("%s.%s", path, tag.Name)

	if fieldValue.Kind() == reflect.Map {
		target := fieldValue.Interface()

		if err := s.doReadMap(fieldPath, mapValues, target); err != nil {
			return err
		}

		return nil
	}

	if fieldValue.Kind() == reflect.Slice {
		if err := s.doReadSlice(fieldPath, mapValues, fieldValue); err != nil {
			return err
		}

		return nil
	}

	if fieldType.Type.Kind() == reflect.Struct && fieldValue.Type() != reflect.TypeOf(time.Time{}) {
		target := fieldValue.Interface()

		if err := s.doReadStruct(fieldPath, mapValues, target); err != nil {
			return fmt.Errorf("can not read nested struct values from path %s: %w", fieldPath, err)
		}

		return nil
	}

	value := fieldValue.Interface()
	mapValues.Set(fieldPath, value)

	return nil
}

func (s *Struct) Write(values *MapX) error {
	return s.doWrite(s.target, values)
}

func (s *Struct) doWrite(target any, sourceValues *MapX) error {
	st := reflect.TypeOf(target)
	sv := reflect.ValueOf(target)

	st = st.Elem()
	sv = sv.Elem()

	var tag *StructTag
	var ok bool

	for i := 0; i < st.NumField(); i++ {
		targetField := st.Field(i)
		targetValue := sv.Field(i)

		// skip unexported fields
		if targetField.PkgPath != "" {
			continue
		}

		if !targetValue.IsValid() {
			return fmt.Errorf("field %s is invalid", targetField.Name)
		}

		if !targetValue.CanSet() {
			return fmt.Errorf("field %s is not addressable", targetField.Name)
		}

		if targetField.Anonymous {
			if err := s.doWriteAnonymous(targetField.Name, targetValue, sourceValues); err != nil {
				return err
			}

			continue
		}

		if tag, ok = s.readTag(targetField.Tag); !ok {
			continue
		}

		if !sourceValues.Has(tag.Name) {
			continue
		}

		if err := s.doWriteValue(tag, sourceValues, targetValue); err != nil {
			return err
		}
	}

	return nil
}

func (s *Struct) doWriteValue(tag *StructTag, sourceValues *MapX, targetValue reflect.Value) (err error) {
	sourceValue := sourceValues.Get(tag.Name).Data()

	if targetValue.Type().Kind() == reflect.Ptr {
		targetValue.Set(reflect.New(targetValue.Type().Elem()))
		targetValue = targetValue.Elem()
	}

	if targetValue.Kind() == reflect.Map {
		if err := s.doWriteMap(tag, targetValue, sourceValues); err != nil {
			return err
		}

		return nil
	}

	if targetValue.Kind() == reflect.Slice {
		if err := s.doWriteSlice(tag, targetValue, sourceValues); err != nil {
			return err
		}

		return nil
	}

	if targetValue.Kind() == reflect.Struct && targetValue.Type() != reflect.TypeOf(time.Time{}) {
		if err := s.doWriteStruct(tag.Name, targetValue, sourceValues); err != nil {
			return err
		}

		return nil
	}

	if sourceValue, err = s.decodeAndCastValue(tag, targetValue.Type(), sourceValue); err != nil {
		return fmt.Errorf("can not decode and cast value for key %s: %w", tag.Name, err)
	}

	targetValue.Set(reflect.ValueOf(sourceValue))

	return nil
}

func (s *Struct) doWriteAnonymous(cfg string, targetValue reflect.Value, sourceValues *MapX) error {
	element := reflect.New(targetValue.Type())
	elementInterface := element.Interface()

	if err := s.doWrite(elementInterface, sourceValues); err != nil {
		return fmt.Errorf("can not write anonymous field %s: %w", cfg, err)
	}

	indirect := reflect.Indirect(element)
	targetValue.Set(indirect)

	return nil
}

func (s *Struct) doWriteMap(tag *StructTag, targetValue reflect.Value, sourceMap *MapX) error {
	var err error
	var elementValue reflect.Value
	var elementMap *MapX
	var finalValue any
	sourceData := sourceMap.Get(tag.Name).Data()

	sourceValue := reflect.ValueOf(sourceData)
	targetType := targetValue.Type()
	targetKeyType := targetType.Key()
	targetValue.Set(reflect.MakeMap(targetType))

	if sourceValue.Kind() != reflect.Map {
		return fmt.Errorf("value for field %s has to be a map but instead is %T", tag.Name, sourceData)
	}

	for _, key := range sourceValue.MapKeys() {
		if keyValue, err := s.cast(targetKeyType, key.Interface()); err != nil {
			return fmt.Errorf("key type %s does not match target type %s: %w", key.Type().Name(), targetKeyType.Name(), err)
		} else {
			key = reflect.ValueOf(keyValue)
		}

		selector := fmt.Sprintf("%s.%v", tag.Name, key.Interface())
		elementData := sourceMap.Get(selector)

		if elementData.IsMap() && targetValue.Type().Elem().Kind() == reflect.Struct {
			elementValue = reflect.New(targetValue.Type().Elem())
			elementInterface := elementValue.Interface()

			if elementMap, err = elementData.Map(); err != nil {
				return fmt.Errorf("element of field %s is not of type map: %w", tag.Name, err)
			}

			if err = s.doWrite(elementInterface, elementMap); err != nil {
				return fmt.Errorf("can not write map element of field %s: %w", tag.Name, err)
			}

			targetValue.SetMapIndex(key, elementValue.Elem())

			continue
		}

		targetMapElementType := targetValue.Type().Elem()
		elementValue := elementData.Data()

		if finalValue, err = s.decodeAndCastValue(tag, targetMapElementType, elementValue); err != nil {
			return fmt.Errorf("can not decode and cast value for key %s: %w", tag.Name, err)
		}

		targetValue.SetMapIndex(key, reflect.ValueOf(finalValue))
	}

	return nil
}

func (s *Struct) doWriteSlice(tag *StructTag, targetValue reflect.Value, sourceValues *MapX) error {
	var err error
	var finalValue any
	var interfaceSlice []any
	targetSliceElementType := targetValue.Type().Elem()

	sourceValue := sourceValues.Get(tag.Name).Data()

	if interfaceSlice, err = s.trySlice(sourceValue); err != nil {
		return fmt.Errorf("value for field %s has to be castable to []any but is of type %T: %w", tag.Name, sourceValue, err)
	}

	for j := 0; j < len(interfaceSlice); j++ {
		switch elementValue := interfaceSlice[j].(type) {
		case map[string]any:
			element := reflect.New(targetSliceElementType)
			elementInterface := element.Interface()
			elementMap := NewMapX(elementValue)

			if err := s.doWrite(elementInterface, elementMap); err != nil {
				return fmt.Errorf("can not write slice element of field %s: %w", tag.Name, err)
			}

			indirect := reflect.Indirect(element)
			targetValue.Set(reflect.Append(targetValue, indirect))
		default:
			if finalValue, err = s.decodeAndCastValue(tag, targetSliceElementType, elementValue); err != nil {
				return fmt.Errorf("can not decode and cast value for key %s: %w", tag.Name, err)
			}

			targetValue.Set(reflect.Append(targetValue, reflect.ValueOf(finalValue)))
		}
	}

	return nil
}

func (s *Struct) doWriteStruct(cfg string, targetValue reflect.Value, sourceValues *MapX) error {
	elementValues, err := sourceValues.Get(cfg).Map()
	if err != nil {
		return fmt.Errorf("value for field %s has to be a map but instead is %T", cfg, sourceValues.Get(cfg).Data())
	}

	element := reflect.New(targetValue.Type())
	element.Elem().Set(targetValue)
	elementInterface := element.Interface()

	if err := s.doWrite(elementInterface, elementValues); err != nil {
		return fmt.Errorf("can not write slice element of field %s: %w", cfg, err)
	}

	indirect := reflect.Indirect(element)
	targetValue.Set(indirect)

	return nil
}

func (s *Struct) decodeAndCastValue(tag *StructTag, targetType reflect.Type, sourceValue any) (any, error) {
	var err error

	if !tag.NoCast {
		if sourceValue, err = s.cast(targetType, sourceValue); err != nil {
			return nil, fmt.Errorf("provided value %v (type %T) doesn't match target type %v", sourceValue, sourceValue, targetType)
		}
	}

	if !tag.NoDecode {
		for _, decoder := range s.decoders {
			if sourceValue, err = decoder(targetType, sourceValue); err != nil {
				return nil, fmt.Errorf("can not decode value %v", sourceValue)
			}
		}
	}

	sourceType := reflect.TypeOf(sourceValue)

	if targetType.Kind() != reflect.Interface && targetType.Kind() != sourceType.Kind() {
		return nil, fmt.Errorf("target type %v and value type %T don't match", targetType, sourceValue)
	}

	return sourceValue, nil
}

func (s *Struct) cast(targetType reflect.Type, value any) (any, error) {
	valueType := reflect.TypeOf(value)

	if valueType.AssignableTo(targetType) {
		return value, nil
	}

	// IMPORTANT: don't convert if the kind would change - we don't want to convert integers to strings, otherwise
	// the int 80 becomes the string "P".
	if valueType.ConvertibleTo(targetType) && valueType.Kind() == targetType.Kind() {
		return reflect.ValueOf(value).Convert(targetType).Interface(), nil
	}

	if valueType.Kind() == reflect.Slice && targetType.Kind() == reflect.Slice {
		return s.castSlice(targetType, value)
	}

	if valueType.Kind() == reflect.Map && targetType.Kind() == reflect.Map {
		return s.castMap(targetType, value)
	}

	for _, caster := range s.casters {
		casted, err := caster(targetType, value)
		if err != nil {
			return nil, fmt.Errorf("caster %T failed: %w", caster, err)
		}

		if casted != nil {
			return casted, nil
		}
	}

	switch targetType.Kind() {
	case reflect.Bool:
		return cast.ToBoolE(value)
	case reflect.Int:
		return cast.ToIntE(value)
	case reflect.Int8:
		return cast.ToInt8E(value)
	case reflect.Int16:
		return cast.ToInt16E(value)
	case reflect.Int32:
		return cast.ToInt32E(value)
	case reflect.Int64:
		return cast.ToInt64E(value)
	case reflect.Interface:
		return value, nil
	case reflect.Float32:
		return cast.ToFloat32E(value)
	case reflect.Float64:
		return cast.ToFloat64E(value)
	case reflect.String:
		return cast.ToStringE(value)
	case reflect.Uint:
		return cast.ToUintE(value)
	case reflect.Uint8:
		return cast.ToUint8E(value)
	case reflect.Uint16:
		return cast.ToUint16E(value)
	case reflect.Uint32:
		return cast.ToUint32E(value)
	case reflect.Uint64:
		return cast.ToUint64E(value)
	default:
		return nil, fmt.Errorf("value %s is not castable to %s", value, targetType.Kind().String())
	}
}

func (s *Struct) castSlice(targetType reflect.Type, value any) (any, error) {
	elemType := targetType.Elem()
	reflectValue := reflect.ValueOf(value)
	resultSlice := reflect.MakeSlice(targetType, reflectValue.Len(), reflectValue.Len())

	for i := 0; i < reflectValue.Len(); i++ {
		if iValue, err := s.cast(elemType, reflectValue.Index(i).Interface()); err != nil {
			return nil, fmt.Errorf("could not cast element %d in slice: %w", i, err)
		} else {
			resultSlice.Index(i).Set(reflect.ValueOf(iValue))
		}
	}

	return resultSlice.Interface(), nil
}

func (s *Struct) castMap(targetType reflect.Type, value any) (any, error) {
	keyType := targetType.Key()
	elemType := targetType.Elem()
	reflectValue := reflect.ValueOf(value)
	resultMap := reflect.MakeMap(targetType)

	for _, key := range reflectValue.MapKeys() {
		var err error
		var keyValue, elemValue any

		if keyValue, err = s.cast(keyType, key.Interface()); err != nil {
			return nil, fmt.Errorf("could not cast key %v in map: %w", key.Interface(), err)
		}

		if elemValue, err = s.cast(elemType, reflectValue.MapIndex(key).Interface()); err != nil {
			return nil, fmt.Errorf("could not cast value at key %v in map: %w", key.Interface(), err)
		}

		resultMap.SetMapIndex(reflect.ValueOf(keyValue), reflect.ValueOf(elemValue))
	}

	return resultMap.Interface(), nil
}

func (s *Struct) trySlice(value any) ([]any, error) {
	var err error
	var str string
	var slice []any

	if slice, ok := value.([]any); ok {
		return slice, nil
	}

	rt := reflect.TypeOf(value)
	rv := reflect.ValueOf(value)

	if rt.Kind() == reflect.Slice {
		for i := 0; i < rv.Len(); i++ {
			slice = append(slice, rv.Index(i).Interface())
		}

		return slice, nil
	}

	if str, err = cast.ToStringE(value); err != nil {
		return nil, fmt.Errorf("value has to be castable to string: %w", err)
	}

	strSlice := strings.Split(str, ",")

	for i := range strSlice {
		strSlice[i] = strings.TrimSpace(strSlice[i])
		slice = append(slice, strSlice[i])
	}

	return slice, nil
}

func (s *Struct) readTag(sourceTag reflect.StructTag) (*StructTag, bool) {
	var ok bool
	var val string

	if val, ok = sourceTag.Lookup(s.settings.FieldTag); !ok {
		return nil, ok
	}

	parts := strings.Split(val, ",")
	parts = funk.Map(parts, strings.TrimSpace)

	tag := &StructTag{
		Name: parts[0],
	}

	if len(parts) == 1 {
		return tag, true
	}

	options := parts[1:]
	options = funk.Map(options, strings.ToLower)

	tag.NoCast = funk.Contains(options, optionNoCast)
	tag.NoDecode = funk.Contains(options, optionNoDecode)

	return tag, true
}
