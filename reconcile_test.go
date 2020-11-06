package autopilot

import (
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

func TestReconcile(t *testing.T) {
	type testCase struct {
		changes           RaftChanges
		state             State
		setupExpectations func(*MockRaft)
	}

	cases := map[string]testCase{
		"promote-one-and-filter-others-and-demotions": {
			state: State{
				Leader: "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
				Servers: map[raft.ServerID]*ServerState{
					"96be11f3-c9b9-45ab-a719-dc9472ada6fe": {
						Server: Server{
							ID:          "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
							Name:        "node1",
							Address:     "198.18.0.1:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftLeader,
						Health: ServerHealth{Healthy: true},
					},
					"4b92b892-ee0d-4644-84fb-3117448a0401": {
						Server: Server{
							ID:          "4b92b892-ee0d-4644-84fb-3117448a0401",
							Name:        "node2",
							Address:     "198.18.0.2:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: true},
					},
					"0a79bbf7-7113-4947-a257-6179326f188c": {
						Server: Server{
							ID:          "0a79bbf7-7113-4947-a257-6179326f188c",
							Name:        "node3",
							Address:     "198.18.0.3:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftNonVoter,
						Health: ServerHealth{Healthy: true},
					},
					"b8508007-68d5-42c9-92a6-28686676867e": {
						Server: Server{
							ID:          "b8508007-68d5-42c9-92a6-28686676867e",
							Name:        "node4",
							Address:     "198.18.0.4:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftNonVoter,
						Health: ServerHealth{Healthy: false},
					},
					"bcd603a7-18e2-48c6-ac60-167e1556f4b0": {
						Server: Server{
							ID:          "bcd603a7-18e2-48c6-ac60-167e1556f4b0",
							Name:        "node5",
							Address:     "198.18.0.5:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: false},
					},
				},
			},
			changes: RaftChanges{
				Promotions: []raft.ServerID{
					// already is a voter so no config change should be made
					"96be11f3-c9b9-45ab-a719-dc9472ada6fe",
					// already is a voter so no config change should be made
					"4b92b892-ee0d-4644-84fb-3117448a0401",
					// should actually be promoted
					"0a79bbf7-7113-4947-a257-6179326f188c",
					// is not healthy so we shouldn't promote it
					"b8508007-68d5-42c9-92a6-28686676867e",
				},
				Demotions: []raft.ServerID{
					// shouldn't actually be done since we are promoting one
					"bcd603a7-18e2-48c6-ac60-167e1556f4b0",
				},
			},
			setupExpectations: func(m *MockRaft) {
				m.On("AddVoter",
					raft.ServerID("0a79bbf7-7113-4947-a257-6179326f188c"),
					raft.ServerAddress("198.18.0.3:8300"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
			},
		},
		"demotions-and-filter-leader-transfer": {
			state: State{
				Leader: "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
				Servers: map[raft.ServerID]*ServerState{
					"96be11f3-c9b9-45ab-a719-dc9472ada6fe": {
						Server: Server{
							ID:          "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
							Name:        "node1",
							Address:     "198.18.0.1:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftLeader,
						Health: ServerHealth{Healthy: true},
					},
					"1f07052e-53c7-4f99-9cb6-6bb5b39ce8a8": {
						Server: Server{
							ID:          "1f07052e-53c7-4f99-9cb6-6bb5b39ce8a8",
							Name:        "node2",
							Address:     "198.18.0.2:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: true},
					},
					"c86da54b-19e1-4ead-a3b7-17807b7712a6": {
						Server: Server{
							ID:          "c86da54b-19e1-4ead-a3b7-17807b7712a6",
							Name:        "node3",
							Address:     "198.18.0.3:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: true},
					},
					"4b92b892-ee0d-4644-84fb-3117448a0401": {
						Server: Server{
							ID:          "4b92b892-ee0d-4644-84fb-3117448a0401",
							Name:        "node4",
							Address:     "198.18.0.4:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: true},
					},
					"0a79bbf7-7113-4947-a257-6179326f188c": {
						Server: Server{
							ID:          "0a79bbf7-7113-4947-a257-6179326f188c",
							Name:        "node5",
							Address:     "198.18.0.5:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftNonVoter,
						Health: ServerHealth{Healthy: true},
					},
				},
			},
			changes: RaftChanges{
				Promotions: []raft.ServerID{
					// already is a voter so no config change should be made
					// this should also not prevent the demotions from happening
					"96be11f3-c9b9-45ab-a719-dc9472ada6fe",
				},
				Demotions: []raft.ServerID{
					// already a non-voter so nothing should be done
					"0a79bbf7-7113-4947-a257-6179326f188c",
					// should adtually get demoted
					"4b92b892-ee0d-4644-84fb-3117448a0401",
				},
				// no leader ship transfer should be done because we did demotions
				Leader: "c86da54b-19e1-4ead-a3b7-17807b7712a6",
			},
			setupExpectations: func(m *MockRaft) {
				m.On("DemoteVoter",
					raft.ServerID("4b92b892-ee0d-4644-84fb-3117448a0401"),
					uint64(0),
					time.Duration(0)).Return(&raftIndexFuture{}).Once()
			},
		},
		"leader-transfer-same-leader": {
			state: State{
				Leader: "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
				Servers: map[raft.ServerID]*ServerState{
					"96be11f3-c9b9-45ab-a719-dc9472ada6fe": {
						Server: Server{
							ID:          "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
							Name:        "node1",
							Address:     "198.18.0.1:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftLeader,
						Health: ServerHealth{Healthy: true},
					},
					"1f07052e-53c7-4f99-9cb6-6bb5b39ce8a8": {
						Server: Server{
							ID:          "1f07052e-53c7-4f99-9cb6-6bb5b39ce8a8",
							Name:        "node2",
							Address:     "198.18.0.2:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: true},
					},
					"c86da54b-19e1-4ead-a3b7-17807b7712a6": {
						Server: Server{
							ID:          "c86da54b-19e1-4ead-a3b7-17807b7712a6",
							Name:        "node3",
							Address:     "198.18.0.3:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: true},
					},
					"0a79bbf7-7113-4947-a257-6179326f188c": {
						Server: Server{
							ID:          "0a79bbf7-7113-4947-a257-6179326f188c",
							Name:        "node5",
							Address:     "198.18.0.5:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftNonVoter,
						Health: ServerHealth{Healthy: true},
					},
				},
			},
			changes: RaftChanges{
				Demotions: []raft.ServerID{
					// already a non-voter so nothing should be done
					"0a79bbf7-7113-4947-a257-6179326f188c",
				},
				// no leader ship transfer should be done because we did demotions
				Leader: "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
			},
		},
		"leader-transfer": {
			state: State{
				Leader: "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
				Servers: map[raft.ServerID]*ServerState{
					"96be11f3-c9b9-45ab-a719-dc9472ada6fe": {
						Server: Server{
							ID:          "96be11f3-c9b9-45ab-a719-dc9472ada6fe",
							Name:        "node1",
							Address:     "198.18.0.1:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftLeader,
						Health: ServerHealth{Healthy: true},
					},
					"1f07052e-53c7-4f99-9cb6-6bb5b39ce8a8": {
						Server: Server{
							ID:          "1f07052e-53c7-4f99-9cb6-6bb5b39ce8a8",
							Name:        "node2",
							Address:     "198.18.0.2:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: true},
					},
					"c86da54b-19e1-4ead-a3b7-17807b7712a6": {
						Server: Server{
							ID:          "c86da54b-19e1-4ead-a3b7-17807b7712a6",
							Name:        "node3",
							Address:     "198.18.0.3:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftVoter,
						Health: ServerHealth{Healthy: true},
					},
					"0a79bbf7-7113-4947-a257-6179326f188c": {
						Server: Server{
							ID:          "0a79bbf7-7113-4947-a257-6179326f188c",
							Name:        "node5",
							Address:     "198.18.0.5:8300",
							NodeStatus:  NodeAlive,
							Version:     "1.9.0",
							RaftVersion: 3,
						},
						State:  RaftNonVoter,
						Health: ServerHealth{Healthy: true},
					},
				},
			},
			changes: RaftChanges{
				Demotions: []raft.ServerID{
					// already a non-voter so nothing should be done
					"0a79bbf7-7113-4947-a257-6179326f188c",
				},
				// no leader ship transfer should be done because we did demotions
				Leader: "1f07052e-53c7-4f99-9cb6-6bb5b39ce8a8",
			},
			setupExpectations: func(m *MockRaft) {
				m.On("LeadershipTransferToServer",
					raft.ServerID("1f07052e-53c7-4f99-9cb6-6bb5b39ce8a8"),
					raft.ServerAddress("198.18.0.2:8300")).Return(&raftIndexFuture{}).Once()
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
				logger:   hclog.NewNullLogger(),
				raft:     mraft,
				delegate: mapp,
				state:    &tcase.state,
				promoter: mpromoter,
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
					},
				},
				FailedVoters: []*Server{
					{
						ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
						Name:       "node3",
						Address:    "198.18.0.3:8300",
						NodeStatus: NodeFailed,
					},
				},
			},
			knownServers: map[raft.ServerID]*Server{
				"51b2d56e-816e-409a-8b8e-afef2cf49663": {
					ID:         "51b2d56e-816e-409a-8b8e-afef2cf49663",
					Name:       "node1",
					Address:    "198.18.0.1:8300",
					NodeStatus: NodeAlive,
				},
				"51fb4248-be6a-43e5-b47f-c089818e2010": {
					ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
					Name:       "node2",
					Address:    "198.18.0.2:8300",
					NodeStatus: NodeAlive,
				},
				"a227f9a9-f55e-4321-b959-5afdcc63c6d4": {
					ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
					Name:       "node7",
					Address:    "198.18.0.7:8300",
					NodeStatus: NodeAlive,
				},
				"3857f1d4-5c23-4016-9078-fee502c0d1be": {
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
				},
				"8830c599-04cc-4b28-9b75-173355d49ab1": {
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
					Name:       "node6",
					Address:    "198.18.0.6:8300",
					NodeStatus: NodeFailed,
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
						},
					},
					"51fb4248-be6a-43e5-b47f-c089818e2010": {
						Server: Server{
							ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
							Name:       "node2",
							Address:    "198.18.0.2:8300",
							NodeStatus: NodeAlive,
						},
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d4": {
						Server: Server{
							ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
							Name:       "node7",
							Address:    "198.18.0.7:8300",
							NodeStatus: NodeAlive,
						},
					},
					"3857f1d4-5c23-4016-9078-fee502c0d1be": {
						Server: Server{
							ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
							Name:       "node3",
							Address:    "198.18.0.3:8300",
							NodeStatus: NodeFailed,
						},
					},
					"8830c599-04cc-4b28-9b75-173355d49ab1": {
						Server: Server{
							ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
							Name:       "node6",
							Address:    "198.18.0.6:8300",
							NodeStatus: NodeFailed,
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
				}).Once()
				mapp.On("RemoveFailedServer", &Server{
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
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
					},
				},
				FailedVoters: []*Server{
					{
						ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
						Name:       "node3",
						Address:    "198.18.0.3:8300",
						NodeStatus: NodeFailed,
					},
					{
						ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
						Name:       "node2",
						Address:    "198.18.0.2:8300",
						NodeStatus: NodeFailed,
					},
				},
			},
			knownServers: map[raft.ServerID]*Server{
				"51b2d56e-816e-409a-8b8e-afef2cf49663": {
					ID:         "51b2d56e-816e-409a-8b8e-afef2cf49663",
					Name:       "node1",
					Address:    "198.18.0.1:8300",
					NodeStatus: NodeAlive,
				},
				"51fb4248-be6a-43e5-b47f-c089818e2010": {
					ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
					Name:       "node2",
					Address:    "198.18.0.2:8300",
					NodeStatus: NodeFailed,
				},
				"a227f9a9-f55e-4321-b959-5afdcc63c6d4": {
					ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
					Name:       "node7",
					Address:    "198.18.0.7:8300",
					NodeStatus: NodeAlive,
				},
				"3857f1d4-5c23-4016-9078-fee502c0d1be": {
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
				},
				"8830c599-04cc-4b28-9b75-173355d49ab1": {
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
					Name:       "node6",
					Address:    "198.18.0.6:8300",
					NodeStatus: NodeFailed,
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
						},
						State: RaftLeader,
					},
					"51fb4248-be6a-43e5-b47f-c089818e2010": {
						Server: Server{
							ID:         "51fb4248-be6a-43e5-b47f-c089818e2010",
							Name:       "node2",
							Address:    "198.18.0.2:8300",
							NodeStatus: NodeFailed,
						},
						State: RaftVoter,
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d4": {
						Server: Server{
							ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d4",
							Name:       "node7",
							Address:    "198.18.0.7:8300",
							NodeStatus: NodeAlive,
						},
						State: RaftVoter,
					},
					"3857f1d4-5c23-4016-9078-fee502c0d1be": {
						Server: Server{
							ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
							Name:       "node3",
							Address:    "198.18.0.3:8300",
							NodeStatus: NodeFailed,
						},
						State: RaftVoter,
					},
					"8830c599-04cc-4b28-9b75-173355d49ab1": {
						Server: Server{
							ID:         "8830c599-04cc-4b28-9b75-173355d49ab1",
							Name:       "node6",
							Address:    "198.18.0.6:8300",
							NodeStatus: NodeFailed,
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
				}).Once()
				mapp.On("RemoveFailedServer", &Server{
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1be",
					Name:       "node3",
					Address:    "198.18.0.3:8300",
					NodeStatus: NodeFailed,
				}).Once()
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

			mapp := NewMockApplicationIntegration(t)
			mapp.On("AutopilotConfig").Return(conf).Once()
			mapp.On("KnownServers").Return(tcase.knownServers).Once()

			mraft := NewMockRaft(t)

			mraft.On("GetConfiguration").Return(&raftConfigFuture{config: tcase.raftConfig}).Once()

			a := &Autopilot{
				logger:   hclog.NewNullLogger(),
				raft:     mraft,
				delegate: mapp,
				state:    &tcase.state,
				promoter: mpromoter,
			}

			if tcase.setupExpectations != nil {
				tcase.setupExpectations(mraft, mapp)
			}

			err := a.pruneDeadServers()
			require.NoError(t, err)
		})
	}
}
