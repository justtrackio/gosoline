// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	cfg "github.com/justtrackio/gosoline/pkg/cfg"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Config is an autogenerated mock type for the Config type
type Config struct {
	mock.Mock
}

type Config_Expecter struct {
	mock *mock.Mock
}

func (_m *Config) EXPECT() *Config_Expecter {
	return &Config_Expecter{mock: &_m.Mock}
}

// AllKeys provides a mock function with no fields
func (_m *Config) AllKeys() []string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for AllKeys")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// Config_AllKeys_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AllKeys'
type Config_AllKeys_Call struct {
	*mock.Call
}

// AllKeys is a helper method to define mock.On call
func (_e *Config_Expecter) AllKeys() *Config_AllKeys_Call {
	return &Config_AllKeys_Call{Call: _e.mock.On("AllKeys")}
}

func (_c *Config_AllKeys_Call) Run(run func()) *Config_AllKeys_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Config_AllKeys_Call) Return(_a0 []string) *Config_AllKeys_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Config_AllKeys_Call) RunAndReturn(run func() []string) *Config_AllKeys_Call {
	_c.Call.Return(run)
	return _c
}

// AllSettings provides a mock function with no fields
func (_m *Config) AllSettings() map[string]interface{} {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for AllSettings")
	}

	var r0 map[string]interface{}
	if rf, ok := ret.Get(0).(func() map[string]interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]interface{})
		}
	}

	return r0
}

// Config_AllSettings_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AllSettings'
type Config_AllSettings_Call struct {
	*mock.Call
}

// AllSettings is a helper method to define mock.On call
func (_e *Config_Expecter) AllSettings() *Config_AllSettings_Call {
	return &Config_AllSettings_Call{Call: _e.mock.On("AllSettings")}
}

func (_c *Config_AllSettings_Call) Run(run func()) *Config_AllSettings_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Config_AllSettings_Call) Return(_a0 map[string]interface{}) *Config_AllSettings_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Config_AllSettings_Call) RunAndReturn(run func() map[string]interface{}) *Config_AllSettings_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: key, optionalDefault
func (_m *Config) Get(key string, optionalDefault ...interface{}) (interface{}, error) {
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, optionalDefault...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...interface{}) (interface{}, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...interface{}) interface{}); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...interface{}) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type Config_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...interface{}
func (_e *Config_Expecter) Get(key interface{}, optionalDefault ...interface{}) *Config_Get_Call {
	return &Config_Get_Call{Call: _e.mock.On("Get",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_Get_Call) Run(run func(key string, optionalDefault ...interface{})) *Config_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]interface{}, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(interface{})
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_Get_Call) Return(_a0 interface{}, _a1 error) *Config_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_Get_Call) RunAndReturn(run func(string, ...interface{}) (interface{}, error)) *Config_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetBool provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetBool(key string, optionalDefault ...bool) (bool, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetBool")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...bool) (bool, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...bool) bool); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string, ...bool) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetBool_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBool'
type Config_GetBool_Call struct {
	*mock.Call
}

// GetBool is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...bool
func (_e *Config_Expecter) GetBool(key interface{}, optionalDefault ...interface{}) *Config_GetBool_Call {
	return &Config_GetBool_Call{Call: _e.mock.On("GetBool",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetBool_Call) Run(run func(key string, optionalDefault ...bool)) *Config_GetBool_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]bool, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(bool)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetBool_Call) Return(_a0 bool, _a1 error) *Config_GetBool_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetBool_Call) RunAndReturn(run func(string, ...bool) (bool, error)) *Config_GetBool_Call {
	_c.Call.Return(run)
	return _c
}

// GetDuration provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetDuration(key string, optionalDefault ...time.Duration) (time.Duration, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetDuration")
	}

	var r0 time.Duration
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...time.Duration) (time.Duration, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...time.Duration) time.Duration); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	if rf, ok := ret.Get(1).(func(string, ...time.Duration) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetDuration_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetDuration'
type Config_GetDuration_Call struct {
	*mock.Call
}

// GetDuration is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...time.Duration
func (_e *Config_Expecter) GetDuration(key interface{}, optionalDefault ...interface{}) *Config_GetDuration_Call {
	return &Config_GetDuration_Call{Call: _e.mock.On("GetDuration",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetDuration_Call) Run(run func(key string, optionalDefault ...time.Duration)) *Config_GetDuration_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]time.Duration, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(time.Duration)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetDuration_Call) Return(_a0 time.Duration, _a1 error) *Config_GetDuration_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetDuration_Call) RunAndReturn(run func(string, ...time.Duration) (time.Duration, error)) *Config_GetDuration_Call {
	_c.Call.Return(run)
	return _c
}

// GetFloat64 provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetFloat64(key string, optionalDefault ...float64) (float64, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetFloat64")
	}

	var r0 float64
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...float64) (float64, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...float64) float64); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		r0 = ret.Get(0).(float64)
	}

	if rf, ok := ret.Get(1).(func(string, ...float64) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetFloat64_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetFloat64'
type Config_GetFloat64_Call struct {
	*mock.Call
}

// GetFloat64 is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...float64
func (_e *Config_Expecter) GetFloat64(key interface{}, optionalDefault ...interface{}) *Config_GetFloat64_Call {
	return &Config_GetFloat64_Call{Call: _e.mock.On("GetFloat64",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetFloat64_Call) Run(run func(key string, optionalDefault ...float64)) *Config_GetFloat64_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]float64, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(float64)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetFloat64_Call) Return(_a0 float64, _a1 error) *Config_GetFloat64_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetFloat64_Call) RunAndReturn(run func(string, ...float64) (float64, error)) *Config_GetFloat64_Call {
	_c.Call.Return(run)
	return _c
}

// GetInt provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetInt(key string, optionalDefault ...int) (int, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetInt")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...int) (int, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...int) int); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(string, ...int) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetInt_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetInt'
type Config_GetInt_Call struct {
	*mock.Call
}

// GetInt is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...int
func (_e *Config_Expecter) GetInt(key interface{}, optionalDefault ...interface{}) *Config_GetInt_Call {
	return &Config_GetInt_Call{Call: _e.mock.On("GetInt",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetInt_Call) Run(run func(key string, optionalDefault ...int)) *Config_GetInt_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]int, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(int)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetInt_Call) Return(_a0 int, _a1 error) *Config_GetInt_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetInt_Call) RunAndReturn(run func(string, ...int) (int, error)) *Config_GetInt_Call {
	_c.Call.Return(run)
	return _c
}

// GetIntSlice provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetIntSlice(key string, optionalDefault ...[]int) ([]int, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetIntSlice")
	}

	var r0 []int
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...[]int) ([]int, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...[]int) []int); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]int)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...[]int) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetIntSlice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetIntSlice'
type Config_GetIntSlice_Call struct {
	*mock.Call
}

// GetIntSlice is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...[]int
func (_e *Config_Expecter) GetIntSlice(key interface{}, optionalDefault ...interface{}) *Config_GetIntSlice_Call {
	return &Config_GetIntSlice_Call{Call: _e.mock.On("GetIntSlice",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetIntSlice_Call) Run(run func(key string, optionalDefault ...[]int)) *Config_GetIntSlice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([][]int, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.([]int)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetIntSlice_Call) Return(_a0 []int, _a1 error) *Config_GetIntSlice_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetIntSlice_Call) RunAndReturn(run func(string, ...[]int) ([]int, error)) *Config_GetIntSlice_Call {
	_c.Call.Return(run)
	return _c
}

// GetMsiSlice provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetMsiSlice(key string, optionalDefault ...[]map[string]interface{}) ([]map[string]interface{}, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetMsiSlice")
	}

	var r0 []map[string]interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...[]map[string]interface{}) ([]map[string]interface{}, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...[]map[string]interface{}) []map[string]interface{}); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]map[string]interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...[]map[string]interface{}) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetMsiSlice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetMsiSlice'
type Config_GetMsiSlice_Call struct {
	*mock.Call
}

// GetMsiSlice is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...[]map[string]interface{}
func (_e *Config_Expecter) GetMsiSlice(key interface{}, optionalDefault ...interface{}) *Config_GetMsiSlice_Call {
	return &Config_GetMsiSlice_Call{Call: _e.mock.On("GetMsiSlice",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetMsiSlice_Call) Run(run func(key string, optionalDefault ...[]map[string]interface{})) *Config_GetMsiSlice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([][]map[string]interface{}, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.([]map[string]interface{})
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetMsiSlice_Call) Return(_a0 []map[string]interface{}, _a1 error) *Config_GetMsiSlice_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetMsiSlice_Call) RunAndReturn(run func(string, ...[]map[string]interface{}) ([]map[string]interface{}, error)) *Config_GetMsiSlice_Call {
	_c.Call.Return(run)
	return _c
}

// GetString provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetString(key string, optionalDefault ...string) (string, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetString")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...string) (string, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...string) string); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string, ...string) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetString_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetString'
type Config_GetString_Call struct {
	*mock.Call
}

// GetString is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...string
func (_e *Config_Expecter) GetString(key interface{}, optionalDefault ...interface{}) *Config_GetString_Call {
	return &Config_GetString_Call{Call: _e.mock.On("GetString",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetString_Call) Run(run func(key string, optionalDefault ...string)) *Config_GetString_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]string, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(string)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetString_Call) Return(_a0 string, _a1 error) *Config_GetString_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetString_Call) RunAndReturn(run func(string, ...string) (string, error)) *Config_GetString_Call {
	_c.Call.Return(run)
	return _c
}

// GetStringMap provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetStringMap(key string, optionalDefault ...map[string]interface{}) (map[string]interface{}, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetStringMap")
	}

	var r0 map[string]interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...map[string]interface{}) (map[string]interface{}, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...map[string]interface{}) map[string]interface{}); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...map[string]interface{}) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetStringMap_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetStringMap'
type Config_GetStringMap_Call struct {
	*mock.Call
}

// GetStringMap is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...map[string]interface{}
func (_e *Config_Expecter) GetStringMap(key interface{}, optionalDefault ...interface{}) *Config_GetStringMap_Call {
	return &Config_GetStringMap_Call{Call: _e.mock.On("GetStringMap",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetStringMap_Call) Run(run func(key string, optionalDefault ...map[string]interface{})) *Config_GetStringMap_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]map[string]interface{}, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(map[string]interface{})
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetStringMap_Call) Return(_a0 map[string]interface{}, _a1 error) *Config_GetStringMap_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetStringMap_Call) RunAndReturn(run func(string, ...map[string]interface{}) (map[string]interface{}, error)) *Config_GetStringMap_Call {
	_c.Call.Return(run)
	return _c
}

// GetStringMapString provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetStringMapString(key string, optionalDefault ...map[string]string) (map[string]string, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetStringMapString")
	}

	var r0 map[string]string
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...map[string]string) (map[string]string, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...map[string]string) map[string]string); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...map[string]string) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetStringMapString_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetStringMapString'
type Config_GetStringMapString_Call struct {
	*mock.Call
}

// GetStringMapString is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...map[string]string
func (_e *Config_Expecter) GetStringMapString(key interface{}, optionalDefault ...interface{}) *Config_GetStringMapString_Call {
	return &Config_GetStringMapString_Call{Call: _e.mock.On("GetStringMapString",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetStringMapString_Call) Run(run func(key string, optionalDefault ...map[string]string)) *Config_GetStringMapString_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]map[string]string, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(map[string]string)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetStringMapString_Call) Return(_a0 map[string]string, _a1 error) *Config_GetStringMapString_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetStringMapString_Call) RunAndReturn(run func(string, ...map[string]string) (map[string]string, error)) *Config_GetStringMapString_Call {
	_c.Call.Return(run)
	return _c
}

// GetStringSlice provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetStringSlice(key string, optionalDefault ...[]string) ([]string, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetStringSlice")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...[]string) ([]string, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...[]string) []string); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...[]string) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetStringSlice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetStringSlice'
type Config_GetStringSlice_Call struct {
	*mock.Call
}

// GetStringSlice is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...[]string
func (_e *Config_Expecter) GetStringSlice(key interface{}, optionalDefault ...interface{}) *Config_GetStringSlice_Call {
	return &Config_GetStringSlice_Call{Call: _e.mock.On("GetStringSlice",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetStringSlice_Call) Run(run func(key string, optionalDefault ...[]string)) *Config_GetStringSlice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([][]string, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.([]string)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetStringSlice_Call) Return(_a0 []string, _a1 error) *Config_GetStringSlice_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetStringSlice_Call) RunAndReturn(run func(string, ...[]string) ([]string, error)) *Config_GetStringSlice_Call {
	_c.Call.Return(run)
	return _c
}

// GetTime provides a mock function with given fields: key, optionalDefault
func (_m *Config) GetTime(key string, optionalDefault ...time.Time) (time.Time, error) {
	_va := make([]interface{}, len(optionalDefault))
	for _i := range optionalDefault {
		_va[_i] = optionalDefault[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetTime")
	}

	var r0 time.Time
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...time.Time) (time.Time, error)); ok {
		return rf(key, optionalDefault...)
	}
	if rf, ok := ret.Get(0).(func(string, ...time.Time) time.Time); ok {
		r0 = rf(key, optionalDefault...)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	if rf, ok := ret.Get(1).(func(string, ...time.Time) error); ok {
		r1 = rf(key, optionalDefault...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config_GetTime_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetTime'
type Config_GetTime_Call struct {
	*mock.Call
}

// GetTime is a helper method to define mock.On call
//   - key string
//   - optionalDefault ...time.Time
func (_e *Config_Expecter) GetTime(key interface{}, optionalDefault ...interface{}) *Config_GetTime_Call {
	return &Config_GetTime_Call{Call: _e.mock.On("GetTime",
		append([]interface{}{key}, optionalDefault...)...)}
}

func (_c *Config_GetTime_Call) Run(run func(key string, optionalDefault ...time.Time)) *Config_GetTime_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]time.Time, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(time.Time)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *Config_GetTime_Call) Return(_a0 time.Time, _a1 error) *Config_GetTime_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Config_GetTime_Call) RunAndReturn(run func(string, ...time.Time) (time.Time, error)) *Config_GetTime_Call {
	_c.Call.Return(run)
	return _c
}

// HasPrefix provides a mock function with given fields: prefix
func (_m *Config) HasPrefix(prefix string) bool {
	ret := _m.Called(prefix)

	if len(ret) == 0 {
		panic("no return value specified for HasPrefix")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(prefix)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Config_HasPrefix_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HasPrefix'
type Config_HasPrefix_Call struct {
	*mock.Call
}

// HasPrefix is a helper method to define mock.On call
//   - prefix string
func (_e *Config_Expecter) HasPrefix(prefix interface{}) *Config_HasPrefix_Call {
	return &Config_HasPrefix_Call{Call: _e.mock.On("HasPrefix", prefix)}
}

func (_c *Config_HasPrefix_Call) Run(run func(prefix string)) *Config_HasPrefix_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Config_HasPrefix_Call) Return(_a0 bool) *Config_HasPrefix_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Config_HasPrefix_Call) RunAndReturn(run func(string) bool) *Config_HasPrefix_Call {
	_c.Call.Return(run)
	return _c
}

// IsSet provides a mock function with given fields: _a0
func (_m *Config) IsSet(_a0 string) bool {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for IsSet")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Config_IsSet_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsSet'
type Config_IsSet_Call struct {
	*mock.Call
}

// IsSet is a helper method to define mock.On call
//   - _a0 string
func (_e *Config_Expecter) IsSet(_a0 interface{}) *Config_IsSet_Call {
	return &Config_IsSet_Call{Call: _e.mock.On("IsSet", _a0)}
}

func (_c *Config_IsSet_Call) Run(run func(_a0 string)) *Config_IsSet_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Config_IsSet_Call) Return(_a0 bool) *Config_IsSet_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Config_IsSet_Call) RunAndReturn(run func(string) bool) *Config_IsSet_Call {
	_c.Call.Return(run)
	return _c
}

// UnmarshalDefaults provides a mock function with given fields: val, additionalDefaults
func (_m *Config) UnmarshalDefaults(val interface{}, additionalDefaults ...cfg.UnmarshalDefaults) error {
	_va := make([]interface{}, len(additionalDefaults))
	for _i := range additionalDefaults {
		_va[_i] = additionalDefaults[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, val)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for UnmarshalDefaults")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}, ...cfg.UnmarshalDefaults) error); ok {
		r0 = rf(val, additionalDefaults...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Config_UnmarshalDefaults_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UnmarshalDefaults'
type Config_UnmarshalDefaults_Call struct {
	*mock.Call
}

// UnmarshalDefaults is a helper method to define mock.On call
//   - val interface{}
//   - additionalDefaults ...cfg.UnmarshalDefaults
func (_e *Config_Expecter) UnmarshalDefaults(val interface{}, additionalDefaults ...interface{}) *Config_UnmarshalDefaults_Call {
	return &Config_UnmarshalDefaults_Call{Call: _e.mock.On("UnmarshalDefaults",
		append([]interface{}{val}, additionalDefaults...)...)}
}

func (_c *Config_UnmarshalDefaults_Call) Run(run func(val interface{}, additionalDefaults ...cfg.UnmarshalDefaults)) *Config_UnmarshalDefaults_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]cfg.UnmarshalDefaults, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(cfg.UnmarshalDefaults)
			}
		}
		run(args[0].(interface{}), variadicArgs...)
	})
	return _c
}

func (_c *Config_UnmarshalDefaults_Call) Return(_a0 error) *Config_UnmarshalDefaults_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Config_UnmarshalDefaults_Call) RunAndReturn(run func(interface{}, ...cfg.UnmarshalDefaults) error) *Config_UnmarshalDefaults_Call {
	_c.Call.Return(run)
	return _c
}

// UnmarshalKey provides a mock function with given fields: key, val, additionalDefaults
func (_m *Config) UnmarshalKey(key string, val interface{}, additionalDefaults ...cfg.UnmarshalDefaults) error {
	_va := make([]interface{}, len(additionalDefaults))
	for _i := range additionalDefaults {
		_va[_i] = additionalDefaults[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key, val)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for UnmarshalKey")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, interface{}, ...cfg.UnmarshalDefaults) error); ok {
		r0 = rf(key, val, additionalDefaults...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Config_UnmarshalKey_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UnmarshalKey'
type Config_UnmarshalKey_Call struct {
	*mock.Call
}

// UnmarshalKey is a helper method to define mock.On call
//   - key string
//   - val interface{}
//   - additionalDefaults ...cfg.UnmarshalDefaults
func (_e *Config_Expecter) UnmarshalKey(key interface{}, val interface{}, additionalDefaults ...interface{}) *Config_UnmarshalKey_Call {
	return &Config_UnmarshalKey_Call{Call: _e.mock.On("UnmarshalKey",
		append([]interface{}{key, val}, additionalDefaults...)...)}
}

func (_c *Config_UnmarshalKey_Call) Run(run func(key string, val interface{}, additionalDefaults ...cfg.UnmarshalDefaults)) *Config_UnmarshalKey_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]cfg.UnmarshalDefaults, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(cfg.UnmarshalDefaults)
			}
		}
		run(args[0].(string), args[1].(interface{}), variadicArgs...)
	})
	return _c
}

func (_c *Config_UnmarshalKey_Call) Return(_a0 error) *Config_UnmarshalKey_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Config_UnmarshalKey_Call) RunAndReturn(run func(string, interface{}, ...cfg.UnmarshalDefaults) error) *Config_UnmarshalKey_Call {
	_c.Call.Return(run)
	return _c
}

// NewConfig creates a new instance of Config. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewConfig(t interface {
	mock.TestingT
	Cleanup(func())
}) *Config {
	mock := &Config{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
