// Code generated by mockery v2.32.4. DO NOT EDIT.

package mock_wal

import (
	context "context"

	logpb "github.com/milvus-io/milvus/internal/proto/logpb"
	message "github.com/milvus-io/milvus/internal/util/logserviceutil/message"

	mock "github.com/stretchr/testify/mock"

	mqwrapper "github.com/milvus-io/milvus/pkg/mq/msgstream/mqwrapper"

	wal "github.com/milvus-io/milvus/internal/lognode/server/wal"
)

// MockBasicWAL is an autogenerated mock type for the BasicWAL type
type MockBasicWAL struct {
	mock.Mock
}

type MockBasicWAL_Expecter struct {
	mock *mock.Mock
}

func (_m *MockBasicWAL) EXPECT() *MockBasicWAL_Expecter {
	return &MockBasicWAL_Expecter{mock: &_m.Mock}
}

// Append provides a mock function with given fields: ctx, msg
func (_m *MockBasicWAL) Append(ctx context.Context, msg message.MutableMessage) (mqwrapper.MessageID, error) {
	ret := _m.Called(ctx, msg)

	var r0 mqwrapper.MessageID
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, message.MutableMessage) (mqwrapper.MessageID, error)); ok {
		return rf(ctx, msg)
	}
	if rf, ok := ret.Get(0).(func(context.Context, message.MutableMessage) mqwrapper.MessageID); ok {
		r0 = rf(ctx, msg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(mqwrapper.MessageID)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, message.MutableMessage) error); ok {
		r1 = rf(ctx, msg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockBasicWAL_Append_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Append'
type MockBasicWAL_Append_Call struct {
	*mock.Call
}

// Append is a helper method to define mock.On call
//   - ctx context.Context
//   - msg message.MutableMessage
func (_e *MockBasicWAL_Expecter) Append(ctx interface{}, msg interface{}) *MockBasicWAL_Append_Call {
	return &MockBasicWAL_Append_Call{Call: _e.mock.On("Append", ctx, msg)}
}

func (_c *MockBasicWAL_Append_Call) Run(run func(ctx context.Context, msg message.MutableMessage)) *MockBasicWAL_Append_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(message.MutableMessage))
	})
	return _c
}

func (_c *MockBasicWAL_Append_Call) Return(_a0 mqwrapper.MessageID, _a1 error) *MockBasicWAL_Append_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockBasicWAL_Append_Call) RunAndReturn(run func(context.Context, message.MutableMessage) (mqwrapper.MessageID, error)) *MockBasicWAL_Append_Call {
	_c.Call.Return(run)
	return _c
}

// Channel provides a mock function with given fields:
func (_m *MockBasicWAL) Channel() *logpb.PChannelInfo {
	ret := _m.Called()

	var r0 *logpb.PChannelInfo
	if rf, ok := ret.Get(0).(func() *logpb.PChannelInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*logpb.PChannelInfo)
		}
	}

	return r0
}

// MockBasicWAL_Channel_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Channel'
type MockBasicWAL_Channel_Call struct {
	*mock.Call
}

// Channel is a helper method to define mock.On call
func (_e *MockBasicWAL_Expecter) Channel() *MockBasicWAL_Channel_Call {
	return &MockBasicWAL_Channel_Call{Call: _e.mock.On("Channel")}
}

func (_c *MockBasicWAL_Channel_Call) Run(run func()) *MockBasicWAL_Channel_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockBasicWAL_Channel_Call) Return(_a0 *logpb.PChannelInfo) *MockBasicWAL_Channel_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockBasicWAL_Channel_Call) RunAndReturn(run func() *logpb.PChannelInfo) *MockBasicWAL_Channel_Call {
	_c.Call.Return(run)
	return _c
}

// Close provides a mock function with given fields:
func (_m *MockBasicWAL) Close() {
	_m.Called()
}

// MockBasicWAL_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type MockBasicWAL_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *MockBasicWAL_Expecter) Close() *MockBasicWAL_Close_Call {
	return &MockBasicWAL_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *MockBasicWAL_Close_Call) Run(run func()) *MockBasicWAL_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockBasicWAL_Close_Call) Return() *MockBasicWAL_Close_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockBasicWAL_Close_Call) RunAndReturn(run func()) *MockBasicWAL_Close_Call {
	_c.Call.Return(run)
	return _c
}

// GetLatestMessageID provides a mock function with given fields: ctx
func (_m *MockBasicWAL) GetLatestMessageID(ctx context.Context) (mqwrapper.MessageID, error) {
	ret := _m.Called(ctx)

	var r0 mqwrapper.MessageID
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (mqwrapper.MessageID, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) mqwrapper.MessageID); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(mqwrapper.MessageID)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockBasicWAL_GetLatestMessageID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLatestMessageID'
type MockBasicWAL_GetLatestMessageID_Call struct {
	*mock.Call
}

// GetLatestMessageID is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockBasicWAL_Expecter) GetLatestMessageID(ctx interface{}) *MockBasicWAL_GetLatestMessageID_Call {
	return &MockBasicWAL_GetLatestMessageID_Call{Call: _e.mock.On("GetLatestMessageID", ctx)}
}

func (_c *MockBasicWAL_GetLatestMessageID_Call) Run(run func(ctx context.Context)) *MockBasicWAL_GetLatestMessageID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockBasicWAL_GetLatestMessageID_Call) Return(_a0 mqwrapper.MessageID, _a1 error) *MockBasicWAL_GetLatestMessageID_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockBasicWAL_GetLatestMessageID_Call) RunAndReturn(run func(context.Context) (mqwrapper.MessageID, error)) *MockBasicWAL_GetLatestMessageID_Call {
	_c.Call.Return(run)
	return _c
}

// Read provides a mock function with given fields: ctx, deliverPolicy
func (_m *MockBasicWAL) Read(ctx context.Context, deliverPolicy wal.ReadOption) (wal.Scanner, error) {
	ret := _m.Called(ctx, deliverPolicy)

	var r0 wal.Scanner
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, wal.ReadOption) (wal.Scanner, error)); ok {
		return rf(ctx, deliverPolicy)
	}
	if rf, ok := ret.Get(0).(func(context.Context, wal.ReadOption) wal.Scanner); ok {
		r0 = rf(ctx, deliverPolicy)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(wal.Scanner)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, wal.ReadOption) error); ok {
		r1 = rf(ctx, deliverPolicy)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockBasicWAL_Read_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Read'
type MockBasicWAL_Read_Call struct {
	*mock.Call
}

// Read is a helper method to define mock.On call
//   - ctx context.Context
//   - deliverPolicy wal.ReadOption
func (_e *MockBasicWAL_Expecter) Read(ctx interface{}, deliverPolicy interface{}) *MockBasicWAL_Read_Call {
	return &MockBasicWAL_Read_Call{Call: _e.mock.On("Read", ctx, deliverPolicy)}
}

func (_c *MockBasicWAL_Read_Call) Run(run func(ctx context.Context, deliverPolicy wal.ReadOption)) *MockBasicWAL_Read_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(wal.ReadOption))
	})
	return _c
}

func (_c *MockBasicWAL_Read_Call) Return(_a0 wal.Scanner, _a1 error) *MockBasicWAL_Read_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockBasicWAL_Read_Call) RunAndReturn(run func(context.Context, wal.ReadOption) (wal.Scanner, error)) *MockBasicWAL_Read_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockBasicWAL creates a new instance of MockBasicWAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockBasicWAL(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockBasicWAL {
	mock := &MockBasicWAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}