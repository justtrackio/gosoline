package mapx

import (
	"github.com/spf13/cast"
	"reflect"
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

func MapStructTimeCaster(targetType reflect.Type, value interface{}) (interface{}, error) {
	if targetType != reflect.TypeOf(time.Time{}) {
		return nil, nil
	}

	return cast.ToTimeE(value)
}
