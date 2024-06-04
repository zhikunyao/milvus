// Code generated by mockery v2.32.4. DO NOT EDIT.

package mock_resolver

import (
	mock "github.com/stretchr/testify/mock"
	resolver "google.golang.org/grpc/resolver"
)

// MockResolver is an autogenerated mock type for the Resolver type
type MockResolver struct {
	mock.Mock
}

type MockResolver_Expecter struct {
	mock *mock.Mock
}

func (_m *MockResolver) EXPECT() *MockResolver_Expecter {
	return &MockResolver_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with given fields:
func (_m *MockResolver) Close() {
	_m.Called()
}

// MockResolver_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type MockResolver_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *MockResolver_Expecter) Close() *MockResolver_Close_Call {
	return &MockResolver_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *MockResolver_Close_Call) Run(run func()) *MockResolver_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockResolver_Close_Call) Return() *MockResolver_Close_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockResolver_Close_Call) RunAndReturn(run func()) *MockResolver_Close_Call {
	_c.Call.Return(run)
	return _c
}

// ResolveNow provides a mock function with given fields: _a0
func (_m *MockResolver) ResolveNow(_a0 resolver.ResolveNowOptions) {
	_m.Called(_a0)
}

// MockResolver_ResolveNow_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ResolveNow'
type MockResolver_ResolveNow_Call struct {
	*mock.Call
}

// ResolveNow is a helper method to define mock.On call
//   - _a0 resolver.ResolveNowOptions
func (_e *MockResolver_Expecter) ResolveNow(_a0 interface{}) *MockResolver_ResolveNow_Call {
	return &MockResolver_ResolveNow_Call{Call: _e.mock.On("ResolveNow", _a0)}
}

func (_c *MockResolver_ResolveNow_Call) Run(run func(_a0 resolver.ResolveNowOptions)) *MockResolver_ResolveNow_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(resolver.ResolveNowOptions))
	})
	return _c
}

func (_c *MockResolver_ResolveNow_Call) Return() *MockResolver_ResolveNow_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockResolver_ResolveNow_Call) RunAndReturn(run func(resolver.ResolveNowOptions)) *MockResolver_ResolveNow_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockResolver creates a new instance of MockResolver. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockResolver(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockResolver {
	mock := &MockResolver{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}