package cfg

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/stretchr/objx"
	"reflect"
	"strings"
	"time"
)

type MapStructSettings struct {
	FieldTag   string
	DefaultTag string
	Casters    []MapStructCaster
	Decoders   []MapStructDecoder
}

type MapStruct struct {
	target   interface{}
	casters  []MapStructCaster
	decoders []MapStructDecoder
	settings *MapStructSettings
}

func NewMapStruct(source interface{}, settings *MapStructSettings) (*MapStruct, error) {
	st := reflect.TypeOf(source)

	if st.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("the target value has to be a pointer")
	}

	return &MapStruct{
		target:   source,
		casters:  append([]MapStructCaster{}, settings.Casters...),
		decoders: append([]MapStructDecoder{}, settings.Decoders...),
		settings: settings,
	}, nil
}

func (m *MapStruct) ReadZeroAndDefaultValues() (objx.Map, objx.Map, error) {
	sv := reflect.ValueOf(m.target).Elem().Interface()

	return m.doReadZeroAndDefaultValues(sv)
}

func (m *MapStruct) doReadZeroAndDefaultValues(target interface{}) (objx.Map, objx.Map, error) {
	st := reflect.TypeOf(target)
	sv := reflect.ValueOf(target)

	var err error
	var cfg, val string
	var ok bool
	var values, defaults = objx.Map{}, objx.Map{}

	for i := 0; i < st.NumField(); i++ {
		targetField := st.Field(i)
		targetValue := sv.Field(i)

		// skip unexported fields
		if len(targetField.PkgPath) != 0 {
			continue
		}

		if targetField.Type.Kind() == reflect.Struct && targetField.Anonymous {
			embeddedZeros, embeddedDefaults, err := m.doReadZeroAndDefaultValues(targetValue.Interface())

			if err != nil {
				return nil, nil, fmt.Errorf("can not read from embedded field %s", targetField.Name)
			}

			values.MergeHere(embeddedZeros.Value().MSI())
			defaults.MergeHere(embeddedDefaults.Value().MSI())

			continue
		}

		if cfg, ok = targetField.Tag.Lookup(m.settings.FieldTag); !ok {
			continue
		}

		if targetField.Type.Kind() == reflect.Struct && targetField.Type != reflect.TypeOf(time.Time{}) {
			v, d, err := m.doReadZeroAndDefaultValues(targetValue.Interface())

			if err != nil {
				return nil, nil, fmt.Errorf("can not read from nested field %s", targetField.Name)
			}
			values[cfg] = v.Value().MSI()
			defaults[cfg] = d.Value().MSI()

			continue
		}

		if targetField.Type.Kind() == reflect.Slice {
			values[cfg] = reflect.MakeSlice(targetField.Type, 0, 4).Interface()
		}

		if targetField.Type.Kind() == reflect.Map {
			values[cfg] = reflect.MakeMap(targetField.Type).Interface()
			continue
		}

		if targetField.Type.Kind() != reflect.Slice {
			values[cfg] = reflect.Zero(targetField.Type).Interface()
		}

		if val, ok = targetField.Tag.Lookup(m.settings.DefaultTag); !ok {
			continue
		}

		if defaults[cfg], err = m.cast(targetField.Type, val); err != nil {
			return nil, nil, fmt.Errorf("can not read default from field %s: %w", cfg, err)
		}
	}

	return values, defaults, nil
}

func (m *MapStruct) Read() (*Map, error) {
	mapValues := NewMap()

	if err := m.doReadStruct("", mapValues, m.target); err != nil {
		return nil, err
	}

	return mapValues, nil
}

func (m *MapStruct) doReadMap(path string, mapValues *Map, mp interface{}) error {
	if _, ok := mp.(map[string]interface{}); ok {
		return m.doReadMsi(path, mapValues, mp.(map[string]interface{}))
	}

	valueType := reflect.TypeOf(mp).Elem()

	if valueType.Kind() != reflect.Struct {
		return fmt.Errorf("MSI fields or a map of structs are allowed only for path %s", path)
	}

	mapValue := reflect.ValueOf(mp)
	for _, key := range mapValue.MapKeys() {
		if key.Kind() != reflect.String {
			return fmt.Errorf("only string values are allowed as map keys for path %s", path)
		}

		element := mapValue.MapIndex(key).Interface()
		elementPath := fmt.Sprintf("%s.%s", path, key.String())

		if err := m.doReadStruct(elementPath, mapValues, element); err != nil {
			return fmt.Errorf("can not read path value %s: %w", elementPath, err)
		}
	}

	return nil
}

func (m *MapStruct) doReadMsi(path string, mapValues *Map, msi map[string]interface{}) error {
	for k, v := range msi {
		elementPath := fmt.Sprintf("%s.%s", path, k)
		mapValues.Set(elementPath, v)
	}

	return nil
}

func (m *MapStruct) doReadSlice(path string, mapValues *Map, slice reflect.Value) error {
	sl := make([]interface{}, 0, slice.Len())
	mapValues.Set(path, sl)

	for i := 0; i < slice.Len(); i++ {
		elementValue := slice.Index(i)
		elementPath := fmt.Sprintf("%s[%d]", path, i)
		element := elementValue.Interface()

		if elementValue.Kind() == reflect.Map {
			element = elementValue.Interface()

			if _, ok := element.(map[string]interface{}); !ok {
				return fmt.Errorf("MSI fields are allowed only for path %s", elementPath)
			}

			if err := m.doReadMsi(elementPath, mapValues, element.(map[string]interface{})); err != nil {
				return err
			}

			continue
		}

		if elementValue.Kind() == reflect.Struct {
			if err := m.doReadStruct(elementPath, mapValues, element); err != nil {
				return fmt.Errorf("error on reading slice element on path %s", elementPath)
			}

			continue
		}

		mapValues.Set(elementPath, element)
	}

	return nil
}

func (m *MapStruct) doReadStruct(path string, mapValues *Map, target interface{}) error {
	targetType := reflect.TypeOf(target)
	targetValue := reflect.ValueOf(target)

	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
		targetValue = targetValue.Elem()
	}

	var ok bool
	var err error
	var cfg, fieldPath string

	for i := 0; i < targetValue.NumField(); i++ {
		fieldType := targetType.Field(i)
		fieldValue := targetValue.Field(i)

		// skip unexported fields
		if len(fieldType.PkgPath) != 0 {
			continue
		}

		if fieldType.Anonymous {
			target = fieldValue.Interface()

			if err = m.doReadStruct(path, mapValues, target); err != nil {
				return err
			}

			continue
		}

		// skip fields without tag
		if cfg, ok = fieldType.Tag.Lookup(m.settings.FieldTag); !ok {
			continue
		}

		fieldPath = fmt.Sprintf("%s.%s", path, cfg)

		if fieldValue.Kind() == reflect.Map {
			target = fieldValue.Interface()

			if err = m.doReadMap(fieldPath, mapValues, target); err != nil {
				return err
			}

			continue
		}

		if fieldValue.Kind() == reflect.Slice {
			if err = m.doReadSlice(fieldPath, mapValues, fieldValue); err != nil {
				return err
			}

			continue
		}

		if fieldType.Type.Kind() == reflect.Struct && fieldValue.Type() != reflect.TypeOf(time.Time{}) {
			target = fieldValue.Interface()

			if err = m.doReadStruct(fieldPath, mapValues, target); err != nil {
				return fmt.Errorf("can not read nested struct values from path %s: %w", fieldPath, err)
			}

			continue
		}

		value := fieldValue.Interface()
		mapValues.Set(fieldPath, value)
	}

	return nil
}

func (m *MapStruct) Write(values map[string]interface{}) error {
	return m.doWrite(m.target, values)
}

func (m *MapStruct) doWrite(target interface{}, sourceValues objx.Map) error {
	st := reflect.TypeOf(target)
	sv := reflect.ValueOf(target)

	st = st.Elem()
	sv = sv.Elem()

	var err error
	var cfg string
	var sourceValue interface{}
	var ok bool

	for i := 0; i < st.NumField(); i++ {
		targetField := st.Field(i)
		targetValue := sv.Field(i)

		// skip unexported fields
		if len(targetField.PkgPath) != 0 {
			continue
		}

		if !targetValue.IsValid() {
			return fmt.Errorf("field %s is invalid", cfg)
		}

		if !targetValue.CanSet() {
			return fmt.Errorf("field %s is not addressable", cfg)
		}

		if targetField.Anonymous {
			if err = m.doWriteAnonymous(cfg, targetValue, sourceValues); err != nil {
				return err
			}

			continue
		}

		if cfg, ok = targetField.Tag.Lookup(m.settings.FieldTag); !ok {
			continue
		}

		if !sourceValues.Has(cfg) {
			continue
		}

		sourceValue = sourceValues.Get(cfg).Data()

		if targetValue.Kind() == reflect.Map {
			if err = m.doWriteMap(cfg, targetValue, sourceValues); err != nil {
				return err
			}

			continue
		}

		if targetValue.Kind() == reflect.Slice {
			if err = m.doWriteSlice(cfg, targetValue, sourceValues); err != nil {
				return err
			}

			continue
		}

		if targetValue.Kind() == reflect.Struct && targetValue.Type() != reflect.TypeOf(time.Time{}) {
			if err = m.doWriteStruct(cfg, targetValue, sourceValues); err != nil {
				return err
			}

			continue
		}

		if sourceValue, err = m.decodeAndCastValue(targetValue.Type(), sourceValue); err != nil {
			return fmt.Errorf("can not decode and cast value for key %s: %w", cfg, err)
		}

		targetValue.Set(reflect.ValueOf(sourceValue))
	}

	return nil
}

func (m *MapStruct) doWriteAnonymous(cfg string, targetValue reflect.Value, sourceValues objx.Map) error {
	element := reflect.New(targetValue.Type())
	elementInterface := element.Interface()

	if err := m.doWrite(elementInterface, sourceValues); err != nil {
		return fmt.Errorf("can not write anonymous field %s: %w", cfg, err)
	}

	indirect := reflect.Indirect(element)
	targetValue.Set(indirect)

	return nil
}

func (m *MapStruct) doWriteMap(cfg string, targetValue reflect.Value, sourceValues objx.Map) error {
	var err error
	var finalValue interface{}
	var sourceValue = sourceValues.Get(cfg).Data()

	mlv := reflect.ValueOf(sourceValue)
	targetValue.Set(reflect.MakeMap(targetValue.Type()))

	if mlv.Kind() != reflect.Map {
		return fmt.Errorf("value for field %s has to be a map but instead is %T", cfg, sourceValue)
	}

	for _, key := range mlv.MapKeys() {
		selector := fmt.Sprintf("%s.%s", cfg, key.String())
		elementValueX := sourceValues.Get(selector)

		switch elementValueX.Data().(type) {
		case map[string]interface{}:
			element := reflect.New(targetValue.Type().Elem())
			elementInterface := element.Interface()

			if err = m.doWrite(elementInterface, elementValueX.MSI()); err != nil {
				return fmt.Errorf("can not write map element of field %s: %w", cfg, err)
			}

			indirect := reflect.Indirect(element)
			targetValue.SetMapIndex(key, indirect)
		default:
			targetMapElementType := targetValue.Type().Elem()
			elementValue := elementValueX.Data()

			if finalValue, err = m.decodeAndCastValue(targetMapElementType, elementValue); err != nil {
				return fmt.Errorf("can not decode and cast value for key %s: %w", cfg, err)
			}

			targetValue.SetMapIndex(key, reflect.ValueOf(finalValue))
		}
	}

	return nil
}

func (m *MapStruct) doWriteSlice(cfg string, targetValue reflect.Value, sourceValues objx.Map) error {
	var err error
	var finalValue interface{}
	var interfaceSlice []interface{}
	var targetSliceElementType = targetValue.Type().Elem()

	sourceValue := sourceValues.Get(cfg).Data()

	if interfaceSlice, err = m.trySlice(sourceValue); err != nil {
		return fmt.Errorf("value for field %s has to be castable to []interface{} but is of type %T: %w", cfg, sourceValue, err)
	}

	for j := 0; j < len(interfaceSlice); j++ {
		switch elementValue := interfaceSlice[j].(type) {
		case map[string]interface{}:
			element := reflect.New(targetSliceElementType)
			elementInterface := element.Interface()

			if err := m.doWrite(elementInterface, elementValue); err != nil {
				return fmt.Errorf("can not write slice element of field %s: %w", cfg, err)
			}

			indirect := reflect.Indirect(element)
			targetValue.Set(reflect.Append(targetValue, indirect))
		default:
			if finalValue, err = m.decodeAndCastValue(targetSliceElementType, elementValue); err != nil {
				return fmt.Errorf("can not decode and cast value for key %s: %w", cfg, err)
			}

			targetValue.Set(reflect.Append(targetValue, reflect.ValueOf(finalValue)))
		}
	}

	return nil
}

func (m *MapStruct) doWriteStruct(cfg string, targetValue reflect.Value, sourceValues objx.Map) error {
	elementValues := sourceValues.Get(cfg).MSI()

	if elementValues == nil {
		return fmt.Errorf("value for field %s has to be a map but instead is %T", cfg, sourceValues.Get(cfg).Data())
	}

	element := reflect.New(targetValue.Type())
	elementInterface := element.Interface()

	if err := m.doWrite(elementInterface, elementValues); err != nil {
		return fmt.Errorf("can not write slice element of field %s: %w", cfg, err)
	}

	indirect := reflect.Indirect(element)
	targetValue.Set(indirect)

	return nil
}

func (m *MapStruct) decodeAndCastValue(targetType reflect.Type, sourceValue interface{}) (interface{}, error) {
	var err error

	if sourceValue, err = m.cast(targetType, sourceValue); err != nil {
		return nil, fmt.Errorf("provided value %v doesn't match target type %v", sourceValue, targetType)
	}

	for _, decoder := range m.decoders {
		if sourceValue, err = decoder(targetType, sourceValue); err != nil {
			return nil, fmt.Errorf("can not decode value %v", sourceValue)
		}
	}

	sourceType := reflect.TypeOf(sourceValue)

	if targetType.Kind() != reflect.Interface && targetType.Kind() != sourceType.Kind() {
		return nil, fmt.Errorf("target type %v and value type %T don't match", targetType, sourceValue)
	}

	return sourceValue, nil
}

func (m *MapStruct) cast(targetType reflect.Type, value interface{}) (interface{}, error) {
	for _, caster := range m.casters {
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
	}

	return nil, fmt.Errorf("value %s is not castable to %s", value, targetType.Kind().String())
}

func (m *MapStruct) trySlice(value interface{}) ([]interface{}, error) {
	var err error
	var str string
	var slice []interface{}

	if slice, ok := value.([]interface{}); ok {
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
