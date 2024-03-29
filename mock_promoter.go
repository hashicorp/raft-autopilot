// Code generated by mockery v2.14.0. DO NOT EDIT.

package autopilot

import (
	raft "github.com/hashicorp/raft"
	mock "github.com/stretchr/testify/mock"
)

// MockPromoter is an autogenerated mock type for the Promoter type
type MockPromoter struct {
	mock.Mock
}

// CalculatePromotionsAndDemotions provides a mock function with given fields: _a0, _a1
func (_m *MockPromoter) CalculatePromotionsAndDemotions(_a0 *Config, _a1 *State) RaftChanges {
	ret := _m.Called(_a0, _a1)

	var r0 RaftChanges
	if rf, ok := ret.Get(0).(func(*Config, *State) RaftChanges); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(RaftChanges)
	}

	return r0
}

// FilterFailedServerRemovals provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockPromoter) FilterFailedServerRemovals(_a0 *Config, _a1 *State, _a2 *FailedServers) *FailedServers {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *FailedServers
	if rf, ok := ret.Get(0).(func(*Config, *State, *FailedServers) *FailedServers); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*FailedServers)
		}
	}

	return r0
}

// GetNodeTypes provides a mock function with given fields: _a0, _a1
func (_m *MockPromoter) GetNodeTypes(_a0 *Config, _a1 *State) map[raft.ServerID]NodeType {
	ret := _m.Called(_a0, _a1)

	var r0 map[raft.ServerID]NodeType
	if rf, ok := ret.Get(0).(func(*Config, *State) map[raft.ServerID]NodeType); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[raft.ServerID]NodeType)
		}
	}

	return r0
}

// GetServerExt provides a mock function with given fields: _a0, _a1
func (_m *MockPromoter) GetServerExt(_a0 *Config, _a1 *ServerState) interface{} {
	ret := _m.Called(_a0, _a1)

	var r0 interface{}
	if rf, ok := ret.Get(0).(func(*Config, *ServerState) interface{}); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	return r0
}

// GetStateExt provides a mock function with given fields: _a0, _a1
func (_m *MockPromoter) GetStateExt(_a0 *Config, _a1 *State) interface{} {
	ret := _m.Called(_a0, _a1)

	var r0 interface{}
	if rf, ok := ret.Get(0).(func(*Config, *State) interface{}); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	return r0
}

// IsPotentialVoter provides a mock function with given fields: _a0
func (_m *MockPromoter) IsPotentialVoter(_a0 NodeType) bool {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(NodeType) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

type mockConstructorTestingTNewMockPromoter interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockPromoter creates a new instance of MockPromoter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockPromoter(t mockConstructorTestingTNewMockPromoter) *MockPromoter {
	mock := &MockPromoter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
