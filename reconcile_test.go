package autopilot

import (
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/mock"
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
				// should transfer leadership because no demotions actually needed performing
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
		expectedServers   CategorizedServers
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
						ID:       "51b2d56e-816e-409a-8b8e-afef2cf49661",
						Address:  "198.18.0.1:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "51fb4248-be6a-43e5-b47f-c089818e2012",
						Address:  "198.18.0.2:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
						Address:  "198.18.0.3:8300",
					},
					// this is going to be our failed voter
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
					// this is going to be our stale voter
					{
						Suffrage: raft.Voter,
						ID:       "0aacc844-1d0a-4ba7-bbc2-bd88d51cb236",
						Address:  "198.18.0.6:8300",
					},
					// this is going to be our failed non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "8830c599-04cc-4b28-9b75-173355d49ab7",
						Address:  "198.18.0.7:8300",
					},
				},
			},
			expectedServers: CategorizedServers{
				StaleNonVoters: RaftServerEligibility{
					"db877f23-3e0a-4107-8ed8-bd7c3d710945": &VoterEligibility{
						currentVoter:   false,
						potentialVoter: false,
					},
				},
				StaleVoters: RaftServerEligibility{
					"0aacc844-1d0a-4ba7-bbc2-bd88d51cb236": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: false,
					},
				},
				FailedNonVoters: RaftServerEligibility{
					"8830c599-04cc-4b28-9b75-173355d49ab7": &VoterEligibility{
						currentVoter:   false,
						potentialVoter: true,
					},
				},
				FailedVoters: RaftServerEligibility{
					"3857f1d4-5c23-4016-9078-fee502c0d1b4": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
				},
				HealthyNonVoters: RaftServerEligibility{},
				HealthyVoters: RaftServerEligibility{
					"51b2d56e-816e-409a-8b8e-afef2cf49661": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
					"51fb4248-be6a-43e5-b47f-c089818e2012": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d3": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
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
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"3857f1d4-5c23-4016-9078-fee502c0d1b4": {
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1b4",
					Name:       "node4",
					Address:    "198.18.0.4:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
				"8830c599-04cc-4b28-9b75-173355d49ab7": {
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab7",
					Name:       "node7",
					Address:    "198.18.0.7:8300",
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
						},
					},
					"51fb4248-be6a-43e5-b47f-c089818e2012": {
						Server: Server{
							ID:         "51fb4248-be6a-43e5-b47f-c089818e2012",
							Name:       "node2",
							Address:    "198.18.0.2:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d3": {
						Server: Server{
							ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
							Name:       "node3",
							Address:    "198.18.0.3:8300",
							NodeStatus: NodeAlive,
							NodeType:   NodeVoter,
						},
					},
					"3857f1d4-5c23-4016-9078-fee502c0d1b4": {
						Server: Server{
							ID:         "3857f1d4-5c23-4016-9078-fee502c0d1b4",
							Name:       "node4",
							Address:    "198.18.0.4:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
					},
					// Failed non-voter
					"8830c599-04cc-4b28-9b75-173355d49ab7": {
						Server: Server{
							ID:         "8830c599-04cc-4b28-9b75-173355d49ab7",
							Name:       "node7",
							Address:    "198.18.0.7:8300",
							NodeStatus: NodeFailed,
						},
					},
					// Stale non-voter
					"db877f23-3e0a-4107-8ed8-bd7c3d710945": {
						Server: Server{
							ID:         "db877f23-3e0a-4107-8ed8-bd7c3d710945",
							Name:       "node5",
							Address:    "198.18.0.5:8300",
							NodeStatus: NodeLeft,
						},
					},
				},
			},
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {
				// Stale non-voter
				mraft.On("RemoveServer",
					raft.ServerID("db877f23-3e0a-4107-8ed8-bd7c3d710945"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				// Stale voter
				mraft.On("RemoveServer",
					raft.ServerID("0aacc844-1d0a-4ba7-bbc2-bd88d51cb236"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				// Failed non-voter
				mapp.On("RemoveFailedServer", &Server{
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab7",
					Name:       "node7",
					Address:    "198.18.0.7:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				}).Once()
				// Failed voter
				mapp.On("RemoveFailedServer", &Server{
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1b4",
					Name:       "node4",
					Address:    "198.18.0.4:8300",
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
						ID:       "51b2d56e-816e-409a-8b8e-afef2cf49661",
						Address:  "198.18.0.1:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "51fb4248-be6a-43e5-b47f-c089818e2012",
						Address:  "198.18.0.2:8300",
					},
					{
						Suffrage: raft.Voter,
						ID:       "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
						Address:  "198.18.0.3:8300",
					},
					// this is going to be our failed voter
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
					// this is going to be our stale voter
					{
						Suffrage: raft.Voter,
						ID:       "0aacc844-1d0a-4ba7-bbc2-bd88d51cb236",
						Address:  "198.18.0.6:8300",
					},
					// this is going to be our failed non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "8830c599-04cc-4b28-9b75-173355d49ab7",
						Address:  "198.18.0.7:8300",
					},
				},
			},
			expectedServers: CategorizedServers{
				StaleNonVoters: RaftServerEligibility{
					"db877f23-3e0a-4107-8ed8-bd7c3d710945": &VoterEligibility{
						currentVoter:   false,
						potentialVoter: false,
					},
				},
				StaleVoters: RaftServerEligibility{
					"0aacc844-1d0a-4ba7-bbc2-bd88d51cb236": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: false,
					},
				},
				FailedNonVoters: RaftServerEligibility{
					"8830c599-04cc-4b28-9b75-173355d49ab7": &VoterEligibility{
						currentVoter:   false,
						potentialVoter: true,
					},
				},
				FailedVoters: RaftServerEligibility{
					"3857f1d4-5c23-4016-9078-fee502c0d1b4": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
				},
				HealthyNonVoters: RaftServerEligibility{},
				HealthyVoters: RaftServerEligibility{
					"51b2d56e-816e-409a-8b8e-afef2cf49661": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
					"51fb4248-be6a-43e5-b47f-c089818e2012": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
					"a227f9a9-f55e-4321-b959-5afdcc63c6d3": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
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
					NodeStatus: NodeAlive,
					NodeType:   NodeVoter,
				},
				"3857f1d4-5c23-4016-9078-fee502c0d1b4": {
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1b4",
					Name:       "node4",
					Address:    "198.18.0.4:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				},
				"8830c599-04cc-4b28-9b75-173355d49ab7": {
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab7",
					Name:       "node7",
					Address:    "198.18.0.7:8300",
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
					"a227f9a9-f55e-4321-b959-5afdcc63c6d3": {
						Server: Server{
							ID:         "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
							Name:       "node3",
							Address:    "198.18.0.3:8300",
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
					"8830c599-04cc-4b28-9b75-173355d49ab7": {
						Server: Server{
							ID:         "8830c599-04cc-4b28-9b75-173355d49ab7",
							Name:       "node7",
							Address:    "198.18.0.7:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						},
						State: RaftNonVoter,
					},
				},
			},
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {
				mraft.On("RemoveServer",
					raft.ServerID("db877f23-3e0a-4107-8ed8-bd7c3d710945"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				mraft.On("RemoveServer",
					raft.ServerID("0aacc844-1d0a-4ba7-bbc2-bd88d51cb236"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				mapp.On("RemoveFailedServer", &Server{
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab7",
					Name:       "node7",
					Address:    "198.18.0.7:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				}).Once()
				mapp.On("RemoveFailedServer", &Server{
					ID:         "3857f1d4-5c23-4016-9078-fee502c0d1b4",
					Name:       "node4",
					Address:    "198.18.0.4:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				}).Once()
			},
		},
		"ignore-stabilizing-nodes": {
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
					// this is going to be our failed voter
					{
						Suffrage: raft.Voter,
						ID:       "a227f9a9-f55e-4321-b959-5afdcc63c6d3",
						Address:  "198.18.0.3:8300",
					},
					// this is going to be our stale non-voter
					// (it won't be known to the delegate that supplies 'KnownServers'
					{
						Suffrage: raft.Nonvoter,
						ID:       "db877f23-3e0a-4107-8ed8-bd7c3d710944",
						Address:  "198.18.0.4:8300",
					},
					// this is going to be our failed non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "8830c599-04cc-4b28-9b75-173355d49ab5",
						Address:  "198.18.0.5:8300",
					},
				},
			},
			expectedServers: CategorizedServers{
				StaleNonVoters: RaftServerEligibility{
					"db877f23-3e0a-4107-8ed8-bd7c3d710944": &VoterEligibility{
						currentVoter:   false,
						potentialVoter: false,
					},
				},
				StaleVoters: RaftServerEligibility{},
				FailedNonVoters: RaftServerEligibility{
					"8830c599-04cc-4b28-9b75-173355d49ab5": &VoterEligibility{
						currentVoter:   false,
						potentialVoter: true,
					},
				},
				FailedVoters: RaftServerEligibility{
					"a227f9a9-f55e-4321-b959-5afdcc63c6d3": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
				},
				HealthyNonVoters: RaftServerEligibility{},
				HealthyVoters: RaftServerEligibility{
					"51b2d56e-816e-409a-8b8e-afef2cf49661": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
					"51fb4248-be6a-43e5-b47f-c089818e2012": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
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
				"8830c599-04cc-4b28-9b75-173355d49ab5": {
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab5",
					Name:       "node5",
					Address:    "198.18.0.5:8300",
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
						}, State: RaftVoter,
					},
					// Stale non-voter
					"db877f23-3e0a-4107-8ed8-bd7c3d710944": {
						Server: Server{
							ID:         "db877f23-3e0a-4107-8ed8-bd7c3d710944",
							Name:       "node4",
							Address:    "198.18.0.4:8300",
							NodeStatus: NodeLeft,
							NodeType:   NodeVoter,
						}, State: RaftNonVoter,
					},
					// Failed non-voter
					"8830c599-04cc-4b28-9b75-173355d49ab5": {
						Server: Server{
							ID:         "8830c599-04cc-4b28-9b75-173355d49ab5",
							Name:       "node5",
							Address:    "198.18.0.5:8300",
							NodeStatus: NodeFailed,
							NodeType:   NodeVoter,
						}, State: RaftNonVoter,
					},
				},
			},
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {
				// Stale non-voter
				mraft.On("RemoveServer",
					raft.ServerID("db877f23-3e0a-4107-8ed8-bd7c3d710944"),
					uint64(0),
					time.Duration(0),
				).Return(&raftIndexFuture{}).Once()
				// Failed non-voter
				mapp.On("RemoveFailedServer", &Server{
					ID:         "8830c599-04cc-4b28-9b75-173355d49ab5",
					Name:       "node5",
					Address:    "198.18.0.5:8300",
					NodeStatus: NodeFailed,
					NodeType:   NodeVoter,
				}).Once()
			},
		},
		"stale-non-voters": {
			// 2 working nodes and 2 stale servers - shouldn't remove anything
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
					// this is going to be our stale non-voter
					{
						Suffrage: raft.Nonvoter,
						ID:       "3857f1d4-5c23-4016-9078-fee502c0d1b4",
						Address:  "198.18.0.4:8300",
					},
					// this is going to be our stale non-voter (2)
					{
						Suffrage: raft.Nonvoter,
						ID:       "db877f23-3e0a-4107-8ed8-bd7c3d710945",
						Address:  "198.18.0.5:8300",
					},
				},
			},
			expectedServers: CategorizedServers{
				StaleNonVoters: RaftServerEligibility{
					"3857f1d4-5c23-4016-9078-fee502c0d1b4": &VoterEligibility{
						currentVoter:   false,
						potentialVoter: false,
					},
					"db877f23-3e0a-4107-8ed8-bd7c3d710945": {
						currentVoter:   false,
						potentialVoter: false,
					},
				},
				StaleVoters:      RaftServerEligibility{},
				FailedNonVoters:  RaftServerEligibility{},
				FailedVoters:     RaftServerEligibility{},
				HealthyNonVoters: RaftServerEligibility{},
				HealthyVoters: RaftServerEligibility{
					"51b2d56e-816e-409a-8b8e-afef2cf49661": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
					},
					"51fb4248-be6a-43e5-b47f-c089818e2012": &VoterEligibility{
						currentVoter:   true,
						potentialVoter: true,
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
			setupExpectations: func(mraft *MockRaft, mapp *MockApplicationIntegration) {},
		},
	}

	for name, tcase := range cases {
		name := name
		tcase := tcase
		t.Run(name, func(t *testing.T) {
			conf := &Config{
				CleanupDeadServers:      true,
				MinQuorum:               3,
				ServerStabilizationTime: time.Hour * 6,
			}
			mpromoter := NewMockPromoter(t)
			failedServers := tcase.expectedServers.convertToFailedServers(&tcase.state)
			mpromoter.On("FilterFailedServerRemovals", conf, &tcase.state, mock.AnythingOfType("*autopilot.FailedServers")).Return(failedServers).Once()
			mpromoter.On("PotentialVoterPredicate", NodeVoter).Return(true)
			mapp := NewMockApplicationIntegration(t)
			mapp.On("AutopilotConfig").Return(conf)
			mapp.On("KnownServers").Return(tcase.knownServers)

			mraft := NewMockRaft(t)

			mraft.On("GetConfiguration").Return(&raftConfigFuture{config: tcase.raftConfig}).Once()

			a := &Autopilot{
				logger:                hclog.NewNullLogger(),
				raft:                  mraft,
				delegate:              mapp,
				state:                 &tcase.state,
				promoter:              mpromoter,
				reconciliationEnabled: true,
				time:                  &runtimeTimeProvider{},
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
