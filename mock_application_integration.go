// Code generated by mockery v2.14.0. DO NOT EDIT.

package autopilot

import (
	context "context"

	raft "github.com/hashicorp/raft"
	mock "github.com/stretchr/testify/mock"
)

// MockApplicationIntegration is an autogenerated mock type for the ApplicationIntegration type
type MockApplicationIntegration struct {
	mock.Mock
}

// AutopilotConfig provides a mock function with given fields:
func (_m *MockApplicationIntegration) AutopilotConfig() *Config {
	ret := _m.Called()

	var r0 *Config
	if rf, ok := ret.Get(0).(func() *Config); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Config)
		}
	}

	return r0
}

// FetchServerStats provides a mock function with given fields: _a0, _a1
func (_m *MockApplicationIntegration) FetchServerStats(_a0 context.Context, _a1 map[raft.ServerID]*Server) map[raft.ServerID]*ServerStats {
	ret := _m.Called(_a0, _a1)

	var r0 map[raft.ServerID]*ServerStats
	if rf, ok := ret.Get(0).(func(context.Context, map[raft.ServerID]*Server) map[raft.ServerID]*ServerStats); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[raft.ServerID]*ServerStats)
		}
	}

	return r0
}

// KnownServers provides a mock function with given fields:
func (_m *MockApplicationIntegration) KnownServers() map[raft.ServerID]*Server {
	ret := _m.Called()

	var r0 map[raft.ServerID]*Server
	if rf, ok := ret.Get(0).(func() map[raft.ServerID]*Server); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[raft.ServerID]*Server)
		}
	}

	return r0
}

// NotifyState provides a mock function with given fields: _a0
func (_m *MockApplicationIntegration) NotifyState(_a0 *State) {
	_m.Called(_a0)
}

// RemoveFailedServer provides a mock function with given fields: _a0
func (_m *MockApplicationIntegration) RemoveFailedServer(_a0 *Server) {
	_m.Called(_a0)
}

type mockConstructorTestingTNewMockApplicationIntegration interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockApplicationIntegration creates a new instance of MockApplicationIntegration. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockApplicationIntegration(t mockConstructorTestingTNewMockApplicationIntegration) *MockApplicationIntegration {
	mock := &MockApplicationIntegration{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
