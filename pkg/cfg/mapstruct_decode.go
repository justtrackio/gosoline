package cfg

import (
	"github.com/spf13/cast"
	"reflect"
	"strings"
	"time"
)

type MapStructDecoder func(targetType reflect.Type, val interface{}) (interface{}, error)

type MapStructCaster func(targetType reflect.Type, value interface{}) (interface{}, error)

func MapStructDurationCaster(targetType reflect.Type, value interface{}) (interface{}, error) {
	if targetType != reflect.TypeOf(time.Duration(0)) {
		return nil, nil
	}

	return cast.ToDurationE(value)
}

func MapStructSliceCaster(targetType reflect.Type, value interface{}) (interface{}, error) {
	if targetType.Kind() != reflect.Slice {
		return nil, nil
	}

	if reflect.ValueOf(value).Kind() == reflect.String {
		v := value.(string)

		if len(v) == 0 {
			return make([]string, 0), nil
		}

		return strings.Split(v, ","), nil
	}

	return []interface{}{value}, nil
}

func MapStructTimeCaster(targetType reflect.Type, value interface{}) (interface{}, error) {
	if targetType != reflect.TypeOf(time.Time{}) {
		return nil, nil
	}

	return cast.ToTimeE(value)
}
