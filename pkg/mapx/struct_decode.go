package mapx

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cast"
)

type MapStructDecoder func(targetType reflect.Type, val any) (any, error)

type MapStructCaster func(targetType reflect.Type, value any) (any, error)

func MapStructDurationCaster(targetType reflect.Type, value any) (any, error) {
	if targetType != reflect.TypeOf(time.Duration(0)) {
		return nil, nil
	}

	return cast.ToDurationE(value)
}

func MapStructTimeCaster(targetType reflect.Type, value any) (any, error) {
	if targetType != reflect.TypeOf(time.Time{}) {
		return nil, nil
	}

	return cast.ToTimeE(value)
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
