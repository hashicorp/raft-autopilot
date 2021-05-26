package autopilot

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func loadTestData(t *testing.T, name string) string {
	t.Helper()

	fpath := filepath.Join("testdata", name)

	data, err := ioutil.ReadFile(fpath)
	require.NoError(t, err)

	return string(data)
}

func loadGolden(t *testing.T, name string, got string, shouldUpdate bool) string {
	t.Helper()

	fname := name + ".golden"

	golden := filepath.Join("testdata", fname)
	if shouldUpdate && got != "" {
		err := ioutil.WriteFile(golden, []byte(got), 0644)
		require.NoError(t, err)
	}

	return loadTestData(t, fname)
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	return loadGolden(t, name, "", false)
}

func golden(t *testing.T, name, got string) string {
	t.Helper()
	return loadGolden(t, name, got, *update)
}

func TestAliveServers(t *testing.T) {
	input := map[raft.ServerID]*Server{
		"d27d2c33-b6ee-46ab-a2a5-54aa17a7d0a0": {
			ID:         "d27d2c33-b6ee-46ab-a2a5-54aa17a7d0a0",
			NodeStatus: NodeAlive,
		},
		"6e7c3c5b-6b8c-446c-9ac8-92b56966f332": {
			ID:         "6e7c3c5b-6b8c-446c-9ac8-92b56966f332",
			NodeStatus: NodeFailed,
		},
		"f205e59e-e2dc-4abe-8c68-2859fc198089": {
			ID:         "f205e59e-e2dc-4abe-8c68-2859fc198089",
			NodeStatus: NodeLeft,
		},
	}
	expected := map[raft.ServerID]*Server{
		"d27d2c33-b6ee-46ab-a2a5-54aa17a7d0a0": {
			ID:         "d27d2c33-b6ee-46ab-a2a5-54aa17a7d0a0",
			NodeStatus: NodeAlive,
		},
		"6e7c3c5b-6b8c-446c-9ac8-92b56966f332": {
			ID:         "6e7c3c5b-6b8c-446c-9ac8-92b56966f332",
			NodeStatus: NodeFailed,
		},
	}

	actual := aliveServers(input)
	require.Equal(t, expected, actual)
}

func TestGatherNextStateInputsLeaderFromDelegate(t *testing.T) {
	mtime := NewMockTimeProvider(t)
	mraft := NewMockRaft(t)
	mdel := NewMockApplicationIntegration(t)

	ap := New(mraft, mdel, withTimeProvider(mtime))
	firstStateTime := time.Date(2020, 11, 2, 12, 0, 0, 0, time.UTC)
	ap.state = &State{Healthy: false, firstStateTime: firstStateTime}

	now := time.Date(2020, 11, 02, 12, 0, 0, 5000, time.UTC)
	mtime.On("Now").Return(now).Once()

	conf := &Config{
		CleanupDeadServers:      true,
		LastContactThreshold:    200 * time.Millisecond,
		MaxTrailingLogs:         200,
		MinQuorum:               3,
		ServerStabilizationTime: 10 * time.Second,
	}

	servers := map[raft.ServerID]*Server{
		"7875975d-d54b-49c1-a400-9fefcc706c67": {
			ID:          "7875975d-d54b-49c1-a400-9fefcc706c67",
			Name:        "node1",
			Address:     "198.18.0.1:8300",
			NodeStatus:  NodeAlive,
			Version:     "1.9.0",
			RaftVersion: 3,
			IsLeader:    true,
		},
		"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": {
			ID:          "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1",
			Name:        "node2",
			Address:     "198.18.0.2:8300",
			NodeStatus:  NodeAlive,
			Version:     "1.9.0",
			RaftVersion: 3,
		},
		"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": {
			ID:          "e72eb8da-604d-47cd-bd7f-69ec120ea2b7",
			Name:        "node3",
			Address:     "198.18.0.3:8300",
			NodeStatus:  NodeAlive,
			Version:     "1.9.0",
			RaftVersion: 3,
		},
	}
	var lastIndex uint64 = 1024
	var lastTerm uint64 = 3

	serverStats := map[raft.ServerID]*ServerStats{
		"7875975d-d54b-49c1-a400-9fefcc706c67": {
			LastTerm:  lastTerm,
			LastIndex: lastIndex,
		},
		"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": {
			LastContact: 10 * time.Millisecond,
			LastTerm:    lastTerm,
			LastIndex:   1000,
		},
		"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": {
			LastContact: 15 * time.Millisecond,
			LastTerm:    lastTerm,
			LastIndex:   999,
		},
	}

	var leaderID raft.ServerID = "7875975d-d54b-49c1-a400-9fefcc706c67"

	mdel.On("AutopilotConfig").Return(conf).Once()
	mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
	mdel.On("KnownServers").Return(servers).Once()
	mraft.On("LastIndex").Return(lastIndex).Once()
	mraft.On("Stats").Return(map[string]string{"last_log_term": "3"}).Once()
	mdel.On("FetchServerStats", mock.Anything, servers).Return(serverStats).Once()

	expected := &nextStateInputs{
		Now:            now,
		FirstStateTime: firstStateTime,
		Config:         conf,
		RaftConfig:     &test3VoterRaftConfiguration,
		KnownServers:   servers,
		LatestIndex:    lastIndex,
		LastTerm:       lastTerm,
		FetchedStats:   serverStats,
		LeaderID:       leaderID,
	}

	actual, err := ap.gatherNextStateInputs(context.Background())
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestGatherNextStateInputs(t *testing.T) {
	now := time.Date(2020, 11, 02, 12, 0, 0, 5000, time.UTC)

	type testCase struct {
		state        *State
		expectedTime time.Time
	}

	cases := map[string]testCase{
		"nil": {
			state:        nil,
			expectedTime: now,
		},
		"zero-time": {
			state:        &State{},
			expectedTime: now,
		},
		"real-time": {
			state:        &State{firstStateTime: time.Date(2020, 11, 2, 12, 0, 0, 0, time.UTC)},
			expectedTime: time.Date(2020, 11, 2, 12, 0, 0, 0, time.UTC),
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			mtime := NewMockTimeProvider(t)
			mraft := NewMockRaft(t)
			mdel := NewMockApplicationIntegration(t)

			ap := New(mraft, mdel, withTimeProvider(mtime))
			ap.state = tcase.state

			mtime.On("Now").Return(now).Once()

			conf := &Config{
				CleanupDeadServers:      true,
				LastContactThreshold:    200 * time.Millisecond,
				MaxTrailingLogs:         200,
				MinQuorum:               3,
				ServerStabilizationTime: 10 * time.Second,
			}

			servers := map[raft.ServerID]*Server{
				"7875975d-d54b-49c1-a400-9fefcc706c67": {
					ID:          "7875975d-d54b-49c1-a400-9fefcc706c67",
					Name:        "node1",
					Address:     "198.18.0.1:8300",
					NodeStatus:  NodeAlive,
					Version:     "1.9.0",
					RaftVersion: 3,
				},
				"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": {
					ID:          "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1",
					Name:        "node2",
					Address:     "198.18.0.2:8300",
					NodeStatus:  NodeAlive,
					Version:     "1.9.0",
					RaftVersion: 3,
				},
				"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": {
					ID:          "e72eb8da-604d-47cd-bd7f-69ec120ea2b7",
					Name:        "node3",
					Address:     "198.18.0.3:8300",
					NodeStatus:  NodeAlive,
					Version:     "1.9.0",
					RaftVersion: 3,
				},
			}
			var lastIndex uint64 = 1024
			var lastTerm uint64 = 3

			serverStats := map[raft.ServerID]*ServerStats{
				"7875975d-d54b-49c1-a400-9fefcc706c67": {
					LastTerm:  lastTerm,
					LastIndex: lastIndex,
				},
				"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": {
					LastContact: 10 * time.Millisecond,
					LastTerm:    lastTerm,
					LastIndex:   1000,
				},
				"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": {
					LastContact: 15 * time.Millisecond,
					LastTerm:    lastTerm,
					LastIndex:   999,
				},
			}

			var leaderID raft.ServerID = "7875975d-d54b-49c1-a400-9fefcc706c67"

			mdel.On("AutopilotConfig").Return(conf).Once()
			mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
			mdel.On("KnownServers").Return(servers).Once()
			mraft.On("LastIndex").Return(lastIndex).Once()
			mraft.On("Stats").Return(map[string]string{"last_log_term": "3"}).Once()
			mdel.On("FetchServerStats", mock.Anything, servers).Return(serverStats).Once()
			mraft.On("Leader").Return(raft.ServerAddress("198.18.0.1:8300")).Once()

			expected := &nextStateInputs{
				Now:            now,
				FirstStateTime: tcase.expectedTime,
				Config:         conf,
				RaftConfig:     &test3VoterRaftConfiguration,
				KnownServers:   servers,
				LatestIndex:    lastIndex,
				LastTerm:       lastTerm,
				FetchedStats:   serverStats,
				LeaderID:       leaderID,
			}

			actual, err := ap.gatherNextStateInputs(context.Background())
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func TestNextStateWithInputs(t *testing.T) {
	// * get next servers
	//   * for each server
	//      * build basic server from inputs
	//      * have promoter build ext server
	// * count voters, healthy voters and record a leader
	// * calculate the failure tolerance and overall health
	// * have promoter build ext state
	// * have the promoter calculate the node types and set them
	// * sort the voters

	type testCase struct {
		setupPromoter func(*testing.T, *MockPromoter)
		setupState    func(*testing.T, *State)
	}

	cases := map[string]testCase{
		"typical": {
			setupPromoter: func(t *testing.T, m *MockPromoter) {
				t.Helper()
				m.On("GetServerExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.ServerState")).Return(nil).Times(3)
				m.On("GetStateExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(nil).Once()
				m.On("GetNodeTypes", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(map[raft.ServerID]NodeType{
					"7875975d-d54b-49c1-a400-9fefcc706c67": NodeVoter,
					"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": NodeVoter,
					"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": NodeVoter,
				}).Once()
			},
		},
		"one-failed": {
			setupPromoter: func(t *testing.T, m *MockPromoter) {
				t.Helper()
				m.On("GetServerExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.ServerState")).Return(nil).Times(3)
				m.On("GetStateExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(nil).Once()
				m.On("GetNodeTypes", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(map[raft.ServerID]NodeType{
					"7875975d-d54b-49c1-a400-9fefcc706c67": NodeVoter,
					"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": NodeVoter,
					"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": NodeVoter,
				}).Once()
			},
		},
		"non-voter": {
			setupPromoter: func(t *testing.T, m *MockPromoter) {
				t.Helper()
				m.On("GetServerExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.ServerState")).Return(nil).Times(3)
				m.On("GetStateExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(nil).Once()
				m.On("GetNodeTypes", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(map[raft.ServerID]NodeType{
					"7875975d-d54b-49c1-a400-9fefcc706c67": NodeVoter,
					"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": NodeVoter,
					"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": NodeVoter,
				}).Once()
			},
		},
		"staging": {
			setupPromoter: func(t *testing.T, m *MockPromoter) {
				t.Helper()
				m.On("GetServerExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.ServerState")).Return(nil).Times(3)
				m.On("GetStateExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(nil).Once()
				m.On("GetNodeTypes", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(map[raft.ServerID]NodeType{
					"7875975d-d54b-49c1-a400-9fefcc706c67": NodeVoter,
					"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": NodeVoter,
					"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": NodeVoter,
				}).Once()
			},
		},
		"state-overrides": {
			setupPromoter: func(t *testing.T, m *MockPromoter) {
				t.Helper()
				m.On("GetServerExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.ServerState")).Return(nil).Times(3)
				m.On("GetStateExt", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(nil).Once()
				m.On("GetNodeTypes", mock.AnythingOfType("*autopilot.Config"), mock.AnythingOfType("*autopilot.State")).Return(map[raft.ServerID]NodeType{
					"7875975d-d54b-49c1-a400-9fefcc706c67": NodeVoter,
					"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": NodeVoter,
					"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": NodeVoter,
				}).Once()
			},
			setupState: func(_ *testing.T, state *State) {
				state.Servers = map[raft.ServerID]*ServerState{
					"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": {
						Server: Server{},
						State:  "voter",
						Stats: ServerStats{
							LastContact: 15000000,
							LastIndex:   999,
							LastTerm:    3,
						},
						Health: ServerHealth{
							StableSince: time.Date(2020, 11, 2, 15, 0, 0, 0, time.UTC),
						},
					},
				}
			},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			mraft := NewMockRaft(t)
			mdel := NewMockApplicationIntegration(t)
			mprom := NewMockPromoter(t)

			tcase.setupPromoter(t, mprom)

			var inputs nextStateInputs

			inputPath := filepath.Join(name, "inputs.json")
			statePath := filepath.Join(name, "state.json")

			require.NoError(t, json.Unmarshal([]byte(loadTestData(t, inputPath)), &inputs))

			// we have to create this to use the promoter
			ap := New(mraft, mdel, WithPromoter(mprom))
			if tcase.setupState != nil {
				tcase.setupState(t, ap.state)
			}

			state := ap.nextStateWithInputs(&inputs)
			actualBytes, err := json.MarshalIndent(state, "", "   ")
			require.NoError(t, err)
			actual := string(actualBytes)

			expected := golden(t, statePath, actual)

			require.JSONEq(t, expected, actual)
		})
	}
}
