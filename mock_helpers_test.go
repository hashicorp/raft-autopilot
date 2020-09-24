package autopilot

import (
	"testing"

	"github.com/hashicorp/raft"
)

func NewMockTimeProvider(t *testing.T) *mockTimeProvider {
	t.Helper()
	mtime := mockTimeProvider{}
	t.Cleanup(func() { mtime.AssertExpectations(t) })
	return &mtime
}

func NewMockRaft(t *testing.T) *MockRaft {
	t.Helper()
	mraft := MockRaft{}
	t.Cleanup(func() { mraft.AssertExpectations(t) })
	return &mraft
}

type raftIndexFuture struct {
	index uint64
	err   error
}

func (f *raftIndexFuture) Index() uint64 {
	return f.index
}

func (f *raftIndexFuture) Error() error {
	return f.err
}

type raftConfigFuture struct {
	config raft.Configuration
	index  uint64
	err    error
}

func (f *raftConfigFuture) Index() uint64 {
	return f.index
}

func (f *raftConfigFuture) Error() error {
	return f.err
}

func (f *raftConfigFuture) Configuration() raft.Configuration {
	return f.config
}

func NewMockApplicationIntegration(t *testing.T) *MockApplicationIntegration {
	t.Helper()
	mdel := MockApplicationIntegration{}
	t.Cleanup(func() { mdel.AssertExpectations(t) })
	return &mdel
}

func NewMockPromoter(t *testing.T) *MockPromoter {
	t.Helper()
	mpromoter := MockPromoter{}
	t.Cleanup(func() { mpromoter.AssertExpectations(t) })
	return &mpromoter
}
