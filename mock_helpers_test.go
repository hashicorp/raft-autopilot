// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package autopilot

import (
	"github.com/hashicorp/raft"
)

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
