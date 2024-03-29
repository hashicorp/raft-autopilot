// Code generated by mockery v2.14.0. DO NOT EDIT.

package autopilot

import mock "github.com/stretchr/testify/mock"

// MockOption is an autogenerated mock type for the Option type
type MockOption struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *MockOption) Execute(_a0 *Autopilot) {
	_m.Called(_a0)
}

type mockConstructorTestingTNewMockOption interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockOption creates a new instance of MockOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockOption(t mockConstructorTestingTNewMockOption) *MockOption {
	mock := &MockOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
