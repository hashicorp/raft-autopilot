// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package autopilot

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

type serverHealthState bool

const (
	unhealthy serverHealthState = false
	healthy   serverHealthState = true
)

// makeServerState creates a test ServerState and returns a pointer to it.
// The node index argument is a string that should be a single digit,
// used to generate the server ID, name, and address.
func makeServerState(nodeIndex string, nodeState RaftState, healthy serverHealthState) *ServerState {
	return &ServerState{
		Server: Server{
			ID:          raft.ServerID(fmt.Sprintf("%[1]s%[1]s-%[1]s-%[1]s-%[1]s-%[1]s%[1]s%[1]s", strings.Repeat(nodeIndex, 4))),
			Name:        fmt.Sprintf("node%s", nodeIndex),
			Address:     raft.ServerAddress(fmt.Sprintf("198.18.0.%s:8300", nodeIndex)),
			NodeStatus:  NodeAlive,
			Version:     "1.9.0",
			RaftVersion: 3,
		},
		State: nodeState,
		Health: ServerHealth{
			Healthy: bool(healthy),
		},
	}
}

func TestReconcile(t *testing.T) {
	type testCase struct {
		changes           RaftChanges
		state             State
		setupExpectations func(*MockRaft)
	}

	cases := map[string]testCase{
		"ignore-failed-nonvoter-promote-one-and-filter-others-and-demotions": {
			state: State{
				Leader: "11111111-1111-1111-1111-111111111111",
				Servers: map[raft.ServerID]*ServerState{
					"11111111-1111-1111-1111-111111111111": makeServerState("1", RaftLeader, healthy),
					"22222222-2222-2222-2222-222222222222": makeServerState("2", RaftVoter, healthy),
					"33333333-3333-3333-3333-333333333333": makeServerState("3", RaftNonVoter, healthy),
					"44444444-4444-4444-4444-444444444444": makeServerState("4", RaftNonVoter, unhealthy),
					"55555555-5555-5555-5555-555555555555": makeServerState("5", RaftVoter, healthy),
				},
				Voters: []raft.ServerID{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
					"55555555-5555-5555-5555-555555555555",
				},
			},
			changes: RaftChanges{
				Promotions: []raft.ServerID{
					// already is a voter so no config change should be made
					"11111111-1111-1111-1111-111111111111",
					// already is a voter so no config change should be made
					"22222222-2222-2222-2222-222222222222",
					// should actually be promoted
					"33333333-3333-3333-3333-333333333333",
					// is not healthy so we shouldn't promote it
					"44444444-4444-4444-4444-444444444444",
				},
				Demotions: []raft.ServerID{
					// shouldn't actually be done since it is healthy, and we
					// are promoting another server
					"55555555-5555-5555-5555-555555555555",
				},
			},
			setupExpectations: func(m *MockRaft) {
				m.On("AddVoter",
					raft.ServerID("33333333-3333-3333-3333-333333333333"),
					raft.ServerAddress("198.18.0.3:8300"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
			},
		},
		"demotions-and-filter-leader-transfer": {
			state: State{
				Leader: "11111111-1111-1111-1111-111111111111",
				Servers: map[raft.ServerID]*ServerState{
					"11111111-1111-1111-1111-111111111111": makeServerState("1", RaftLeader, healthy),
					"22222222-2222-2222-2222-222222222222": makeServerState("2", RaftVoter, healthy),
					"33333333-3333-3333-3333-333333333333": makeServerState("3", RaftVoter, healthy),
					"44444444-4444-4444-4444-444444444444": makeServerState("4", RaftVoter, healthy),
					"55555555-5555-5555-5555-555555555555": makeServerState("5", RaftNonVoter, healthy),
				},
				Voters: []raft.ServerID{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
					"33333333-3333-3333-3333-333333333333",
					"44444444-4444-4444-4444-444444444444",
				},
			},
			changes: RaftChanges{
				Promotions: []raft.ServerID{
					// already is a voter so no config change should be made
					// this should also not prevent the demotions from happening
					"11111111-1111-1111-1111-111111111111",
				},
				Demotions: []raft.ServerID{
					// already a non-voter so nothing should be done
					"55555555-5555-5555-5555-555555555555",
					// should actually get demoted
					"44444444-4444-4444-4444-444444444444",
				},
				// no leadership transfer should be done because we did demotions
				Leader: "33333333-3333-3333-3333-333333333333",
			},
			setupExpectations: func(m *MockRaft) {
				m.On("DemoteVoter",
					raft.ServerID("44444444-4444-4444-4444-444444444444"),
					uint64(0),
					time.Duration(0)).Return(&raftIndexFuture{}).Once()
			},
		},
		"leader-transfer-same-leader": {
			state: State{
				Leader: "11111111-1111-1111-1111-111111111111",
				Servers: map[raft.ServerID]*ServerState{
					"11111111-1111-1111-1111-111111111111": makeServerState("1", RaftLeader, healthy),
					"22222222-2222-2222-2222-222222222222": makeServerState("2", RaftVoter, healthy),
					"33333333-3333-3333-3333-333333333333": makeServerState("3", RaftVoter, healthy),
					"55555555-5555-5555-5555-555555555555": makeServerState("5", RaftNonVoter, healthy),
				},
				Voters: []raft.ServerID{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
					"33333333-3333-3333-3333-333333333333",
				},
			},
			changes: RaftChanges{
				Demotions: []raft.ServerID{
					// already a non-voter so nothing should be done
					"55555555-5555-5555-5555-555555555555",
				},
				// no leadership transfer should be done because we did demotions
				Leader: "11111111-1111-1111-1111-111111111111",
			},
		},
		"leader-transfer": {
			state: State{
				Leader: "11111111-1111-1111-1111-111111111111",
				Servers: map[raft.ServerID]*ServerState{
					"11111111-1111-1111-1111-111111111111": makeServerState("1", RaftLeader, healthy),
					"22222222-2222-2222-2222-222222222222": makeServerState("2", RaftVoter, healthy),
					"33333333-3333-3333-3333-333333333333": makeServerState("3", RaftVoter, healthy),
					"55555555-5555-5555-5555-555555555555": makeServerState("5", RaftNonVoter, healthy),
				},
				Voters: []raft.ServerID{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
					"33333333-3333-3333-3333-333333333333",
				},
			},
			changes: RaftChanges{
				Demotions: []raft.ServerID{
					// already a non-voter so nothing should be done
					"55555555-5555-5555-5555-555555555555",
				},
				// should transfer leadership because no demotions actually needed performing
				Leader: "22222222-2222-2222-2222-222222222222",
			},
			setupExpectations: func(m *MockRaft) {
				m.On("LeadershipTransferToServer",
					raft.ServerID("22222222-2222-2222-2222-222222222222"),
					raft.ServerAddress("198.18.0.2:8300")).Return(&raftIndexFuture{}).Once()
			},
		},
		"demote-single-unhealthy-voter-promote-a-non-voter": {
			state: State{
				Leader: "11111111-1111-1111-1111-111111111111",
				Servers: map[raft.ServerID]*ServerState{
					"11111111-1111-1111-1111-111111111111": makeServerState("1", RaftLeader, healthy),
					"22222222-2222-2222-2222-222222222222": makeServerState("2", RaftVoter, unhealthy),
					"33333333-3333-3333-3333-333333333333": makeServerState("3", RaftVoter, unhealthy),
					"44444444-4444-4444-4444-444444444444": makeServerState("4", RaftVoter, healthy),
					"55555555-5555-5555-5555-555555555555": makeServerState("5", RaftNonVoter, healthy),
				},
				Voters: []raft.ServerID{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
					"33333333-3333-3333-3333-333333333333",
					"44444444-4444-4444-4444-444444444444",
					"55555555-5555-5555-5555-555555555555",
				},
			},
			changes: RaftChanges{
				Promotions: []raft.ServerID{
					// promotion should happen after one failed voter is demoted
					"55555555-5555-5555-5555-555555555555",
				},
				Demotions: []raft.ServerID{
					// healthy voter should not be demoted unless there are no unhealthy voters to
					// demote first and no promotions to be made
					"44444444-4444-4444-4444-444444444444",
					// unhealthy voter should be demoted
					"22222222-2222-2222-2222-222222222222",
					// unhealthy voter won't be demoted because we demoted one already
					"33333333-3333-3333-3333-333333333333",
				},
			},
			setupExpectations: func(m *MockRaft) {
				m.On("DemoteVoter",
					raft.ServerID("22222222-2222-2222-2222-222222222222"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				m.On("AddVoter",
					raft.ServerID("55555555-5555-5555-5555-555555555555"),
					raft.ServerAddress("198.18.0.5:8300"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
			},
		},
		"promote-first-with-even-voters": {
			state: State{
				Leader: "11111111-1111-1111-1111-111111111111",
				Servers: map[raft.ServerID]*ServerState{
					"11111111-1111-1111-1111-111111111111": makeServerState("1", RaftLeader, healthy),
					"22222222-2222-2222-2222-222222222222": makeServerState("2", RaftVoter, healthy),
					"33333333-3333-3333-3333-333333333333": makeServerState("3", RaftVoter, unhealthy),
					"44444444-4444-4444-4444-444444444444": makeServerState("4", RaftVoter, unhealthy),
					"55555555-5555-5555-5555-555555555555": makeServerState("5", RaftNonVoter, healthy),
					"66666666-6666-6666-6666-666666666666": makeServerState("6", RaftNonVoter, healthy),
				},
				Voters: []raft.ServerID{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
					"33333333-3333-3333-3333-333333333333",
					"44444444-4444-4444-4444-444444444444",
				},
			},
			changes: RaftChanges{
				Promotions: []raft.ServerID{
					// both promotions should happen
					"55555555-5555-5555-5555-555555555555",
					"66666666-6666-6666-6666-666666666666",
				},
				Demotions: []raft.ServerID{
					// Both servers are unhealthy, but no demotions should happen in this
					// reconciliation since we are starting with an even number of voters. We're
					// still passing the demotions to the reconcile function to ensure that they are
					// filtered out.
					"33333333-3333-3333-3333-333333333333",
					"44444444-4444-4444-4444-444444444444",
				},
			},
			setupExpectations: func(m *MockRaft) {
				m.On("AddVoter",
					raft.ServerID("55555555-5555-5555-5555-555555555555"),
					raft.ServerAddress("198.18.0.5:8300"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				m.On("AddVoter",
					raft.ServerID("66666666-6666-6666-6666-666666666666"),
					raft.ServerAddress("198.18.0.6:8300"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
			},
		},
		"demote-any-with-even-voters-and-no-promotions": {
			state: State{
				Leader: "11111111-1111-1111-1111-111111111111",
				Servers: map[raft.ServerID]*ServerState{
					"11111111-1111-1111-1111-111111111111": makeServerState("1", RaftLeader, healthy),
					"22222222-2222-2222-2222-222222222222": makeServerState("2", RaftVoter, healthy),
					"33333333-3333-3333-3333-333333333333": makeServerState("3", RaftVoter, unhealthy),
					"44444444-4444-4444-4444-444444444444": makeServerState("4", RaftVoter, unhealthy),
					"55555555-5555-5555-5555-555555555555": makeServerState("5", RaftVoter, healthy),
					"66666666-6666-6666-6666-666666666666": makeServerState("6", RaftVoter, healthy),
				},
				Voters: []raft.ServerID{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
					"33333333-3333-3333-3333-333333333333",
					"44444444-4444-4444-4444-444444444444",
				},
			},
			changes: RaftChanges{
				Demotions: []raft.ServerID{
					"33333333-3333-3333-3333-333333333333",
					"44444444-4444-4444-4444-444444444444",
					"55555555-5555-5555-5555-555555555555",
				},
			},
			setupExpectations: func(m *MockRaft) {
				m.On("DemoteVoter",
					raft.ServerID("33333333-3333-3333-3333-333333333333"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				m.On("DemoteVoter",
					raft.ServerID("44444444-4444-4444-4444-444444444444"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				m.On("DemoteVoter",
					raft.ServerID("55555555-5555-5555-5555-555555555555"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
			},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			mpromoter := NewMockPromoter(t)
			mpromoter.On("CalculatePromotionsAndDemotions", &Config{}, &tcase.state).Return(tcase.changes).Once()

			mapp := NewMockApplicationIntegration(t)
			mapp.On("AutopilotConfig").Return(&Config{}).Once()

			mraft := NewMockRaft(t)

			a := &Autopilot{
				logger:                hclog.NewNullLogger(),
				raft:                  mraft,
				delegate:              mapp,
				state:                 &tcase.state,
				promoter:              mpromoter,
				reconciliationEnabled: true,
			}

			if tcase.setupExpectations != nil {
				tcase.setupExpectations(mraft)
			}

			err := a.reconcile()
			require.NoError(t, err)
		})
	}
}

func TestPruneDeadServers(t *testing.T) {
	type testCase struct {
		expectedFailed    FailedServers
		knownServers      map[raft.ServerID]*Server
		raftConfig        raft.Configuration
		state             State
		setupExpectations func(*MockRaft, *MockApplicationIntegration)
	}

	cases := map[string]testCase{
		"one-of-each-removal": {
			raftConfig: raft.Configuration{
				Servers: []raft.Server{
					{
						Suffrage: raft.Voter,
						ID:       "51b2d56e-816e-409a-8b8e-afef2cf49663",
						Address:  "198.18.0.1:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "51fb4248-be6a-43e5-b47f-c089818e2010",
						Address:  "198.18.0.2:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
						Address:  "198.18.0.7:8300",
					},
					// this is going to be our failed voter
					{
						Suffrage: raft.Voter,
						ID:       "3857f1d4-5c23-4016-9078-fee502c0d1be",
						Address:  "198.18.0.3:8300",
					},
					// this is going to be our stale non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "db877f23-3e0a-4107-8ed8-bd7c3d710944",
						Address:  "198.18.0.4:8300",
					},
					// this is going to be our stale voter
					{
						Suffrage: raft.Voter,
						ID:       "0aacc844-1d0a-4ba7-bbc2-bd88d51cb231",
						Address:  "198.18.0.5:8300",
					},
					// this is going to be our failed non voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "8830c599-04cc-4b28-9b75-173355d49ab1",
						Address:  "198.18.0.6:8300",
					},
				},
			},
			expectedFailed: FailedServers{
				StaleNonVoters: []raft.ServerID{"db877f23-3e0a-4107-8ed8-bd7c3d710944"},
				StaleVoters:    []raft.ServerID{"0aacc844-1d0a-4ba7-bbc2-bd88d51cb231"},
				FailedNonVoters: []*Server{
					{
						ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
						Name:       "node6",
						Address:    "198.18.0.6:8300",
						NodeStatus: NodeFailed,
						NodeType:   NodeVoter,
					},
				},
				FailedVoters: []*Server{
					{
						ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
						Name:       "node3",
						Address:    "198.18.0.3:8300",
						NodeStatus: NodeFailed,
						NodeType:   NodeVoter,
					},
				},
			},
			knownServers: map[raft.ServerID]*Server{
				"51b2d56e-816e-409a-8b8e-afef2cf49663": {
					ID:         "51b2d56e-816e-409a-8b8e-afef2cf49663",
					Name:       "node1",
					Address:    "198.18.0.1:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"51fb4248-be6a-43e5-b47f-c089818e2010": {
					ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
					Name:       "node2",
					Address:    "198.18.0.2:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"a227f9a9-f55e-4321-b959-5afdcc63c6d4": {
					ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
					Name:       "node7",
					Address:    "198.18.0.7:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"3857f1d4-5c23-4016-9078-fee502c0d1be": {
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
				"8830c599-04cc-4b28-9b75-173355d49ab1": {
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
					Name:       "node6",
					Address:    "198.18.0.6:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
			},
			state: State{
				Servers: map[raft.ServerID]*ServerState{
					"51b2d56e-816e-409a-8b8e-afef2cf49663": {
						Server: Server{
							ID:         "51b2d56e-816e-409a-8b8e-afef2cf49663",
							Name:       "node1",
							Address:    "198.18.0.1:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
					},
					"51fb4248-be6a-43e5-b47f-c089818e2010": {
						Server: Server{
							ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
							Name:       "node2",
							Address:    "198.18.0.2:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d4": {
						Server: Server{
							ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
							Name:       "node7",
							Address:    "198.18.0.7:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
					},
					"3857f1d4-5c23-4016-9078-fee502c0d1be": {
						Server: Server{
							ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
							Name:       "node3",
							Address:    "198.18.0.3:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
					},
					"8830c599-04cc-4b28-9b75-173355d49ab1": {
						Server: Server{
							ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
							Name:       "node6",
							Address:    "198.18.0.6:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
					},
				},
			},
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {
				mraft.On("RemoveServer",
					raft.ServerID("db877f23-3e0a-4107-8ed8-bd7c3d710944"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				mraft.On("RemoveServer",
					raft.ServerID("0aacc844-1d0a-4ba7-bbc2-bd88d51cb231"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				mapp.On("RemoveFailedServer", &Server{
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
					Name:       "node6",
					Address:    "198.18.0.6:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				}).Once()
				mapp.On("RemoveFailedServer", &Server{
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				}).Once()
			},
		},
		"quorum-instability-prevention": {
			raftConfig: raft.Configuration{
				Servers: []raft.Server{
					{
						Suffrage: raft.Voter,
						ID:       "51b2d56e-816e-409a-8b8e-afef2cf49663",
						Address:  "198.18.0.1:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "51fb4248-be6a-43e5-b47f-c089818e2010",
						Address:  "198.18.0.2:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
						Address:  "198.18.0.7:8300",
					},
					// this is going to be our failed voter
					{
						Suffrage: raft.Voter,
						ID:       "3857f1d4-5c23-4016-9078-fee502c0d1be",
						Address:  "198.18.0.3:8300",
					},
					// this is going to be our stale non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "db877f23-3e0a-4107-8ed8-bd7c3d710944",
						Address:  "198.18.0.4:8300",
					},
					// this is going to be our stale voter
					{
						Suffrage: raft.Voter,
						ID:       "0aacc844-1d0a-4ba7-bbc2-bd88d51cb231",
						Address:  "198.18.0.5:8300",
					},
					// this is going to be our failed non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "8830c599-04cc-4b28-9b75-173355d49ab1",
						Address:  "198.18.0.6:8300",
					},
				},
			},
			expectedFailed: FailedServers{
				StaleNonVoters: []raft.ServerID{"db877f23-3e0a-4107-8ed8-bd7c3d710944"},
				StaleVoters:    []raft.ServerID{"0aacc844-1d0a-4ba7-bbc2-bd88d51cb231"},
				FailedNonVoters: []*Server{
					{
						ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
						Name:       "node6",
						Address:    "198.18.0.6:8300",
						NodeStatus: NodeFailed,
						NodeType:   NodeVoter,
					},
				},
				FailedVoters: []*Server{
					{
						ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
						Name:       "node3",
						Address:    "198.18.0.3:8300",
						NodeStatus: NodeFailed,
						NodeType:   NodeVoter,
					},
					{
						ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
						Name:       "node2",
						Address:    "198.18.0.2:8300",
						NodeStatus: NodeFailed,
						NodeType:   NodeVoter,
					},
				},
			},
			knownServers: map[raft.ServerID]*Server{
				"51b2d56e-816e-409a-8b8e-afef2cf49663": {
					ID:         "51b2d56e-816e-409a-8b8e-afef2cf49663",
					Name:       "node1",
					Address:    "198.18.0.1:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"51fb4248-be6a-43e5-b47f-c089818e2010": {
					ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
					Name:       "node2",
					Address:    "198.18.0.2:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
				"a227f9a9-f55e-4321-b959-5afdcc63c6d4": {
					ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
					Name:       "node7",
					Address:    "198.18.0.7:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"3857f1d4-5c23-4016-9078-fee502c0d1be": {
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
				"8830c599-04cc-4b28-9b75-173355d49ab1": {
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
					Name:       "node6",
					Address:    "198.18.0.6:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
			},
			state: State{
				Servers: map[raft.ServerID]*ServerState{
					"51b2d56e-816e-409a-8b8e-afef2cf49663": {
						Server: Server{
							ID:         "51b2d56e-816e-409a-8b8e-afef2cf49663",
							Name:       "node1",
							Address:    "198.18.0.1:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
						State: RaftLeader,
					},
					"51fb4248-be6a-43e5-b47f-c089818e2010": {
						Server: Server{
							ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
							Name:       "node2",
							Address:    "198.18.0.2:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
						State: RaftVoter,
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d4": {
						Server: Server{
							ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
							Name:       "node7",
							Address:    "198.18.0.7:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
						State: RaftVoter,
					},
					"3857f1d4-5c23-4016-9078-fee502c0d1be": {
						Server: Server{
							ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
							Name:       "node3",
							Address:    "198.18.0.3:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
						State: RaftVoter,
					},
					"8830c599-04cc-4b28-9b75-173355d49ab1": {
						Server: Server{
							ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
							Name:       "node6",
							Address:    "198.18.0.6:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
						State: RaftNonVoter,
					},
				},
			},
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {
				mraft.On("RemoveServer",
					raft.ServerID("db877f23-3e0a-4107-8ed8-bd7c3d710944"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				mraft.On("RemoveServer",
					raft.ServerID("0aacc844-1d0a-4ba7-bbc2-bd88d51cb231"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				mapp.On("RemoveFailedServer", &Server{
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
					Name:       "node6",
					Address:    "198.18.0.6:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				}).Once()
				mapp.On("RemoveFailedServer", &Server{
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				}).Once()
			},
		},
		"dont-go-under-min-quorum-with-3": {
			raftConfig: raft.Configuration{
				Servers: []raft.Server{
					{
						Suffrage: raft.Voter,
						ID:       "51b2d56e-816e-409a-8b8e-afef2cf49661",
						Address:  "198.18.0.1:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "51fb4248-be6a-43e5-b47f-c089818e2012",
						Address:  "198.18.0.2:8300",
					},
					// this is going to be our failed non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
						Address:  "198.18.0.3:8300",
					},
				},
			},
			expectedFailed: FailedServers{
				FailedNonVoters: []*Server{
					{
						ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
						Name:       "node3",
						Address:    "198.18.0.3:8300",
						NodeStatus: NodeFailed,
						NodeType:   NodeVoter,
					},
				},
			},
			knownServers: map[raft.ServerID]*Server{
				"51b2d56e-816e-409a-8b8e-afef2cf49661": {
					ID:         "51b2d56e-816e-409a-8b8e-afef2cf49661",
					Name:       "node1",
					Address:    "198.18.0.1:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"51fb4248-be6a-43e5-b47f-c089818e2012": {
					ID:         "51fb4248-be6a-43e5-b47f-c089818e2012",
					Name:       "node2",
					Address:    "198.18.0.2:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"a227f9a9-f55e-4321-b959-5afdcc63c6d3": {
					ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
			},
			state: State{
				Servers: map[raft.ServerID]*ServerState{
					"51b2d56e-816e-409a-8b8e-afef2cf49661": {
						Server: Server{
							ID:         "51b2d56e-816e-409a-8b8e-afef2cf49661",
							Name:       "node1",
							Address:    "198.18.0.1:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						}, State: RaftVoter,
					},
					"51fb4248-be6a-43e5-b47f-c089818e2012": {
						Server: Server{
							ID:         "51fb4248-be6a-43e5-b47f-c089818e2012",
							Name:       "node2",
							Address:    "198.18.0.2:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						}, State: RaftVoter,
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d3": {
						Server: Server{
							ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
							Name:       "node3",
							Address:    "198.18.0.3:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						}, State: RaftNonVoter,
					},
				},
			},
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {
			},
		},
		"dont-go-under-min-quorum-with-2": {
			raftConfig: raft.Configuration{
				Servers: []raft.Server{
					{
						Suffrage: raft.Voter,
						ID:       "51b2d56e-816e-409a-8b8e-afef2cf49661",
						Address:  "198.18.0.1:8300",
					},
					// this is going to be our failed voter
					{
						Suffrage: raft.Voter,
						ID:       "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
						Address:  "198.18.0.3:8300",
					},
				},
			},
			expectedFailed: FailedServers{
				FailedVoters: []*Server{
					{
						ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
						Name:       "node3",
						Address:    "198.18.0.3:8300",
						NodeStatus: NodeFailed,
						NodeType:   NodeVoter,
					},
				},
			},
			knownServers: map[raft.ServerID]*Server{
				"51b2d56e-816e-409a-8b8e-afef2cf49661": {
					ID:         "51b2d56e-816e-409a-8b8e-afef2cf49661",
					Name:       "node1",
					Address:    "198.18.0.1:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"a227f9a9-f55e-4321-b959-5afdcc63c6d3": {
					ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
			},
			state: State{
				Servers: map[raft.ServerID]*ServerState{
					"51b2d56e-816e-409a-8b8e-afef2cf49661": {
						Server: Server{
							ID:         "51b2d56e-816e-409a-8b8e-afef2cf49661",
							Name:       "node1",
							Address:    "198.18.0.1:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						}, State: RaftVoter,
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d3": {
						Server: Server{
							ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
							Name:       "node3",
							Address:    "198.18.0.3:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						}, State: RaftVoter,
					},
				},
			},
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {
			},
		},
		"stale-non-voters": {
			// 2 working nodes and 2 stale servers - should only remove stale
			// non-voter and refuse to remove anything else due to restrictions
			// preventing removal half or more of the servers in the cluster at once.
			raftConfig: raft.Configuration{
				Servers: []raft.Server{
					{
						Suffrage: raft.Voter,
						ID:       "51b2d56e-816e-409a-8b8e-afef2cf49661",
						Address:  "198.18.0.1:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "51fb4248-be6a-43e5-b47f-c089818e2012",
						Address:  "198.18.0.2:8300",
					},
					// this is going to be our stale voter
					{
						Suffrage: raft.Voter,
						ID:       "3857f1d4-5c23-4016-9078-fee502c0d1b4",
						Address:  "198.18.0.4:8300",
					},
					// this is going to be our stale non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "db877f23-3e0a-4107-8ed8-bd7c3d710945",
						Address:  "198.18.0.5:8300",
					},
				},
			},
			knownServers: map[raft.ServerID]*Server{
				"51b2d56e-816e-409a-8b8e-afef2cf49661": {
					ID:         "51b2d56e-816e-409a-8b8e-afef2cf49661",
					Name:       "node1",
					Address:    "198.18.0.1:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"51fb4248-be6a-43e5-b47f-c089818e2012": {
					ID:         "51fb4248-be6a-43e5-b47f-c089818e2012",
					Name:       "node2",
					Address:    "198.18.0.2:8300",
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
			},
			state: State{
				Servers: map[raft.ServerID]*ServerState{
					"51b2d56e-816e-409a-8b8e-afef2cf49661": {
						Server: Server{
							ID:         "51b2d56e-816e-409a-8b8e-afef2cf49661",
							Name:       "node1",
							Address:    "198.18.0.1:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
						State: RaftLeader,
					},
					"51fb4248-be6a-43e5-b47f-c089818e2012": {
						Server: Server{
							ID:         "51fb4248-be6a-43e5-b47f-c089818e2012",
							Name:       "node2",
							Address:    "198.18.0.2:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
						State: RaftVoter,
					},
					"3857f1d4-5c23-4016-9078-fee502c0d1b4": {
						Server: Server{
							ID:         "3857f1d4-5c23-4016-9078-fee502c0d1b4",
							Name:       "node4",
							Address:    "198.18.0.4:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
						State: RaftVoter,
					},
					"db877f23-3e0a-4107-8ed8-bd7c3d710945": {
						Server: Server{
							ID:         "db877f23-3e0a-4107-8ed8-bd7c3d710945",
							Name:       "node5",
							Address:    "198.18.0.5:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
						State: RaftNonVoter,
					},
				},
			},
			expectedFailed: FailedServers{
				StaleNonVoters: []raft.ServerID{
					"db877f23-3e0a-4107-8ed8-bd7c3d710945",
				},
				StaleVoters: []raft.ServerID{
					"3857f1d4-5c23-4016-9078-fee502c0d1b4",
				},
			},
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {
				// Stale non-voter
				mraft.On("RemoveServer",
					raft.ServerID("db877f23-3e0a-4107-8ed8-bd7c3d710945"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
			},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			conf := &Config{
				CleanupDeadServers: true,
				MinQuorum:          3,
			}
			mpromoter := NewMockPromoter(t)
			mpromoter.On("FilterFailedServerRemovals", conf, &tcase.state, &tcase.expectedFailed).Return(&tcase.expectedFailed).Once()
			mpromoter.On("IsPotentialVoter", NodeVoter).Return(true)
			mapp := NewMockApplicationIntegration(t)
			mapp.On("AutopilotConfig").Return(conf).Times(5)
			mapp.On("KnownServers").Return(tcase.knownServers).Once()

			mraft := NewMockRaft(t)

			mraft.On("GetConfiguration").Return(&raftConfigFuture{config: tcase.raftConfig}).Once()

			a := &Autopilot{
				logger:                hclog.NewNullLogger(),
				raft:                  mraft,
				delegate:              mapp,
				state:                 &tcase.state,
				promoter:              mpromoter,
				reconciliationEnabled: true,
			}

			if tcase.setupExpectations != nil {
				tcase.setupExpectations(mraft, mapp)
			}

			err := a.pruneDeadServers()
			require.NoError(t, err)
		})
	}
}

func TestReconcileDisabled(t *testing.T) {
	ap := New(NewMockRaft(t), NewMockApplicationIntegration(t),
		WithLogger(testLogger(t)),
		WithReconciliationDisabled())
	require.NoError(t, ap.reconcile())
}

func TestPruneDeadServersDisabled(t *testing.T) {
	ap := New(NewMockRaft(t), NewMockApplicationIntegration(t),
		WithLogger(testLogger(t)),
		WithReconciliationDisabled())
	require.NoError(t, ap.pruneDeadServers())
}
