// Code generated by mockery v2.32.4. DO NOT EDIT.

package syncmgr

import (
	context "context"

	conc "github.com/milvus-io/milvus/pkg/util/conc"

	mock "github.com/stretchr/testify/mock"

	msgpb "github.com/milvus-io/milvus-proto/go-api/v2/msgpb"
)

// MockSyncManager is an autogenerated mock type for the SyncManager type
type MockSyncManager struct {
	mock.Mock
}

type MockSyncManager_Expecter struct {
	mock *mock.Mock
}

func (_m *MockSyncManager) EXPECT() *MockSyncManager_Expecter {
	return &MockSyncManager_Expecter{mock: &_m.Mock}
}

// GetEarliestPosition provides a mock function with given fields: channel
func (_m *MockSyncManager) GetEarliestPosition(channel string) (int64, *msgpb.MsgPosition) {
	ret := _m.Called(channel)

	var r0 int64
	var r1 *msgpb.MsgPosition
	if rf, ok := ret.Get(0).(func(string) (int64, *msgpb.MsgPosition)); ok {
		return rf(channel)
	}
	if rf, ok := ret.Get(0).(func(string) int64); ok {
		r0 = rf(channel)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(string) *msgpb.MsgPosition); ok {
		r1 = rf(channel)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*msgpb.MsgPosition)
		}
	}

	return r0, r1
}

// MockSyncManager_GetEarliestPosition_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetEarliestPosition'
type MockSyncManager_GetEarliestPosition_Call struct {
	*mock.Call
}

// GetEarliestPosition is a helper method to define mock.On call
//   - channel string
func (_e *MockSyncManager_Expecter) GetEarliestPosition(channel interface{}) *MockSyncManager_GetEarliestPosition_Call {
	return &MockSyncManager_GetEarliestPosition_Call{Call: _e.mock.On("GetEarliestPosition", channel)}
}

func (_c *MockSyncManager_GetEarliestPosition_Call) Run(run func(channel string)) *MockSyncManager_GetEarliestPosition_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockSyncManager_GetEarliestPosition_Call) Return(_a0 int64, _a1 *msgpb.MsgPosition) *MockSyncManager_GetEarliestPosition_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSyncManager_GetEarliestPosition_Call) RunAndReturn(run func(string) (int64, *msgpb.MsgPosition)) *MockSyncManager_GetEarliestPosition_Call {
	_c.Call.Return(run)
	return _c
}

// SyncData provides a mock function with given fields: ctx, task
func (_m *MockSyncManager) SyncData(ctx context.Context, task Task) *conc.Future[struct{}] {
	ret := _m.Called(ctx, task)

	var r0 *conc.Future[struct{}]
	if rf, ok := ret.Get(0).(func(context.Context, Task) *conc.Future[struct{}]); ok {
		r0 = rf(ctx, task)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*conc.Future[struct{}])
		}
	}

	return r0
}

// MockSyncManager_SyncData_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SyncData'
type MockSyncManager_SyncData_Call struct {
	*mock.Call
}

// SyncData is a helper method to define mock.On call
//   - ctx context.Context
//   - task Task
func (_e *MockSyncManager_Expecter) SyncData(ctx interface{}, task interface{}) *MockSyncManager_SyncData_Call {
	return &MockSyncManager_SyncData_Call{Call: _e.mock.On("SyncData", ctx, task)}
}

func (_c *MockSyncManager_SyncData_Call) Run(run func(ctx context.Context, task Task)) *MockSyncManager_SyncData_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(Task))
	})
	return _c
}

func (_c *MockSyncManager_SyncData_Call) Return(_a0 *conc.Future[struct{}]) *MockSyncManager_SyncData_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockSyncManager_SyncData_Call) RunAndReturn(run func(context.Context, Task) *conc.Future[struct{}]) *MockSyncManager_SyncData_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockSyncManager creates a new instance of MockSyncManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSyncManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSyncManager {
	mock := &MockSyncManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}