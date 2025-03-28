// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mdl "github.com/justtrackio/gosoline/pkg/mdl"
	mdlsub "github.com/justtrackio/gosoline/pkg/mdlsub"

	mock "github.com/stretchr/testify/mock"
)

// SubscriberCore is an autogenerated mock type for the SubscriberCore type
type SubscriberCore struct {
	mock.Mock
}

type SubscriberCore_Expecter struct {
	mock *mock.Mock
}

func (_m *SubscriberCore) EXPECT() *SubscriberCore_Expecter {
	return &SubscriberCore_Expecter{mock: &_m.Mock}
}

// GetLatestModelIdVersion provides a mock function with given fields: modelId
func (_m *SubscriberCore) GetLatestModelIdVersion(modelId mdl.ModelId) (int, error) {
	ret := _m.Called(modelId)

	if len(ret) == 0 {
		panic("no return value specified for GetLatestModelIdVersion")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(mdl.ModelId) (int, error)); ok {
		return rf(modelId)
	}
	if rf, ok := ret.Get(0).(func(mdl.ModelId) int); ok {
		r0 = rf(modelId)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(mdl.ModelId) error); ok {
		r1 = rf(modelId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SubscriberCore_GetLatestModelIdVersion_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLatestModelIdVersion'
type SubscriberCore_GetLatestModelIdVersion_Call struct {
	*mock.Call
}

// GetLatestModelIdVersion is a helper method to define mock.On call
//   - modelId mdl.ModelId
func (_e *SubscriberCore_Expecter) GetLatestModelIdVersion(modelId interface{}) *SubscriberCore_GetLatestModelIdVersion_Call {
	return &SubscriberCore_GetLatestModelIdVersion_Call{Call: _e.mock.On("GetLatestModelIdVersion", modelId)}
}

func (_c *SubscriberCore_GetLatestModelIdVersion_Call) Run(run func(modelId mdl.ModelId)) *SubscriberCore_GetLatestModelIdVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(mdl.ModelId))
	})
	return _c
}

func (_c *SubscriberCore_GetLatestModelIdVersion_Call) Return(_a0 int, _a1 error) *SubscriberCore_GetLatestModelIdVersion_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SubscriberCore_GetLatestModelIdVersion_Call) RunAndReturn(run func(mdl.ModelId) (int, error)) *SubscriberCore_GetLatestModelIdVersion_Call {
	_c.Call.Return(run)
	return _c
}

// GetModelIds provides a mock function with no fields
func (_m *SubscriberCore) GetModelIds() []string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetModelIds")
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

// SubscriberCore_GetModelIds_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetModelIds'
type SubscriberCore_GetModelIds_Call struct {
	*mock.Call
}

// GetModelIds is a helper method to define mock.On call
func (_e *SubscriberCore_Expecter) GetModelIds() *SubscriberCore_GetModelIds_Call {
	return &SubscriberCore_GetModelIds_Call{Call: _e.mock.On("GetModelIds")}
}

func (_c *SubscriberCore_GetModelIds_Call) Run(run func()) *SubscriberCore_GetModelIds_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *SubscriberCore_GetModelIds_Call) Return(_a0 []string) *SubscriberCore_GetModelIds_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SubscriberCore_GetModelIds_Call) RunAndReturn(run func() []string) *SubscriberCore_GetModelIds_Call {
	_c.Call.Return(run)
	return _c
}

// GetOutput provides a mock function with given fields: spec
func (_m *SubscriberCore) GetOutput(spec *mdlsub.ModelSpecification) (mdlsub.Output, error) {
	ret := _m.Called(spec)

	if len(ret) == 0 {
		panic("no return value specified for GetOutput")
	}

	var r0 mdlsub.Output
	var r1 error
	if rf, ok := ret.Get(0).(func(*mdlsub.ModelSpecification) (mdlsub.Output, error)); ok {
		return rf(spec)
	}
	if rf, ok := ret.Get(0).(func(*mdlsub.ModelSpecification) mdlsub.Output); ok {
		r0 = rf(spec)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(mdlsub.Output)
		}
	}

	if rf, ok := ret.Get(1).(func(*mdlsub.ModelSpecification) error); ok {
		r1 = rf(spec)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SubscriberCore_GetOutput_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetOutput'
type SubscriberCore_GetOutput_Call struct {
	*mock.Call
}

// GetOutput is a helper method to define mock.On call
//   - spec *mdlsub.ModelSpecification
func (_e *SubscriberCore_Expecter) GetOutput(spec interface{}) *SubscriberCore_GetOutput_Call {
	return &SubscriberCore_GetOutput_Call{Call: _e.mock.On("GetOutput", spec)}
}

func (_c *SubscriberCore_GetOutput_Call) Run(run func(spec *mdlsub.ModelSpecification)) *SubscriberCore_GetOutput_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*mdlsub.ModelSpecification))
	})
	return _c
}

func (_c *SubscriberCore_GetOutput_Call) Return(_a0 mdlsub.Output, _a1 error) *SubscriberCore_GetOutput_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SubscriberCore_GetOutput_Call) RunAndReturn(run func(*mdlsub.ModelSpecification) (mdlsub.Output, error)) *SubscriberCore_GetOutput_Call {
	_c.Call.Return(run)
	return _c
}

// GetTransformer provides a mock function with given fields: spec
func (_m *SubscriberCore) GetTransformer(spec *mdlsub.ModelSpecification) (mdlsub.ModelTransformer, error) {
	ret := _m.Called(spec)

	if len(ret) == 0 {
		panic("no return value specified for GetTransformer")
	}

	var r0 mdlsub.ModelTransformer
	var r1 error
	if rf, ok := ret.Get(0).(func(*mdlsub.ModelSpecification) (mdlsub.ModelTransformer, error)); ok {
		return rf(spec)
	}
	if rf, ok := ret.Get(0).(func(*mdlsub.ModelSpecification) mdlsub.ModelTransformer); ok {
		r0 = rf(spec)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(mdlsub.ModelTransformer)
		}
	}

	if rf, ok := ret.Get(1).(func(*mdlsub.ModelSpecification) error); ok {
		r1 = rf(spec)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SubscriberCore_GetTransformer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetTransformer'
type SubscriberCore_GetTransformer_Call struct {
	*mock.Call
}

// GetTransformer is a helper method to define mock.On call
//   - spec *mdlsub.ModelSpecification
func (_e *SubscriberCore_Expecter) GetTransformer(spec interface{}) *SubscriberCore_GetTransformer_Call {
	return &SubscriberCore_GetTransformer_Call{Call: _e.mock.On("GetTransformer", spec)}
}

func (_c *SubscriberCore_GetTransformer_Call) Run(run func(spec *mdlsub.ModelSpecification)) *SubscriberCore_GetTransformer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*mdlsub.ModelSpecification))
	})
	return _c
}

func (_c *SubscriberCore_GetTransformer_Call) Return(_a0 mdlsub.ModelTransformer, _a1 error) *SubscriberCore_GetTransformer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SubscriberCore_GetTransformer_Call) RunAndReturn(run func(*mdlsub.ModelSpecification) (mdlsub.ModelTransformer, error)) *SubscriberCore_GetTransformer_Call {
	_c.Call.Return(run)
	return _c
}

// Persist provides a mock function with given fields: ctx, spec, model
func (_m *SubscriberCore) Persist(ctx context.Context, spec *mdlsub.ModelSpecification, model mdlsub.Model) error {
	ret := _m.Called(ctx, spec, model)

	if len(ret) == 0 {
		panic("no return value specified for Persist")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *mdlsub.ModelSpecification, mdlsub.Model) error); ok {
		r0 = rf(ctx, spec, model)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SubscriberCore_Persist_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Persist'
type SubscriberCore_Persist_Call struct {
	*mock.Call
}

// Persist is a helper method to define mock.On call
//   - ctx context.Context
//   - spec *mdlsub.ModelSpecification
//   - model mdlsub.Model
func (_e *SubscriberCore_Expecter) Persist(ctx interface{}, spec interface{}, model interface{}) *SubscriberCore_Persist_Call {
	return &SubscriberCore_Persist_Call{Call: _e.mock.On("Persist", ctx, spec, model)}
}

func (_c *SubscriberCore_Persist_Call) Run(run func(ctx context.Context, spec *mdlsub.ModelSpecification, model mdlsub.Model)) *SubscriberCore_Persist_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*mdlsub.ModelSpecification), args[2].(mdlsub.Model))
	})
	return _c
}

func (_c *SubscriberCore_Persist_Call) Return(_a0 error) *SubscriberCore_Persist_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SubscriberCore_Persist_Call) RunAndReturn(run func(context.Context, *mdlsub.ModelSpecification, mdlsub.Model) error) *SubscriberCore_Persist_Call {
	_c.Call.Return(run)
	return _c
}

// Transform provides a mock function with given fields: ctx, spec, input
func (_m *SubscriberCore) Transform(ctx context.Context, spec *mdlsub.ModelSpecification, input interface{}) (mdlsub.Model, error) {
	ret := _m.Called(ctx, spec, input)

	if len(ret) == 0 {
		panic("no return value specified for Transform")
	}

	var r0 mdlsub.Model
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *mdlsub.ModelSpecification, interface{}) (mdlsub.Model, error)); ok {
		return rf(ctx, spec, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *mdlsub.ModelSpecification, interface{}) mdlsub.Model); ok {
		r0 = rf(ctx, spec, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(mdlsub.Model)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *mdlsub.ModelSpecification, interface{}) error); ok {
		r1 = rf(ctx, spec, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SubscriberCore_Transform_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Transform'
type SubscriberCore_Transform_Call struct {
	*mock.Call
}

// Transform is a helper method to define mock.On call
//   - ctx context.Context
//   - spec *mdlsub.ModelSpecification
//   - input interface{}
func (_e *SubscriberCore_Expecter) Transform(ctx interface{}, spec interface{}, input interface{}) *SubscriberCore_Transform_Call {
	return &SubscriberCore_Transform_Call{Call: _e.mock.On("Transform", ctx, spec, input)}
}

func (_c *SubscriberCore_Transform_Call) Run(run func(ctx context.Context, spec *mdlsub.ModelSpecification, input interface{})) *SubscriberCore_Transform_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*mdlsub.ModelSpecification), args[2].(interface{}))
	})
	return _c
}

func (_c *SubscriberCore_Transform_Call) Return(_a0 mdlsub.Model, _a1 error) *SubscriberCore_Transform_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SubscriberCore_Transform_Call) RunAndReturn(run func(context.Context, *mdlsub.ModelSpecification, interface{}) (mdlsub.Model, error)) *SubscriberCore_Transform_Call {
	_c.Call.Return(run)
	return _c
}

// NewSubscriberCore creates a new instance of SubscriberCore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSubscriberCore(t interface {
	mock.TestingT
	Cleanup(func())
}) *SubscriberCore {
	mock := &SubscriberCore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
