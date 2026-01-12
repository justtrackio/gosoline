package mapx

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ettle/strcase"
	"github.com/spf13/cast"
)

type MapStructDecoder func(targetType reflect.Type, val any) (any, error)

type MapStructCaster func(targetType reflect.Type, value any) (any, error)

// SnakeCaseMatchName matches snake_case map keys to PascalCase/camelCase field names.
// The comparison is case-insensitive after converting the map key to PascalCase.
//
// Examples:
//   - "user_id" matches "UserId", "userId", "UserID"
//   - "created_at" matches "CreatedAt"
//   - "api_token" matches "ApiToken"
func SnakeCaseMatchName(mapKey, fieldName string) bool {
	return strings.EqualFold(strcase.ToPascal(mapKey), fieldName)
}

// MapStructTimeCaster casts string values to time.Time or *time.Time based on target type.
// For *time.Time: empty strings result in a nil pointer.
// For both: non-empty strings are parsed using cast.ToTimeE.
func MapStructTimeCaster(targetType reflect.Type, value any) (any, error) {
	timeType := reflect.TypeOf(time.Time{})
	timePtrType := reflect.TypeOf((*time.Time)(nil))

	switch targetType {
	case timePtrType:
		// Handle empty string -> nil pointer
		if str, ok := value.(string); ok && str == "" {
			return (*time.Time)(nil), nil
		}

		t, err := cast.ToTimeE(value)
		if err != nil {
			return nil, err
		}

		return &t, nil

	case timeType:
		return cast.ToTimeE(value)

	default:
		return nil, nil
	}
}

// MapStructDurationCaster casts values to time.Duration or *time.Duration based on target type.
// For *time.Duration: empty strings result in a nil pointer.
// For both: non-empty strings are parsed using cast.ToDurationE.
func MapStructDurationCaster(targetType reflect.Type, value any) (any, error) {
	durationType := reflect.TypeOf(time.Duration(0))
	durationPtrType := reflect.TypeOf((*time.Duration)(nil))

	switch targetType {
	case durationPtrType:
		// Handle empty string -> nil pointer
		if str, ok := value.(string); ok && str == "" {
			return (*time.Duration)(nil), nil
		}

		d, err := cast.ToDurationE(value)
		if err != nil {
			return nil, err
		}

		return &d, nil

	case durationType:
		return cast.ToDurationE(value)

	default:
		return nil, nil
	}
}

var mapStructSliceCasters = map[reflect.Kind]MapStructCaster{
	reflect.Bool:    MapStructBoolSliceCaster,
	reflect.Float32: MapStructFloat32SliceCaster,
	reflect.Float64: MapStructFloat64SliceCaster,
	reflect.Int:     MapStructIntSliceCaster,
	reflect.Int64:   MapStructInt64SliceCaster,
	reflect.String:  MapStructStringSliceCaster,
}

func MapStructStringSliceCaster(_ reflect.Type, value any) (any, error) {
	return strings.Split(value.(string), ","), nil
}

func MapStructIntSliceCaster(_ reflect.Type, value any) (any, error) {
	bits := strings.Split(value.(string), ",")

	return cast.ToIntSliceE(bits)
}

func MapStructInt64SliceCaster(_ reflect.Type, value any) (any, error) {
	bits := strings.Split(value.(string), ",")
	out := make([]int64, len(bits))
	var err error

	for i, bit := range bits {
		out[i], err = cast.ToInt64E(bit)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func MapStructFloat32SliceCaster(_ reflect.Type, value any) (any, error) {
	bits := strings.Split(value.(string), ",")
	out := make([]float32, len(bits))
	var err error

	for i, bit := range bits {
		out[i], err = cast.ToFloat32E(bit)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func MapStructFloat64SliceCaster(_ reflect.Type, value any) (any, error) {
	bits := strings.Split(value.(string), ",")
	out := make([]float64, len(bits))
	var err error

	for i, bit := range bits {
		out[i], err = cast.ToFloat64E(bit)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func MapStructBoolSliceCaster(_ reflect.Type, value any) (any, error) {
	bits := strings.Split(value.(string), ",")

	return cast.ToBoolSliceE(bits)
}

// MapStructSliceCaster casts values to []T, based on casters in mapStructSliceCasters
func MapStructSliceCaster(targetType reflect.Type, value any) (any, error) {
	if targetType.Kind() != reflect.Slice {
		return nil, nil
	}

	elemType := targetType.Elem()

	caster, ok := mapStructSliceCasters[elemType.Kind()]
	if !ok {
		return nil, fmt.Errorf("no slice caster found for type []%s", elemType.String())
	}

	out, err := caster(targetType, value)
	if err != nil {
		return nil, fmt.Errorf("caster %T failed to cast value %v to type []%s: %w", caster, value, elemType.String(), err)
	}

	return out, nil
}
