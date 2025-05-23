// Code generated by mockery v2.53.3. DO NOT EDIT.

package mock_hook

import mock "github.com/stretchr/testify/mock"

// MockEncryptor is an autogenerated mock type for the Encryptor type
type MockEncryptor struct {
	mock.Mock
}

type MockEncryptor_Expecter struct {
	mock *mock.Mock
}

func (_m *MockEncryptor) EXPECT() *MockEncryptor_Expecter {
	return &MockEncryptor_Expecter{mock: &_m.Mock}
}

// Encrypt provides a mock function with given fields: plainText
func (_m *MockEncryptor) Encrypt(plainText []byte) ([]byte, error) {
	ret := _m.Called(plainText)

	if len(ret) == 0 {
		panic("no return value specified for Encrypt")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) ([]byte, error)); ok {
		return rf(plainText)
	}
	if rf, ok := ret.Get(0).(func([]byte) []byte); ok {
		r0 = rf(plainText)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(plainText)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockEncryptor_Encrypt_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Encrypt'
type MockEncryptor_Encrypt_Call struct {
	*mock.Call
}

// Encrypt is a helper method to define mock.On call
//   - plainText []byte
func (_e *MockEncryptor_Expecter) Encrypt(plainText interface{}) *MockEncryptor_Encrypt_Call {
	return &MockEncryptor_Encrypt_Call{Call: _e.mock.On("Encrypt", plainText)}
}

func (_c *MockEncryptor_Encrypt_Call) Run(run func(plainText []byte)) *MockEncryptor_Encrypt_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]byte))
	})
	return _c
}

func (_c *MockEncryptor_Encrypt_Call) Return(cipherText []byte, err error) *MockEncryptor_Encrypt_Call {
	_c.Call.Return(cipherText, err)
	return _c
}

func (_c *MockEncryptor_Encrypt_Call) RunAndReturn(run func([]byte) ([]byte, error)) *MockEncryptor_Encrypt_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockEncryptor creates a new instance of MockEncryptor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockEncryptor(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockEncryptor {
	mock := &MockEncryptor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
