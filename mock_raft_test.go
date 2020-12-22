// Code generated by mockery v1.0.0. DO NOT EDIT.

package autopilot

import (
	raft "github.com/hashicorp/raft"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// MockRaft is an autogenerated mock type for the Raft type
type MockRaft struct {
	mock.Mock
}

// AddNonvoter provides a mock function with given fields: id, address, prevIndex, timeout
func (_m *MockRaft) AddNonvoter(id raft.ServerID, address raft.ServerAddress, prevIndex uint64, timeout time.Duration) raft.IndexFuture {
	ret := _m.Called(id, address, prevIndex, timeout)

	var r0 raft.IndexFuture
	if rf, ok := ret.Get(0).(func(raft.ServerID, raft.ServerAddress, uint64, time.Duration) raft.IndexFuture); ok {
		r0 = rf(id, address, prevIndex, timeout)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(raft.IndexFuture)
		}
	}

	return r0
}

// AddVoter provides a mock function with given fields: id, address, prevIndex, timeout
func (_m *MockRaft) AddVoter(id raft.ServerID, address raft.ServerAddress, prevIndex uint64, timeout time.Duration) raft.IndexFuture {
	ret := _m.Called(id, address, prevIndex, timeout)

	var r0 raft.IndexFuture
	if rf, ok := ret.Get(0).(func(raft.ServerID, raft.ServerAddress, uint64, time.Duration) raft.IndexFuture); ok {
		r0 = rf(id, address, prevIndex, timeout)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(raft.IndexFuture)
		}
	}

	return r0
}

// DemoteVoter provides a mock function with given fields: id, prevIndex, timeout
func (_m *MockRaft) DemoteVoter(id raft.ServerID, prevIndex uint64, timeout time.Duration) raft.IndexFuture {
	ret := _m.Called(id, prevIndex, timeout)

	var r0 raft.IndexFuture
	if rf, ok := ret.Get(0).(func(raft.ServerID, uint64, time.Duration) raft.IndexFuture); ok {
		r0 = rf(id, prevIndex, timeout)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(raft.IndexFuture)
		}
	}

	return r0
}

// GetConfiguration provides a mock function with given fields:
func (_m *MockRaft) GetConfiguration() raft.ConfigurationFuture {
	ret := _m.Called()

	var r0 raft.ConfigurationFuture
	if rf, ok := ret.Get(0).(func() raft.ConfigurationFuture); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(raft.ConfigurationFuture)
		}
	}

	return r0
}

// LastIndex provides a mock function with given fields:
func (_m *MockRaft) LastIndex() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// Leader provides a mock function with given fields:
func (_m *MockRaft) Leader() raft.ServerAddress {
	ret := _m.Called()

	var r0 raft.ServerAddress
	if rf, ok := ret.Get(0).(func() raft.ServerAddress); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(raft.ServerAddress)
	}

	return r0
}

// LeadershipTransferToServer provides a mock function with given fields: id, address
func (_m *MockRaft) LeadershipTransferToServer(id raft.ServerID, address raft.ServerAddress) raft.Future {
	ret := _m.Called(id, address)

	var r0 raft.Future
	if rf, ok := ret.Get(0).(func(raft.ServerID, raft.ServerAddress) raft.Future); ok {
		r0 = rf(id, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(raft.Future)
		}
	}

	return r0
}

// RemoveServer provides a mock function with given fields: id, prevIndex, timeout
func (_m *MockRaft) RemoveServer(id raft.ServerID, prevIndex uint64, timeout time.Duration) raft.IndexFuture {
	ret := _m.Called(id, prevIndex, timeout)

	var r0 raft.IndexFuture
	if rf, ok := ret.Get(0).(func(raft.ServerID, uint64, time.Duration) raft.IndexFuture); ok {
		r0 = rf(id, prevIndex, timeout)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(raft.IndexFuture)
		}
	}

	return r0
}

// Stats provides a mock function with given fields:
func (_m *MockRaft) Stats() map[string]string {
	ret := _m.Called()

	var r0 map[string]string
	if rf, ok := ret.Get(0).(func() map[string]string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]string)
		}
	}

	return r0
}